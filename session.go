// Copyright 2012 Lightpoke. All rights reserved.
// This source code is subject to the terms and
// conditions defined in the "License.txt" file.

package organics

import (
	"fmt"
	"sync"
	"time"
)

// Session represents an user's session, which is built up of zero to multiple
// different connections.
//
// An session is considered dead once there are no more connections to
// represent it.
type Session struct {
	*Store

	access sync.RWMutex

	key                               string
	server                            *Server
	dead                              bool
	connections                       map[interface{}]*ServerConnection
	deathNotify, deathCompletedNotify chan bool
	deathNotifications                []chan bool
	hasDataChangedRoutine             bool
	stopSaving                        chan bool
}

// String returns an string representation of this Session.
//
// For security reasons the string will not contain the stores data, and will
// instead only contain the length of the store (I.e. how many objects it
// contains).
func (s *Session) String() string {
	return fmt.Sprintf("Session(Store.Len()=%v, Dead=%t)", s.Store.Len(), s.Dead())
}

// Dead tells weather this Session is dead or not. An session is considered
// dead once it's Kill() method has been called.
func (s *Session) Dead() bool {
	s.access.RLock()
	defer s.access.RUnlock()

	return s.dead
}

// DeathNotify returns an channel on which true will be sent once this session
// is killed via it's Kill() method.
//
// If the session is already dead, and channel is still returned and will still
// have true sent over it.
func (s *Session) DeathNotify() chan bool {
	s.access.Lock()
	defer s.access.Unlock()

	ch := make(chan bool, 1)

	if s.dead {
		// Already dead
		ch <- true

	} else {
		// We'll send notification later.
		s.deathNotifications = append(s.deathNotifications, ch)
	}

	return ch
}

// Kill kills this session, all connections that represent this session will be
// killed as well.
//
// If this session is already dead, this function is no-op.
func (s *Session) Kill() {
	if !s.Dead() {
		// Signal death
		s.deathNotify <- true

		// Wait for completion
		<-s.deathCompletedNotify
	}
}

// Connections returns an list of all the underlying connections which
// represent this session.
//
// If this session is dead, an slice of length zero is returned.
func (s *Session) Connections() []*ServerConnection {
	s.access.RLock()
	defer s.access.RUnlock()

	if s.dead {
		return make([]*ServerConnection, 0)
	}

	connections := make([]*ServerConnection, len(s.connections))
	i := 0
	for _, conn := range s.connections {
		connections[i] = conn
		i++
	}
	return connections
}

// Request performs an request on each connection who currently represents this
// session.
//
// See Connection.Request() for more information on how request work.
func (s *Session) Request(requestName interface{}, sequence ...interface{}) {
	for _, conn := range s.Connections() {
		conn.Request(requestName, sequence...)
	}
}

func (s *Session) waitToSave() {
	s.access.RLock()

	// Get the session provider, panic if there is none yet.
	sp := s.server.Provider()
	if sp == nil {
		panic("Server has no session provider set.")
	}

	s.access.RUnlock()

	w := s.ChangeWatcher()

	for {
		// Wait for session data to change
		select {
		case whatChanged := <-w:
			err := sp.Save(s.key, whatChanged, s.Store)
			if err != nil {
				logger().Println(err)
			}

		case <-s.stopSaving:
			// We need to stop saving after an certain period of time.
			s.access.RLock()
			stopSavingTimer := time.After(s.server.SessionTimeout())
			s.access.RUnlock()

			for {
				select {
				case whatChanged := <-w:
					err := sp.Save(s.key, whatChanged, s.Store)
					if err != nil {
						logger().Println(err)
					}

				case <-stopSavingTimer:
					s.access.Lock()
					s.server = nil
					s.access.Unlock()
					return
				}
			}
		}
	}
}

func (s *Session) waitForDeath() {
	<-s.deathNotify

	// Request each connection's death first
	for _, conn := range s.Connections() {
		conn.Kill()
	}

	s.access.Lock()

	// Get the session provider, panic if there is none yet.
	sp := s.server.Provider()
	if sp == nil {
		panic("Server has no session provider set.")
	}

	s.dead = true

	deathNotifications := make([]chan bool, len(s.deathNotifications))
	for i, ch := range s.deathNotifications {
		deathNotifications[i] = ch
	}
	s.deathNotifications = make([]chan bool, 0)

	s.access.Unlock()

	for _, ch := range deathNotifications {
		ch <- true
		close(ch)
	}

	for _, key := range s.Store.Keys() {
		err := sp.Save(s.key, key, s.Store)
		if err != nil {
			logger().Println(err)
		}
	}

	s.server.uncache(s.key)

	logger().Println("DeathNotify():", s)
	s.deathCompletedNotify <- true
	s.stopSaving <- true
}

func (s *Session) removeConnection(key interface{}) {
	s.access.Lock()

	delete(s.connections, key)

	if len(s.connections) == 0 {
		s.access.Unlock()

		// No more connections to represent this session, kill it.
		s.Kill()

		return
	}

	s.access.Unlock()
}

func (s *Session) addConnection(key interface{}, c *ServerConnection) {
	s.access.Lock()
	defer s.access.Unlock()

	s.connections[key] = c
}

func (s *Session) getConnection(key interface{}) *ServerConnection {
	s.access.RLock()
	defer s.access.RUnlock()

	return s.connections[key]
}

func (s *Session) start() {
	go s.waitForDeath()
	go s.waitToSave()
}

// newSession returns an new initialized session object.
func newSession(key string, server *Server) *Session {
	s := new(Session)
	s.key = key
	s.Store = NewStore()
	s.server = server
	s.dead = false
	s.connections = make(map[interface{}]*ServerConnection)
	s.deathNotify = make(chan bool)
	s.deathCompletedNotify = make(chan bool)
	s.deathNotifications = make([]chan bool, 0)
	s.stopSaving = make(chan bool)
	return s
}
