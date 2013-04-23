// Copyright 2012 Lightpoke. All rights reserved.
// This source code is subject to the terms and
// conditions defined in the "License.txt" file.

package organics

import (
	"fmt"
	"sync"
)

// SessionProvider is the interface that an storage provider needs to fill in order to be accepted
// as an session provider.
type SessionProvider interface {
	// Store should store the session however the provider deems neccisary, in an way such that the
	// provider can retreive it later when an call to Get() (defined in this interface) is made.
	//
	// Should an error be returned, both the session creation and (organics) connection creation
	// will fail.
	Store(key string, s *Session) error

	// Get should return an previously stored pointer to an session object or nil if there is no
	// such session known by the key parameter.
	Get(key string) *Session
}

type lockedSessionProvider struct {
	sp SessionProvider
	sync.Mutex
}

func newLockedSessionProvider(sp SessionProvider) SessionProvider {
	n := new(lockedSessionProvider)
	n.sp = sp
	return n
}

func (l *lockedSessionProvider) Store(key string, s *Session) error {
	l.Lock()
	defer l.Unlock()

	return l.sp.Store(key, s)
}

func (l *lockedSessionProvider) Get(key string) *Session {
	l.Lock()
	defer l.Unlock()

	return l.sp.Get(key)
}

// Session represents an user's session, which is built up of zero to multiple Connection objects.
type Session struct {
	Store

	access sync.RWMutex

	dead                                bool
	connections                         map[interface{}]*Connection
	deathNotify                         chan bool
	onDeathNotifications, quietNotifies []chan bool
	hasDataChangedRoutine               bool
}

// String returns an string representation of this Session.
func (s *Session) String() string {
	return fmt.Sprintf("Session(%s, Dead=%t)", s.Store.String(), s.Dead())
}

// Dead tells weather this Session is dead or not.
func (s *Session) Dead() bool {
	s.access.RLock()
	defer s.access.RUnlock()

	return s.dead
}

// DeathNotify returns an channel on which true will be sent once Session.Kill() has been called.
func (s *Session) DeathNotify() chan bool {
	s.access.Lock()
	defer s.access.Unlock()

	ch := make(chan bool, 1)
	s.onDeathNotifications = append(s.onDeathNotifications, ch)
	return ch
}

// Kill kills this Session, all Connections that represent this Session will be killed as well.
func (s *Session) Kill() {
	if !s.Dead() {
		s.access.Lock()
		s.dead = true

		deathNotifications := make([]chan bool, len(s.onDeathNotifications))
		for i, ch := range s.onDeathNotifications {
			deathNotifications[i] = ch
		}
		s.onDeathNotifications = make([]chan bool, 0)
		s.access.Unlock()

		// Request each connection's death first
		for _, conn := range s.connections {
			if conn != nil {
				conn.Kill()
			}
		}

		// Inform all DeathNotify() channels of death
		for _, ch := range deathNotifications {
			ch <- true
			close(ch)
		}

		logger.Println("DeathNotify():", s)
	}
}

// Connections returns all underlying Connections which represent this Session.
func (s *Session) Connections() []*Connection {
	s.access.RLock()
	defer s.access.RUnlock()

	connections := make([]*Connection, len(s.connections))
	i := 0
	for _, conn := range s.connections {
		connections[i] = conn
		i++
	}
	return connections
}

// QuietNotify returns an single channel on which true will be sent, as soon as there are no longer
// any Connections which represent this Session.
//
// Also see: Session.Quiet()
func (s *Session) QuietNotify() chan bool {
	ch := make(chan bool, 1)
	s.quietNotifies = append(s.quietNotifies, ch)
	return ch
}

// Quiet tells weather there are no Connections which represent this Session at this current point
// in time.
//
// Short hand for:
//  len(s.Connections()) == 0
func (s *Session) Quiet() bool {
	s.access.RLock()
	defer s.access.RUnlock()

	return len(s.connections) == 0
}

// Request simply calls Request() on each Connection returned by s.Connections()
//
// See Connection.Request() for more information.
func (s *Session) Request(requestName interface{}, sequence ...interface{}) {
	for _, conn := range s.Connections() {
		conn.Request(requestName, sequence...)
	}
}

func (s *Session) waitForDeath() {
	<-s.deathNotify

	for _, ch := range s.onDeathNotifications {
		ch <- true
		close(ch)
	}
}

func (s *Session) removeConnection(key interface{}) {
	s.access.Lock()
	delete(s.connections, key)
	s.access.Unlock()

	if len(s.Connections()) == 0 {
		s.access.RLock()
		defer s.access.RUnlock()

		// Send notifications that all the connections are gone
		for _, ch := range s.quietNotifies {
			ch <- true
		}
		s.quietNotifies = make([]chan bool, 0)
	}
}

func (s *Session) addConnection(key interface{}, c *Connection) {
	s.access.Lock()
	defer s.access.Unlock()

	s.connections[key] = c
}

func (s *Session) getConnection(key interface{}) *Connection {
	s.access.RLock()
	defer s.access.RUnlock()

	return s.connections[key]
}

// NewSession returns an new initialized session object, only for use when implementing an
// SessionProvider, there is no other purpose to create an Session object on your own.
func NewSession() *Session {
	s := new(Session)
	s.deathNotify = make(chan bool)
	s.connections = make(map[interface{}]*Connection)
	s.Store.data = make(map[interface{}]interface{})
	s.Store.dataChangedNotify = make(chan bool, 1)
	return s
}
