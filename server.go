// Copyright 2012 Lightpoke. All rights reserved.
// This source code is subject to the terms and
// conditions defined in the "License.txt" file.

// Organics provides two-way communication between an browser and web server.
package organics

import (
	"bytes"
	"golang.org/x/net/websocket"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"runtime/debug"
	"strings"
	//"sync"
	"time"
)

// Server is an Organics HTTP and WebSocket server, it fulfills the Handler
// interface defined in the net/http package.
type Server struct {
	host
	
	webSocketServer *websocket.Server
	sessionProvider SessionProvider

	sessions                      map[interface{}]*Session
	origins                       map[string]bool
	sessionKeySize int64
	sessionTimeout                time.Duration
	connections                   []*ServerConnection
}

// Utility function to convert []interface{} into []reflect.Value
func interfaceToValueSlice(s []interface{}) []reflect.Value {
	valueArgs := make([]reflect.Value, len(s))
	for i, value := range s {
		valueArgs[i] = reflect.ValueOf(value)
	}
	return valueArgs
}

// Generates new session key using the servers session key size.
func (s *Server) generateSessionKey() (string, error) {
	key := make([]byte, s.SessionKeySize())
	n, err := io.ReadFull(rand.Reader, key)
	if n != len(key) || err != nil {
		return "", err
	}
	hash := sha256.New()
	n, err = hash.Write(key)
	if n != len(key) || err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(hash.Sum(nil)), nil
}


func (s *Server) doConnectHandler(connection *ServerConnection) {
	logger().Println("Connected", connection)

	s.access.Lock()
	s.connections = append(s.connections, connection)
	s.access.Unlock()
	go func() {
		<-connection.DeathNotify()
		s.access.Lock()
		for i, c := range s.connections {
			if c == connection {
				s.connections = append(s.connections[:i], s.connections[i+1:]...)
			}
		}
		s.access.Unlock()
	}()

	handler := s.getHandler(Connect)
	if handler == nil {
		return
	}
	fn := reflect.ValueOf(handler)

	defer func() {
		if r := recover(); r != nil {
			msg, ok := r.(string)
			if !ok {
				panic(r)
			}

			buf := new(bytes.Buffer)
			fmt.Fprintf(buf, "Connect handler panic:\n\n")
			fmt.Fprintf(buf, "Expected type:\n")
			fmt.Fprintf(buf, "\t")

			fmt.Fprintf(buf, "func(*ServerConnection)")
			fmt.Fprintf(buf, "\nFound type:\n\t")
			fmt.Fprintf(buf, "%s\n\n", fn.Type().String())
			fmt.Fprintf(buf, "%s\n\n", msg)
			fmt.Fprintf(buf, "%s", string(debug.Stack()))
			panic(string(buf.Bytes()))
		}
	}()
	fn.Call([]reflect.Value{reflect.ValueOf(connection)})
}

func (s *Server) cachedSession(key string) (session *Session, ok bool) {
	s.access.RLock()
	defer s.access.RUnlock()

	session, ok = s.sessions[key]
	return
}

func (s *Server) cacheSession(key string, session *Session) {
	s.access.Lock()
	defer s.access.Unlock()

	s.sessions[key] = session
}

func (s *Server) uncache(key string) {
	s.access.Lock()
	defer s.access.Unlock()

	delete(s.sessions, key)
}

func (s *Server) getSession(req *http.Request) *Session {
	// Get the session provider, panic if there is none yet.
	sp := s.Provider()
	if sp == nil {
		panic("Server has no session provider set.")
	}

	// We'll store the session in here if we find the cookie properly
	var session *Session

	// Did they send us their existing session key?
	cookie, err := req.Cookie("organics-session")
	if err != http.ErrNoCookie {
		// The client thinks they have an existing session. It could be in one
		// of two places then. In the memory map session cache (I.e. existing
		// session pointer), or the session provider might be aware of it's
		// data (in which case we need to make a new session object using the
		// data).
		var ok bool
		session, ok = s.cachedSession(cookie.Value)
		if !ok {
			// It's not in the cache of already-created session objects. See if
			// the session provider is aware of it's data, then.
			sessionStore := sp.Load(cookie.Value)
			if sessionStore != nil {
				// The session provider has the session data; now create an new
				// session object for it.
				session = newSession(cookie.Value, s)
				session.Store = sessionStore
				session.start()

				// Cache this new object for later.
				s.cacheSession(cookie.Value, session)
			}
		}
	}
	return session
}

func (s *Server) ensureSessionExists(req *http.Request, setCookie func(*http.Cookie)) (*Session, bool) {
	// They just connected. At this point it is important to realize that HTTP
	// protocol is not an connection based protocol, although most browsers do
	// so today, HTTP clients are never required to keep HTTP connections open,
	// so we cannot assume that.
	//
	// An cookie is stored which is an unique identifier of their session
	// object related to their rtEstablishConnection request. The cookie is
	// cryptographically random, and therefor cannot be simply guessed or
	// bruteforced easily.
	//
	// Multiple connection objects can point to the same session object.
	//   An good example of this is gmail, if you log into gmail in one browser
	//   tab, then open another, and visit gmail again, you are already logged
	//   in. In this scenario each browser tab is considered and unique
	//   connection, each of which hold reference to the common session used
	//   between them.
	//
	// There is no way to know that two connection objects created by an
	// rtEstablishConnection request belong to an single session, so the way
	// that we manage this is by having the browser store an session cookie.
	// The session cookie is secure and tells the server where to look for
	// their session object, assuming it is there, we can add another
	// connection object to the session based off the fact that the request
	// type is rtEstablishConnection.

	// Get the session provider, panic if there is none yet.
	sp := s.Provider()
	if sp == nil {
		panic("Server has no session provider set.")
	}

	session := s.getSession(req)

	// If we need to give them an new session, let's do that.
	if session == nil || session.Dead() {
		// Generate an random session key
		sessionKey, err := s.generateSessionKey()
		if err != nil {
			// This should never really happen.
			logger().Println("Failed to generate session key:", err)
			req.Close = true
			return nil, false
		}

		// Create the cookie
		cookie := new(http.Cookie)
		cookie.HttpOnly = true
		cookie.Name = "organics-session"
		cookie.Value = sessionKey

		// Create an session
		session = newSession(sessionKey, s)
		session.start()

		// Cache it now
		s.cacheSession(sessionKey, session)

		// Give it to their browser
		setCookie(cookie)
	}
	return session, true
}

func (s *Server) handlePreflightRequest(w http.ResponseWriter, req *http.Request) {
	allowOrigin := "null"

	allOriginsAllowed := s.OriginAccess("*")
	if allOriginsAllowed {
		allowOrigin = "*"

	} else {
		theirOrigin := w.Header()["Origin"]
		if len(theirOrigin) == 0 {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if s.OriginAccess(theirOrigin[0]) {
			allowOrigin = theirOrigin[0]
		}
	}

	w.Header()["Access-Control-Allow-Origin"] = []string{allowOrigin}
	w.Header()["Access-Control-Max-Age"] = []string{"600"}
	w.Header()["Access-Control-Allow-Methods"] = []string{"POST"}

	w.WriteHeader(http.StatusOK)
}

// ServeHTTP fulfills the Handler interface define in the http package.
func (s *Server) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// Respond to browser pre-flight requests, following the origin rules we
	// have defined.
	if req.Method == "OPTIONS" {
		s.handlePreflightRequest(w, req)
		return
	}

	// Upgrade to WebSocket if they support it
	upgrades, ok := req.Header["Upgrade"]
	if ok {
		for _, v := range upgrades {
			// Safari sends "WebSocket" instead of "websocket" (all other
			// browsers), so make sure to check against the lower case word.
			if strings.ToLower(v) == "websocket" {
				s.webSocketServer.ServeHTTP(w, req)
				return
			}
		}
	}

	// At this point we know they have no support for WebSocket, which means we
	// should fall back to long-polling.
	//
	// Defined in longpoll.go
	s.lpHandleRequest(w, req)
}

// Connections returns all connections this Server currently has.
func (s *Server) Connections() []*ServerConnection {
	s.access.RLock()
	defer s.access.RUnlock()

	conns := make([]*ServerConnection, len(s.connections))
	for i, c := range s.connections {
		conns[i] = c
	}
	return conns
}

// Handle defines that when an request with the specified requestName comes in,
// that the requestHandler function will be invoked in order to handle the
// request.
//
// The requestName parameter may be of any valid json.Marshal() type.
//
// The requestHandler parameter must be an function, with the type specified
// below, where T is any valid json.Marshal() type.
//
// func(T, T, ..., *Session) (T, T, ...)
func (s *Server) Handle(requestName, requestHandler interface{}) {
	s.access.Lock()
	defer s.access.Unlock()

	fn := reflect.ValueOf(requestHandler)
	if fn.Kind() != reflect.Func {
		panic("requestHandler parameter type incorrect! Must be function!")
	}

	fnType := fn.Type()
	connectionParam := fnType.In(fnType.NumIn() - 1)
	var connectionType *ServerConnection
	if connectionParam != reflect.TypeOf(connectionType) {
		panic("requestHandler parameter type incorrect! Last parameter must be *organics.ServerConnection")
	}

	if requestHandler == nil {
		delete(s.requestHandlers, requestName)
	} else {
		s.requestHandlers[requestName] = requestHandler
	}
}

// Provider returns the session provider that is in use by this server, as it
// was passed in originally via NewServer().
func (s *Server) Provider() SessionProvider {
	s.access.RLock()
	defer s.access.RUnlock()

	return s.sessionProvider
}

// SetSessionKeySize specifies the number of cryptographically random bytes
// which will be used before sha256 hash and base64 encoding, as the per-user
// random session key identifier.
//
// Default: 64
func (s *Server) SetSessionKeySize(size int64) {
	s.access.Lock()
	defer s.access.Unlock()

	s.sessionKeySize = size
}

// SessionKeySize returns the size of session keys.
//
// See SetSessionKeySize() for more information.
func (s *Server) SessionKeySize() int64 {
	s.access.RLock()
	defer s.access.RUnlock()

	return s.sessionKeySize
}

// SetSessionTimeout specifies the duration in which an session's data will be
// monitored for changes after the session is killed.
//
// Default (30 seconds): 30 * time.Second
func (s *Server) SetSessionTimeout(t time.Duration) {
	s.access.Lock()
	defer s.access.Unlock()

	s.sessionTimeout = t
}

// SessionTimeout returns the session timeout of this server.
//
// See SetSessionTimeout() for more information about this value.
func (s *Server) SessionTimeout() time.Duration {
	s.access.RLock()
	defer s.access.RUnlock()

	return s.sessionTimeout
}

// SetOriginAccess specifies an origin string to allow access to or deny access
// to.
//
// If origin is "*" it is recognized as 'all origins'.
//
// Note: This applies to Cross-Origin Resource Sharing (CORS) and WebSocket
// Origin headers.
func (s *Server) SetOriginAccess(origin string, allowed bool) {
	s.access.Lock()
	defer s.access.Unlock()

	if !allowed {
		delete(s.origins, origin)
	} else {
		s.origins[origin] = true
	}
}

// OriginAccess tells weather an origin string is currently set to allow access
// or deny it, based on an previous call to SetOriginAccess(), if there was no
// previous call to SetOriginAccess for this origin string, false is returned.
//
// Note: If an origin of "*" was previously set via SetOriginAccess("*", true),
// then this function will always return true.
//
// Note: This applies to Cross-Origin Resource Sharing (CORS) and WebSocket
// Origin headers.
func (s *Server) OriginAccess(origin string) bool {
	s.access.RLock()
	defer s.access.RUnlock()

	if allowed, ok := s.origins["*"]; allowed && ok {
		return true
	}

	allowed, ok := s.origins[origin]
	if !ok {
		return false
	}
	return allowed
}

// Origins returns an map of all origin's and their respective access values.
//
// The access value will always be true. (Origins with access denied simply do
// not exist in the map at all).
func (s *Server) Origins() map[string]bool {
	s.access.RLock()
	defer s.access.RUnlock()

	originsCopy := make(map[string]bool)
	for k, v := range s.origins {
		originsCopy[k] = v
	}
	return originsCopy
}

// Kill kills each connection currently known to the server. It is short-hand
// for the following code:
//
//  for _, c := range s.Connections() {
//      c.Kill()
//  }
//
func (s *Server) Kill() {
	for _, c := range s.Connections() {
		c.Kill()
	}
}

// NewServer returns an new, and initialized Server.
func NewServer(sessionProvider SessionProvider) *Server {
	s := new(Server)
	s.sessions = make(map[interface{}]*Session)
	s.origins = make(map[string]bool)
	s.requestHandlers = make(map[interface{}]interface{})
	s.sessionProvider = sessionProvider
	s.webSocketServer = s.makeWebSocketServer()

	// Max message size: 1MB
	s.maxBufferSize = 1 * 1024 * 1024

	// Size in bytes
	s.sessionKeySize = 64

	// Ping response every 5 minutes
	s.pingRate = 5 * time.Minute

	// Consider connection dead if no ping response in 30 seconds
	s.pingTimeout = 30 * time.Second

	// Stop saving session data 30 seconds after it's death.
	s.sessionTimeout = 30 * time.Second
	return s
}
