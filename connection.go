// Copyright 2012 Lightpoke. All rights reserved.
// This source code is subject to the terms and
// conditions defined in the "License.txt" file.

package organics

import (
	"fmt"
	"reflect"
	"sync"
	"time"
)

// Method describes an single connection method.
type Method uint8

const (
	// Describes the long polling connection method.
	LongPolling Method = iota

	// Describes the web socket connection method.
	WebSocket
)

// String returns an string formatted version of the specified method, or an
// empty string if the method is invalid (unknown).
//
// For example:
//  WebSocket.String() would return "WebSocket"
//  LongPolling.String() would return "LongPolling"
//
func (m Method) String() string {
	switch m {
	case LongPolling:
		return "LongPolling"

	case WebSocket:
		return "WebSocket"
	}
	return ""
}

// Connection represents an single connection to an web browser, this
// connection will remain for as long as the user keeps the web page open
// through their browser.
type connection struct {
	
	access                                                   sync.RWMutex
	dead                                                     bool
	deathNotify, deathCompletedNotify, deathWantedNotify     chan bool
	deathNotifications                                       []chan bool
	address                                                  string
	method                                                   Method
	session                                                  *Session
	disconnectFromTimeout, disconnectTimerReset, performPing chan bool
	lpWaitingForDeath, hasDisconnectTimer                    bool

	messageChan       chan *message
	requestCurrentId  float64
	requestCompleters map[float64]interface{}
}

type ClientConnection struct {
	*connection
}

type ServerConnection struct {
	*Store
	*connection
	key                                                      interface{}
	session                                                  *Session
}

// Request makes an request to the other end of this Connection.
//
// If this connection is dead, or this Connection's Session is dead, this
// function is no-op.
//
// The first parameter to this function is the request name, this can be any
// value that the json.Marshal() function will accept.
//
// The second parameter(s) is any sequence of arguments to the request, as many
// as you wish, while noting these arguments must be accepted by the
// json.Marshal() function.
//
// The third (and optional) argument is an function that will be called when
// the request has completed (I.e. has been invoked on the other end) and it
// will be given the response data from the request, it shoul be of the type:
//
//  func(T, T, ...)
//
// Where T are whatever data types the function on the other end will return
// once invoked.
func (c *connection) Request(requestName interface{}, sequence ...interface{}) {
	if c.Dead() {
		return
	}

	if c.session.Dead() {
		return
	}

	var onComplete interface{}
	var hasOnComplete bool
	var args []interface{}

	c.access.Lock()
	id := c.requestCurrentId

	c.requestCurrentId += 1
	// Handle overflow, -1 is special id.
	if c.requestCurrentId == -1 {
		c.requestCurrentId += 1
	}
	c.access.Unlock()

	if len(sequence) > 0 {
		onComplete = sequence[len(sequence)-1]
		hasOnComplete = reflect.ValueOf(onComplete).Kind() == reflect.Func

		if hasOnComplete {
			args = sequence[:len(sequence)-1]
			c.requestCompleters[id] = onComplete
		} else {
			args = sequence
			id = -1 // Never send response to us, please.
		}
	}

	go func() {
		select {
		case c.messageChan <- newRequestMessage(id, requestName, args):
			// Sent message okay!
			return

		case <-c.DeathNotify():
			// Connection closed; can't send.
			return
		}
	}()
}

// String returns an string representation of this Connection.
//
// For security reasons the string will not contain the stores data, and will
// instead only contain the length of the store (I.e. how many objects it
// contains).
func (c *ServerConnection) String() string {
	return fmt.Sprintf("Connection(%s, Store.Len()=%v, Dead=%t, Method=%s)", c.Address(), c.Store.Len(), c.Dead(), c.Method().String())
}

func (c *ClientConnection) String() string {
	return fmt.Sprintf("Connection(%s, Dead=%t, Method=%s)", c.Address(), c.Dead(), c.Method().String())
}

// Address returns the client address of this connection, usually for logging purposes.
//
// Format: ip:port
func (c *connection) Address() string {
	// Note: no locking needed, never written to past creation time.
	return c.address
}

// Method returns one of the predefined constant methods which represents the
// method in use by this connection.
func (c *connection) Method() Method {
	// Note: no locking needed, never written to past creation time.
	return c.method
}

// Session returns the session object that is associated with this connection.
func (c *ServerConnection) Session() *Session {
	// Note: no locking needed, never written to past createion time.
	return c.session
}

// Dead tells weather this Connection is dead or not.
func (c *connection) Dead() bool {
	c.access.RLock()
	defer c.access.RUnlock()

	return c.dead
}

// Kill kills this connection as soon as possible, this function will block
// until the connection is dead.
//
// If this connection is already dead, this function is no-op.
func (c *connection) Kill() {
	if !c.Dead() {
		// Signal death
		c.deathWantedNotify <- true

		// Wait for completion
		<-c.deathCompletedNotify
	}
}

// DeathNotify returns an new channel on which true will be sent once this
// connection is killed.
//
// An connection is considered killed once it's Kill() method has been called.
//
// If this connection is dead, then this function returns nil.
func (c *connection) DeathNotify() chan bool {
	if c.Dead() {
		return nil
	}

	c.access.Lock()
	defer c.access.Unlock()

	ch := make(chan bool, 1)
	c.deathNotifications = append(c.deathNotifications, ch)
	return ch
}

func (c *ClientConnection) waitForDeath() {
	c.connection.waitForDeath()
	c.deathCompletedNotify <- true
}

func (c *ServerConnection) waitForDeath() {
	c.connection.waitForDeath()
	c.Session().removeConnection(c.key)
	c.deathCompletedNotify <- true
}

func (c *connection) waitForDeath() {
	<-c.deathNotify

	c.access.Lock()
	c.dead = true

	deathNotifications := make([]chan bool, len(c.deathNotifications))
	for i, ch := range c.deathNotifications {
		deathNotifications[i] = ch
	}
	c.deathNotifications = make([]chan bool, 0)
	c.access.Unlock()

	for _, ch := range deathNotifications {
		ch <- true
		close(ch)
	}

	logger().Println("DeathNotify():", c)
}

func (c *connection) disconnectTimer(timeout, rate time.Duration) {
	c.access.Lock()
	defer c.access.Unlock()

	if !c.hasDisconnectTimer {
		c.hasDisconnectTimer = true

		go func() {
			deathNotify := c.DeathNotify()
			for {
				select {
				case <-deathNotify:
					return

				case <-c.disconnectTimerReset:
					// At this point, there is no need to do anything for "resetting" the timer
					break

				case <-time.After(rate):
					c.performPing <- true
					select {
					case <-deathNotify:
						return

					case <-time.After(timeout):
						logger().Println("Ping timeout", c)
						c.disconnectFromTimeout <- true

					case <-c.disconnectTimerReset:
						break
					}
				}
			}
		}()
	}
}

func (c *connection) resetDisconnectTimer() {
	c.disconnectTimerReset <- true
}

func newServerConnection(address string, session *Session, key interface{}, method Method) *ServerConnection {
	c := new(ServerConnection)
	c.connection = newConnection(address, method)

	c.Store = NewStore()
	c.key = key // Used for removal from session via removeConnection()

	c.session = session
	session.addConnection(key, c)
	return c
}

func newClientConnection(address string, method Method) *ClientConnection {
	c := new(ClientConnection)
	c.connection = newConnection(address, method)
	return c
}

func newConnection(address string, method Method) *connection {
	c := new(connection)

	c.deathNotify = make(chan bool)
	c.deathWantedNotify = make(chan bool)
	c.deathCompletedNotify = make(chan bool)
	c.messageChan = make(chan *message)
	c.requestCompleters = make(map[float64]interface{})
	c.address = address
	c.method = method

	c.disconnectFromTimeout = make(chan bool, 1)
	c.disconnectTimerReset = make(chan bool, 1)
	c.performPing = make(chan bool, 1)
	go c.waitForDeath()

	return c
}
