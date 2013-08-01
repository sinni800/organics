// Copyright 2012 Lightpoke. All rights reserved.
// This source code is subject to the terms and
// conditions defined in the "License.txt" file.

package organics

import (
	"bytes"
	"code.google.com/p/go.net/websocket"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"reflect"
	"runtime/debug"
)

// See https://code.google.com/p/organics/issues/detail?id=4 (issue 4)
func sendMessage(ws *websocket.Conn, msg string) error {
	w, err := ws.NewFrameWriter(websocket.TextFrame)
	if err != nil {
		return err
	}
	_, err = w.Write([]byte(msg))
	w.Close()
	return err
}

func receiveMessage(ws *websocket.Conn, limit int64) (string, error) {
again:
	frame, err := ws.NewFrameReader()
	if err != nil {
		return "", err
	}
	frame, err = ws.HandleFrame(frame)
	if err != nil {
		return "", err
	}
	if frame == nil {
		goto again
	}
	data, err := ioutil.ReadAll(&io.LimitedReader{frame, limit})
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (s *Server) webSocketWrite(ws *websocket.Conn, connection *Connection) {
	deathNotify := connection.DeathNotify()
	for {
		select {
		case <-deathNotify:
			return

		case <-connection.performPing:
			err := sendMessage(ws, "")
			if err != nil {
				logger().Println("Error writing", err, connection)
				connection.Kill()
				return
			}

		case msg := <-connection.messageChan:
			encoded, err := msg.jsonEncode()
			if err != nil {
				logger().Println(err)
				connection.Kill()
				return
			}
			err = sendMessage(ws, string(encoded))
			if err != nil {
				logger().Println("Error writing", err, connection)
				connection.Kill()
				return
			}
		}
	}
}

func (s *Server) webSocketHandleMessage(msg string, ws *websocket.Conn, connection *Connection) {
	// Any message means they're active, sense they sent it to us.
	connection.resetDisconnectTimer()

	// Messages of length zero, are ping responses (A.K.A. Pong), nothing more than that.
	if len(msg) == 0 {
		return
	}

	decoded := new(message)
	err := decoded.jsonDecode([]byte(msg))
	if err != nil {
		logger().Println(err)
		connection.Kill()
		return
	}

	if decoded.isRequest == false {
		// It's an response to one of our requests
		onComplete, ok := connection.requestCompleters[decoded.id]
		if !ok {
			// Should never happen.
			logger().Println("Invalid request response, id not valid, ignoring.")
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
				panic(string(buf.Bytes()))
			}
		}()
		fn.Call(valueArgs)

	} else {
		// It's an request, so grab the request handler, and try to invoke it.
		handler := s.getHandler(decoded.requestName)
		if handler == nil {
			logger().Printf("No handler for message \"%s\"\n", decoded.requestName)
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
				panic(string(buf.Bytes()))
			}
		}()
		responseValues := fn.Call(valueArgs)

		responseArgs := make([]interface{}, len(responseValues))
		for i, v := range responseValues {
			responseArgs[i] = v.Interface()
		}

		responseMsg := newResponseMessage(decoded.id, responseArgs)

		select {
		case <-connection.DeathNotify():
			return

		case connection.messageChan <- responseMsg:
			break
		}
	}
}

func (s *Server) webSocketWaitForDeath(ws *websocket.Conn, connection *Connection) {
	// Wait untill someone wants this connection dead
	select {
	case <-connection.deathWantedNotify:
		break

	case <-connection.disconnectFromTimeout:
		break
	}

	// Inform everyone it's dead
	connection.deathNotify <- true

	// Wait for above to complete
	<-connection.deathCompletedNotify

	// Close the WebSocket now
	ws.Close()
}

func (s *Server) handleWebSocket(ws *websocket.Conn) {
	origin := ws.Request().Header["Origin"]
	if len(origin) == 0 {
		logger().Println("WebSocket connection without origin header, dropped.")
		ws.Close()
		return
	}

	if !s.OriginAccess(origin[0]) {
		logger().Println("WebSocket connection from non-allowed origin, dropped.")
		if len(origin) <= 256 {
			logger().Printf("^ %q\n", origin[0])
		}
		ws.Close()
		return
	}

	// They should have an existing session object at this point.
	sp := s.Provider()
	if sp == nil {
		panic("No session provider is installed on the server")
	}

	session := s.getSession(ws.Request())
	if session == nil {
		// They don't have an session known to us, drop them.
		logger().Println("WebSocket with an invalid session, dropping.")
		ws.Close()
		return
	}

	// WebSocket just connected, so we need to store it with their Session
	//
	// In this case, we'll use the actualy websocket connection as the key, since WebSocket is an
	// connection-based protocol.
	connection := newConnection(ws.Request().RemoteAddr, session, ws, WebSocket)

	go s.webSocketWrite(ws, connection)
	go s.webSocketWaitForDeath(ws, connection)
	connection.disconnectTimer(s.PingTimeout(), s.PingRate())

	s.doConnectHandler(connection)

	for {
		msg, err := receiveMessage(ws, s.MaxBufferSize())
		if err != nil {
			if err != io.EOF {
				logger().Println("receiveMessage() failed:", err)
			}
			connection.Kill()
			break
		}

		defer func() {
			if e := recover(); e != nil {
				logger().Println(fmt.Sprint(e))
			}
		}()
		s.webSocketHandleMessage(msg, ws, connection)
	}
}

func (s *Server) handleWebSocketHandshake(config *websocket.Config, req *http.Request) error {
	var err error

	var origin string
	switch config.Version {
	case websocket.ProtocolVersionHybi13:
		origin = req.Header.Get("Origin")
	case websocket.ProtocolVersionHybi08:
		origin = req.Header.Get("Sec-Websocket-Origin")
	}

	if !s.OriginAccess(origin) {
		err = fmt.Errorf("WebSocket connection from disallowed origin %q, dropped.")
		logger().Println(err)
		return err
	}

	_, ok := s.ensureSessionExists(req, func(cookie *http.Cookie) {
		// Set cookie
		config.Header = make(http.Header)
		config.Header.Set("Set-Cookie", cookie.String())

		// Set Cookie header so that handleWebSocket above can see the updated
		// cookie.
		req.Header.Set("Cookie", cookie.String())
	})
	if !ok {
		err = errors.New(http.StatusText(http.StatusInternalServerError))
		logger().Println(err)
		return err
	}

	return nil
}

func (s *Server) makeWebSocketServer() *websocket.Server {
	return &websocket.Server{
		Handler:   s.handleWebSocket,
		Handshake: s.handleWebSocketHandshake,
	}
}
