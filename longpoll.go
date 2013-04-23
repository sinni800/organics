// Copyright 2012 Lightpoke. All rights reserved.
// This source code is subject to the terms and
// conditions defined in the "License.txt" file.

package organics

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"runtime/debug"
	"strconv"
)

func (c *Connection) lpWaitForDeath() {
	if !c.lpWaitingForDeath {
		c.lpWaitingForDeath = true
		go func() {
			// Wait untill someone wants this connection dead
			<-c.deathWantedNotify

			// Inform everyone it's dead
			c.deathNotify <- true

			// Wait for above to complete
			<-c.deathCompletedNotify
		}()
	}
}

func (s *Server) lpHandleLongPoll(w http.ResponseWriter, req *http.Request, session *Session, connection *Connection) {
	// This is an rtLongPoll request, we respond to it when we want to send something to this
	// connection.
	//
	// Additionally, we monitor for an CloseNotify event, in case they navigate away from the page,
	// thus closing their connection.

	// We know for an fact that this is an valid request, so we should reset their disconnection
	// timeout timer.
	connection.resetDisconnectTimer()

	// Enter an select which will determine our next action.
	select {
	case <-connection.performPing:
		// The Connection's disconnect timer says we should ping the client to determine if it
		// is active.
		//
		// We do that simply by responding with no data (an empty response = an ping).
		w.WriteHeader(http.StatusOK)
		return

	case <-connection.DeathNotify():
		// This connection is supposed to be dead, according to /someone/, so kill it.
		w.WriteHeader(http.StatusServiceUnavailable)
		req.Close = true
		return

	case <-connection.disconnectFromTimeout:
		// They must've exceeded the timout, so close their connection.
		connection.Kill()
		w.WriteHeader(http.StatusRequestTimeout)
		req.Close = true
		return

	case <-w.(http.CloseNotifier).CloseNotify():
		// They could close the connection at any time (refreshing, losing connectivity...)
		//
		// that (in most modern browsers) will trigger this.
		connection.Kill()
		return

	case msg := <-connection.messageChan:
		// Looks like we have something we would like to send to them, so we'll go ahead
		// and respond now.
		//
		// This ID is simply forwarded back to the client, if the ID is >= 0, then this
		// data is meant to be an response to an previous request from the client, so we
		// send this >= 0 number back to the client, so they know which request it belonged
		// to originally.
		//
		encoded, err := msg.jsonEncode()
		if err != nil {
			logger.Println(err)
			connection.Kill()
			w.WriteHeader(http.StatusBadRequest)
			req.Close = true
			return
		}
		w.Write(encoded)
		return
	}
}

func (s *Server) lpHandleMessage(w http.ResponseWriter, req *http.Request, session *Session, connection *Connection) {
	// This is an rtMessage request, it is either an request or response to one of our requests.

	// We need them to specify and content-length header, we'll check here to make sure they do
	// give it to us.
	contentLengthH, ok := req.Header["Content-Length"]
	if !ok || len(contentLengthH) != 1 {
		// They need to specify content length
		w.WriteHeader(http.StatusLengthRequired)
		req.Close = true
		return
	}
	// If they sent us an bad (non - int) content length header, we have no idea what they're
	// doing, but it's just as wrong as never providing the header at all.
	contentLength, err := strconv.Atoi(contentLengthH[0])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		req.Close = true
		return
	}

	// We have an maximum buffer size, we want to avoid people allocating an ton of memory and
	// crashing the server, so this limit protects against that specifically.
	//
	// If you need to send more JSON data than this; just raise your MaxBufferSize to whatever
	// it is that you'll be needing at max in an single request or response.
	if int64(contentLength) > s.MaxBufferSize() {
		session.Kill()
		w.WriteHeader(http.StatusRequestEntityTooLarge)
		req.Close = true
		return
	}

	// This seems to be an valid message thus far, so lets reset their disconnection timeout timer.
	connection.resetDisconnectTimer()

	// Make an slice to store the data in
	data := make([]byte, contentLength)

	// Try and read the content, if we encounter error inform them of an bad request and kill
	// the connection and session alike.
	n, err := io.ReadFull(req.Body, data)
	if n != len(data) || err != nil {
		w.WriteHeader(http.StatusBadRequest)
		req.Close = true
		return
	}

	decoded := new(message)
	err = decoded.jsonDecode(data)
	if err != nil {
		logger.Println(err)
		w.WriteHeader(http.StatusBadRequest)
		req.Close = true
		return
	}

	if !decoded.isRequest {
		// It's an response to one of our requests
		onComplete, ok := connection.requestCompleters[decoded.id]
		if !ok {
			// Should never happen.
			logger.Println("Invalid request response, id not valid, ignoring.")
			return
		}

		valueArgs := interfaceToValueSlice(decoded.args)
		fn := reflect.ValueOf(onComplete)

		defer func() {
			if r := recover(); r != nil {
				msg, ok := r.(string)
				if !ok {
					panic(r)
				}

				buf := new(bytes.Buffer)
				fmt.Fprintf(buf, "Request handler onComplete panic:\n\n")
				fmt.Fprintf(buf, "Expected type:\n")
				fmt.Fprintf(buf, "\t")

				fmt.Fprintf(buf, "func(")
				for n := 0; n < len(valueArgs); n++ {
					fmt.Fprintf(buf, valueArgs[n].Type().String())
					if n+1 < len(valueArgs) {
						fmt.Fprintf(buf, ", ")
					}
				}
				fmt.Fprintf(buf, ") ...")

				fmt.Fprintf(buf, "\nFound type:\n\t")
				fmt.Fprintf(buf, "%s\n\n", fn.Type().String())
				fmt.Fprintf(buf, "%s\n\n", msg)
				fmt.Fprintf(buf, "%s", string(debug.Stack()))
				debugLogger.Println(string(buf.Bytes()))
			}
		}()
		fn.Call(valueArgs)

	} else {
		// It's an request, so grab the request handler, and try to invoke it.
		handler := s.getHandler(decoded.requestName)
		if handler == nil {
			logger.Printf("No handler for message \"%s\"\n", decoded.requestName)
			return
		}
		fn := reflect.ValueOf(handler)

		valueArgs := interfaceToValueSlice(decoded.args)
		valueArgs = append(valueArgs, reflect.ValueOf(connection))

		defer func() {
			if r := recover(); r != nil {
				msg, ok := r.(string)
				if !ok {
					panic(r)
				}

				buf := new(bytes.Buffer)
				fmt.Fprintf(buf, "Request handler \"%s\" panic:\n\n", decoded.requestName)
				fmt.Fprintf(buf, "Expected type:\n")
				fmt.Fprintf(buf, "\t")

				fmt.Fprintf(buf, "func(")
				for n := 0; n < len(valueArgs); n++ {
					fmt.Fprintf(buf, valueArgs[n].Type().String())
					if n+1 < len(valueArgs) {
						fmt.Fprintf(buf, ", ")
					}
				}
				fmt.Fprintf(buf, ") ...")

				fmt.Fprintf(buf, "\nFound type:\n\t")
				fmt.Fprintf(buf, "%s\n\n", fn.Type().String())
				fmt.Fprintf(buf, "%s\n\n", msg)
				fmt.Fprintf(buf, "%s", string(debug.Stack()))
				debugLogger.Println(string(buf.Bytes()))
			}
		}()
		responseValues := fn.Call(valueArgs)

		responseArgs := make([]interface{}, len(responseValues))
		for i, v := range responseValues {
			responseArgs[i] = v.Interface()
		}

		responseMsg := newResponseMessage(decoded.id, responseArgs)

		// Maybe while trying to send this request to the long-polling request, they never actually
		// perform another long-polling request, so look for an timeout here.
		select {
		case <-connection.DeathNotify():
			// This connection is supposed to be dead, kill it.
			w.WriteHeader(http.StatusServiceUnavailable)
			req.Close = true
			return

		case <-connection.disconnectFromTimeout:
			// They must've exceeded the disconnection timout, so close their connection.
			connection.Kill()
			w.WriteHeader(http.StatusRequestTimeout)
			req.Close = true
			return

		case <-w.(http.CloseNotifier).CloseNotify():
			// They could close the connection at any time (refreshing, losing connectivity...)
			//
			// Most modern browsers, will do this.
			connection.Kill()
			return

		case connection.messageChan <- responseMsg:
			// We sent the data over the long-polling response channel, without hitting the
			// cases above! Success!
		}
	}

	// Finally, respond now that the request has completed.
	w.WriteHeader(http.StatusOK)
}

func (s *Server) lpHandleRequest(w http.ResponseWriter, req *http.Request) {
	// For long-polling we only use POST requests, for data going both ways (Because an request can
	// always modify server state, using GET would violate HTTP specification). Any other methods
	// will be denied by this server.
	if req.Method != "POST" {
		// HTTP Specification says we need to specify this header
		w.Header()["Allow"] = []string{"POST"}

		// Whatever they're trying to do at this point, they are wrong and/or not an Organics client.
		w.WriteHeader(http.StatusMethodNotAllowed)
		req.Close = true
		return
	}

	// We require that they have the X-Organics-Req header set, this is both to determine what
	// type of request it is, and to avoid confusion about what this server may provide.
	organicsReqH, ok := req.Header["X-Organics-Req"]
	if !ok || len(organicsReqH) != 1 {
		// They didn't provide the header, so they're in the wrong.
		logger.Println("bad request | X-Organics-Req header not present")
		w.WriteHeader(http.StatusBadRequest)
		req.Close = true
		return
	}

	// Check to see if they specified an invalid request type in the X-Organics-Req header, and if
	// they did, inform them of an bad request.
	organicsReq := organicsReqH[0]
	if !validRequestType(organicsReq) {
		// They gave us an wrong header value, it is bad.
		logger.Println("bad request | X-Organics-Req header invalid value")
		w.WriteHeader(http.StatusBadRequest)
		req.Close = true
		return
	}

	// Both Long Polling and WebSocket models will reach this code.
	//
	// WebSocket (see issue 2) will send rtEstablishConnection over normal POST in order to gain
	// proper cookies.
	//
	// Long Polling will send rtEstablishConnection in order to gain proper cookies, as well.
	//
	// 1) (WebSocket or LongPoll) Client sends rtEstablishConnection, server should create
	//    connection and session objects which will be used throughout lifetime of client. Request
	//    returns right away with http OK status code.
	//
	// 2) (LongPoll) Client, constantly, sends rtLongPoll, server waits (an potentially long time)
	//    untill the server wishes to respond to that client with an message, request responds once
	//    this happens.
	//
	// 3) (LongPoll) Client makes request, when it wants, sending rtMessage, server handles message
	//    and responds via previous (or future) rtLongPoll request.

	if organicsReq == rtWebSocketEstablishConnection || organicsReq == rtLongPollEstablishConnection {
		session, ok := s.ensureSessionExists(w, req)
		if !ok {
			return
		}

		// If this is an Long Polling establish connection request, the client needs to know what
		// connection object they are. We send their connection id as an string, and they send it
		// back through the X-Organics-Conn header.
		if organicsReq == rtLongPollEstablishConnection {

			// Double bonus: instead of senting an additional CSRF token, we make the connection
			// id the CSRF token.
			connectionId, err := s.generateSessionKey()
			if err != nil {
				// This should really never happen
				logger.Println("Failed to generate connection key identifier:", err)
				w.WriteHeader(http.StatusInternalServerError)
				req.Close = true
				return
			}

			// Create their new connection, using connectionId as the key
			connection := newConnection(req.RemoteAddr, session, connectionId, LongPolling)

			// Add their connection object, so we can find it later.
			session.addConnection(connectionId, connection)

			// Send it to them
			w.Write([]byte(connectionId))

			// We make an timer that will determine if they have been disconnected due to their connection
			// not responding.
			connection.disconnectTimer(s.PingTimeout(), s.PingRate())
			connection.lpWaitForDeath()

			// Notify the server's Handler functions that this connection has connected.
			s.doConnectHandler(connection)
			return
		}

		// Note: WebSocket's will add their connection to the session later, since WebSocket is an
		// connection-based protocol.

		// Lastly, at this point, this is all they wanted.
		w.WriteHeader(http.StatusOK)
		return
	}

	// At this point, this request is either an rtLongPoll or rtMessage, they should already have
	// an connection and session object, from an previous rtLongPollEstablishConnection.

	// We'll need to retrieve their session
	session := s.getSession(w, req)
	if session == nil {
		// For some reason, they have no session object. Either they got here *magically* by an
		// mistake, or their Organics client is totally messed up.
		//
		// Either way, they're in the wrong here.
		logger.Println("Long Polling request never sent establish connection message, ignored.")
		req.Close = true
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if session.Dead() {
		logger.Println("bad request | request for dead session attempted")
		w.WriteHeader(http.StatusBadRequest)
		req.Close = true
		return
	}

	// Now we need to figure out which connection they are. They send this information to us in an
	// header.
	organicsConnH, ok := req.Header["X-Organics-Conn"]
	if !ok || len(organicsConnH) != 1 {
		// They didn't provide the header, so they're in the wrong.
		logger.Println("bad request | X-Organics-Conn header not present")
		w.WriteHeader(http.StatusBadRequest)
		req.Close = true
		return
	}
	organicsConn := organicsConnH[0]

	// See if we recognize the connection they claim to be.
	connection := session.getConnection(organicsConn)
	if connection == nil {
		// Seems we did not find it, so they're in the wrong.
		logger.Println("bad request | X-Organics-Conn value invalid")
		w.WriteHeader(http.StatusBadRequest)
		req.Close = true
		return
	}
	if connection.Dead() {
		logger.Println("bad request | request for dead X-Organics-Conn attempted")
		w.WriteHeader(http.StatusBadRequest)
		req.Close = true
		return
	}

	// We've now got an validated connection and session object, and can continue through with the
	// rtLongPoll or rtMessage request.
	if organicsReq == rtLongPoll {
		s.lpHandleLongPoll(w, req, session, connection)
		return
	} else if organicsReq == rtMessage {
		s.lpHandleMessage(w, req, session, connection)
		return
	}
}
