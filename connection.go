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

type ConnectionMethod uint8

const (
	LongPolling ConnectionMethod = iota
	WebSocket
)

// String returns an string formatted version of the specified method, or an empty string if an
// invalid method string is passed in.
//
// For example: organics.MethodString(organics.MethodWebSocket) will return simply "WebSocket"
func MethodString(method ConnectionMethod) string {
	switch method {
	case LongPolling:
		return "LongPolling"

	case WebSocket:
		return "WebSocket"
	}
	return ""
}

// Connection represents an single connection to an web browser, this connection will remain for
// as long as the user keeps the web page open through their browser.
type Connection struct {
	Store

	key                                                      interface{}
	access                                                   sync.RWMutex
	dead                                                     bool
	deathNotify, deathCompletedNotify, deathWantedNotify     chan bool
	onDeathNotifications                                     []chan bool
	address                                                  string
	method                                                   ConnectionMethod
	session                                                  *Session
	disconnectFromTimeout, disconnectTimerReset, performPing chan bool
	lpWaitingForDeath, hasDisconnectTimer                    bool

	messageChan       chan *message
	requestCurrentId  float64
	requestCompleters map[float64]interface{}
}

// Request makes an request to the other end of this Connection.
//
// If this connection is dead, or this Connection's Session is dead, this function is no-op.
//
// The first parameter to this function is the request name, this can be any value that
// encoding/json.Marshal() will accept.
//
// The second parameter(s) is any sequence of arguments to the request, as many as you wish, while
// noting these arguments must be accepted by encoding/json.Marshal().
//
// The third and (optional) argument is an function that will be called when the request completes,
// and it will be given the response data from the request, it should be of type:
//
// func(T, T, ...)
//
// Where T are whatever types the other end of this Connection will respond with.
func (c *Connection) Request(requestName interface{}, sequence ...interface{}) {
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

	c.messageChan <- newRequestMessage(id, requestName, args)
}

// String returns an string representation of this Connection.
func (c *Connection) String() string {
	method := MethodString(c.method)
	return fmt.Sprintf("Connection(%s, %s, Dead=%t, Method=%s)", c.Address(), c.Store.String(), c.Dead(), method)
}

// Address returns the client address of this connection, usually for logging purposes.
//
// Format: ip:port
func (c *Connection) Address() string {
	// Note: no locking needed, never written to past creation time.
	return c.address
}

// Method returns one of the predefined constant organics.ConnectionMethod's which represent the method this
// connection is based upon.
func (c *Connection) Method() ConnectionMethod {
	// Note: no locking needed, never written to past creation time.
	return c.method
}

// Session returns the Session that is associated with this Connection.
func (c *Connection) Session() *Session {
	// Note: no locking needed, never written to past createion time.
	return c.session
}

// Dead tells weather this Connection is dead or not.
func (c *Connection) Dead() bool {
	c.access.RLock()
	defer c.access.RUnlock()

	return c.dead
}

// Kill will kill this connection as soon as possible, this function will block until the
// connection is dead.
//
// If this connection is already dead, this function is no-op.
func (c *Connection) Kill() {
	if !c.Dead() {
		c.deathWantedNotify <- true
	}
}

// DeathNotify returns an new channel on which true will be sent once Connection.Kill() has been
// called.
//
// If this connection is dead, then this function returns nil.
func (c *Connection) DeathNotify() chan bool {
	if c.Dead() {
		return nil
	}

	c.access.Lock()
	defer c.access.Unlock()

	ch := make(chan bool, 1)
	c.onDeathNotifications = append(c.onDeathNotifications, ch)
	return ch
}

func (c *Connection) waitForDeath() {
	<-c.deathNotify

	c.access.Lock()
	c.dead = true

	deathNotifications := make([]chan bool, len(c.onDeathNotifications))
	for i, ch := range c.onDeathNotifications {
		deathNotifications[i] = ch
	}
	c.onDeathNotifications = make([]chan bool, 0)
	c.access.Unlock()

	for _, ch := range deathNotifications {
		ch <- true
		close(ch)
	}

	c.Session().removeConnection(c.key)

	logger.Println("DeathNotify():", c)

	c.deathCompletedNotify <- true
}

func (c *Connection) disconnectTimer(timeout, rate time.Duration) {
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
						logger.Println("Ping timeout", c)
						c.disconnectFromTimeout <- true

					case <-c.disconnectTimerReset:
						break
					}
				}
			}
		}()
	}
}

func (c *Connection) resetDisconnectTimer() {
	c.disconnectTimerReset <- true
}

func newConnection(address string, session *Session, key interface{}, method ConnectionMethod) *Connection {
	c := new(Connection)
	c.key = key // Used for removal from session via removeConnection()
	c.deathNotify = make(chan bool)
	c.deathWantedNotify = make(chan bool)
	c.deathCompletedNotify = make(chan bool)
	c.data = make(map[interface{}]interface{})
	c.messageChan = make(chan *message)
	c.requestCompleters = make(map[float64]interface{})
	c.session = session
	c.address = address
	c.method = method

	c.disconnectFromTimeout = make(chan bool, 1)
	c.disconnectTimerReset = make(chan bool, 1)
	c.performPing = make(chan bool, 1)
	go c.waitForDeath()
	return c
}
