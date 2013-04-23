// Copyright 2012 Lightpoke. All rights reserved.
// This source code is subject to the terms and
// conditions defined in the "License.txt" file.

// Organics provides two-way communication between an browser and web server.
package organics

import (
	"bytes"
	"code.google.com/p/go.net/websocket"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"reflect"
	"runtime/debug"
	"strings"
	"sync"
	"time"
)

var (
	logger, debugLogger *log.Logger
)

// SetDebugOutput specifies an io.Writer which Organics will write debug information to.
func SetDebugOutput(w io.Writer) {
	logger = log.New(w, "organics ", log.Ltime)
}

func init() {
	connect := 0
	Connect = &connect

	if logger == nil {
		logger = log.New(ioutil.Discard, "organics ", 0)
	}

	debugLogger = log.New(os.Stdout, "organics ", log.Ltime)
}

func validRequestType(requestType string) bool {
	switch requestType {
	case rtWebSocketEstablishConnection:
		return true
	case rtLongPollEstablishConnection:
		return true
	case rtLongPoll:
		return true
	case rtMessage:
		return true
	}
	return false
}

func interfaceToValueSlice(s []interface{}) []reflect.Value {
	valueArgs := make([]reflect.Value, len(s))
	for i, value := range s {
		valueArgs[i] = reflect.ValueOf(value)
	}
	return valueArgs
}

// Server is an Organics HTTP and WebSocket server, it fulfills the net/http.Handler interface.
type Server struct {
	access sync.RWMutex

	webSocketHandler websocket.Handler
	sessionProvider  SessionProvider

	origins                       map[string]bool
	requestHandlers               map[interface{}]interface{}
	maxBufferSize, sessionKeySize int64
	pingRate, pingTimeout         time.Duration
	connections                   []*Connection
}

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

// Connections returns all connections this Server currently has.
func (s *Server) Connections() []*Connection {
	s.access.RLock()
	defer s.access.RUnlock()

	conns := make([]*Connection, len(s.connections))
	for i, c := range s.connections {
		conns[i] = c
	}
	return conns
}

// Handle defines that when an request with the specified requestName comes in, that the
// requestHandler function will handle it.
//
// The requestName parameter may be of any valid encoding/json.Marshal() type.
//
// The requestHandler parameter must be an function, with the type specified below, where T is any
// encoding/json.Marshal() valid value
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
	var connectionType *Connection
	if connectionParam != reflect.TypeOf(connectionType) {
		panic("requestHandler parameter type incorrect! Last parameter must be *organics.Connection")
	}

	if requestHandler == nil {
		delete(s.requestHandlers, requestName)
	} else {
		s.requestHandlers[requestName] = requestHandler
	}
}

func (s *Server) getHandler(requestName interface{}) interface{} {
	s.access.RLock()
	defer s.access.RUnlock()
	return s.requestHandlers[requestName]
}

func (s *Server) doConnectHandler(connection *Connection) {
	logger.Println("Connected", connection)

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

			fmt.Fprintf(buf, "func(*Connection)")
			fmt.Fprintf(buf, "\nFound type:\n\t")
			fmt.Fprintf(buf, "%s\n\n", fn.Type().String())
			fmt.Fprintf(buf, "%s\n\n", msg)
			fmt.Fprintf(buf, "%s", string(debug.Stack()))
			panic(string(buf.Bytes()))
		}
	}()
	fn.Call([]reflect.Value{reflect.ValueOf(connection)})
}

// Provider returns the session provider that is in use by this server, as it was passed in
// originally via organics.NewServer().
func (s *Server) Provider() SessionProvider {
	s.access.RLock()
	defer s.access.RUnlock()

	return s.sessionProvider
}

// SetMaxBufferSize sets the maximum size in bytes that the buffer which stores an single request
// may be.
//
// If an single JSON request exceeds this size, then the message will be refused, and the session
// killed.
//
// Default: 1 * 1024 * 1024 (1MB)
func (s *Server) SetMaxBufferSize(size int64) {
	s.access.Lock()
	defer s.access.Unlock()

	s.maxBufferSize = size
}

// MaxBufferSize returns the maximum buffer size of this Server.
//
// See Server.SetMaxBufferSize() for more information.
func (s *Server) MaxBufferSize() int64 {
	s.access.RLock()
	defer s.access.RUnlock()

	return s.maxBufferSize
}

// SetSessionKeySize specifies the number of cryptographically random bytes which will be used
// before sha256 hash and base64 encoding, as the per-user random session key identifier.
//
// Default is 64
func (s *Server) SetSessionKeySize(size int64) {
	s.access.Lock()
	defer s.access.Unlock()

	s.sessionKeySize = size
}

// SessionKeySize returns the size of session keys.
//
// See Server.SetSessionKeySize() for more information.
func (s *Server) SessionKeySize() int64 {
	s.access.RLock()
	defer s.access.RUnlock()

	return s.sessionKeySize
}

// SetPingTimeout specifies an duration which will be used to determine if an client is still
// considered connected.
//
// If an client leaves open it's long-polling POST request, then after PingRate() duration, the
// server will ask the client to respond ASAP, the client will then have PingTimeout() duration to
// respond, or else it will be considered disconnected, and the connection will be killed.
//
// This fixes an particular issue of leaving connection objects open forever, as web browsers are
// never required to close an HTTP connection (althought most do), and some proxies might leave an
// connection open perminantly, causing the servers memory to fill with dead connections, and thus
// an crash occuring.
//
// Default: 30 * time.Second (30 seconds)
func (s *Server) SetPingTimeout(t time.Duration) {
	s.access.Lock()
	defer s.access.Unlock()

	s.pingTimeout = t
}

// PingTimeout returns the ping timeout.
//
// See Server.SetPingTimeout() for more information.
func (s *Server) PingTimeout() time.Duration {
	s.access.RLock()
	defer s.access.RUnlock()

	return s.pingTimeout
}

// SetPingRate specifies the interval duration at which the server should request long-polling
// clients to verify they are still connected and active.
//
// If an client leaves open it's long-polling POST request, then after PingRate() duration, the
// server will ask the client to respond ASAP, the client will then have PingTimeout() duration to
// respond, or else it will be considered disconnected, and the connection will be killed.
//
// This fixes an particular issue of leaving connection objects open forever, as web browsers are
// never required to close an HTTP connection (althought most do), and some proxies might leave an
// connection open perminantly, causing the servers memory to fill with dead connections, and thus
// an crash occuring.
//
// Default: 5 * time.Minute (5 minutes)
func (s *Server) SetPingRate(t time.Duration) {
	s.access.Lock()
	defer s.access.Unlock()

	s.pingRate = t
}

// PingTimeout returns the ping rate of this Server.
//
// See Server.SetPingRate() for more information.
func (s *Server) PingRate() time.Duration {
	s.access.RLock()
	defer s.access.RUnlock()

	return s.pingRate
}

func (s *Server) ensureDataChangedRoutineExists(session *Session, sessionKey string) {
	// Ensure an goroutine exists to notify the session provider of session data changing, so that
	// it can save the session.
	if !session.hasDataChangedRoutine {
		session.hasDataChangedRoutine = true
		go func() {
			deathNotify := session.DeathNotify()
			for {
				select {
				case <-deathNotify:
					return

				case <-session.Store.dataChangedNotify:
					sp := s.Provider()
					if sp == nil {
						panic("No session provider is installed on the server")
					}
					err := sp.Store(sessionKey, session)
					if err != nil {
						log.Println("Session provider failed to store session:", err)
						return
					}
				}
			}
		}()
	}
}

func (s *Server) getSession(w http.ResponseWriter, req *http.Request) *Session {
	// Get the session provider, panic if there is none yet.
	sp := s.Provider()
	if sp == nil {
		panic("Server has no session provider set.")
	}

	var session *Session

	// Did they send us their existing session key?
	cookie, err := req.Cookie("organics-session")

	if err != http.ErrNoCookie {
		// They think they have an session already, this will either return an *Session or will
		// return nil, so this works in our case here.
		session = sp.Get(cookie.Value)
	}
	return session
}

func (s *Server) ensureSessionExists(w http.ResponseWriter, req *http.Request) (*Session, bool) {
	// They just connected. At this point it is important to realize that HTTP protocol is
	// not an connection based protocol, although most browsers do so today, HTTP clients are
	// never required to keep HTTP connections open, so we cannot assume that.
	//
	// An cookie is stored which is an unique identifier of their session object related to
	// their rtEstablishConnection request. The cookie is cryptographically random, and
	// therefor cannot be simply guessed or burteforced easily.
	//
	// Multiple connection objects can point to the same session object.
	//   An good example of this is gmail, if you log into gmail in one browser tab, then open
	//   another, and visit gmail again, you are already logged in. In this scenario each
	//   browser tab is considered and unique connection, each of which hold reference to the
	//   common session used between them.
	//
	// There is no way to know that two connection objects created by an rtEstablishConnection
	// request belong to an single session, so the way that we manage this is by having the
	// browser store an session cookie. The session cookie is secure and tells the server where
	// to look for their session object, assuming it is there, we can add another connection
	// object to the session based off the fact that the request type is rtEstablishConnection.

	// Get the session provider, panic if there is none yet.
	sp := s.Provider()
	if sp == nil {
		panic("Server has no session provider set.")
	}

	session := s.getSession(w, req)

	// If we need to give them an new session, let's do that.
	if session == nil || session.Dead() {
		// Generate an random session key
		sessionKey, err := s.generateSessionKey()
		if err != nil {
			// This should never really happen.
			logger.Println("Failed to generate session key:", err)
			w.WriteHeader(http.StatusInternalServerError)
			req.Close = true
			return nil, false
		}

		// Create the cookie
		cookie := new(http.Cookie)
		cookie.HttpOnly = true
		cookie.Name = "organics-session"
		cookie.Value = sessionKey

		// Create an session
		session = NewSession()

		// Store it
		err = sp.Store(sessionKey, session)
		if err != nil {
			log.Println("Session provider failed to store session:", err)
			w.WriteHeader(http.StatusInternalServerError)
			req.Close = true
			return nil, false
		}

		// Give it to their browser
		http.SetCookie(w, cookie)

		// Ensure their session has something to notify the session provider of changes in the
		// Session's Store's data.
		s.ensureDataChangedRoutineExists(session, sessionKey)
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

// ServeHTTP enables this Organics Server to implement the http.Handler interface.
func (s *Server) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// Respond to browser pre-flight requests, following the origin rules we have defined.
	if req.Method == "OPTIONS" {
		s.handlePreflightRequest(w, req)
		return
	}

	// Upgrade to WebSocket if they support it
	upgrades, ok := req.Header["Upgrade"]
	if ok {
		for _, v := range upgrades {
			// Safari sends "WebSocket" instead of "websocket" (all other browsers), so make sure
			// to check against the lower case word.
			if strings.ToLower(v) == "websocket" {
				s.webSocketHandler.ServeHTTP(w, req)
				return
			}
		}
	}

	// At this point we know they have no support for WebSocket, which means we should fall back to
	// long-polling.
	//
	// Defined in longpoll.go
	s.lpHandleRequest(w, req)
}

// SetOriginAccess specifies an origin string to allow access to or deny access to.
//
// If origin is "*", it is recognized as "all origins".
//
// Note: This applies to Cross-Origin Resource Sharing (CORS) and WebSocket Origin headers.
func (s *Server) SetOriginAccess(origin string, allowed bool) {
	s.access.Lock()
	defer s.access.Unlock()

	s.origins[origin] = allowed
}

// OriginAccess tells weather an origin string is currently set to allow access or deny it, based
// on an previous call to SetOriginAccess(), if there was no previous call to SetOriginAccess for
// this origin string, false is returned.
//
// Note: If an origin of "*" was previously set via SetOriginAccess("*", true), then this function
// will always return true.
//
// Note: This applies to Cross-Origin Resource Sharing (CORS) and WebSocket Origin headers.
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
func (s *Server) Origins() map[string]bool {
	s.access.RLock()
	defer s.access.RUnlock()

	originsCopy := make(map[string]bool)
	for k, v := range s.origins {
		originsCopy[k] = v
	}
	return originsCopy
}

// NewServer returns an new, and initialized Server.
func NewServer(sessionProvider SessionProvider) *Server {
	s := new(Server)
	s.origins = make(map[string]bool)
	s.requestHandlers = make(map[interface{}]interface{})
	s.sessionProvider = sessionProvider
	s.webSocketHandler = s.makeWebSocketHandler()

	s.maxBufferSize = 1 * 1024 * 1024 // Max message size: 1MB
	s.sessionKeySize = 64             // Bytes
	s.pingRate = 5 * time.Minute      // Ping response every 5 minutes
	s.pingTimeout = 30 * time.Second  // Consider connection dead if no ping response in 30 seconds

	//s.pingRate = 10 * time.Second
	//s.pingTimeout = 30 * time.Second

	return s
}
