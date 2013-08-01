// Copyright 2012 Lightpoke. All rights reserved.
// This source code is subject to the terms and
// conditions defined in the "License.txt" file.

package organics

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"strings"
	"sync"
)

const (
	storeVersion uint8 = 1
)

// Store is an atomic data storage, which is used both by the Connection struct
// and Session struct to provide session and connection based storage
// facilities.
type Store struct {
	access              sync.RWMutex
	data                map[interface{}]interface{}
	dataChangeNotifiers []chan bool
	dataWatchers        map[chan interface{}]bool
}

func (s *Store) sendDataChanged() {
	for _, ch := range s.dataChangeNotifiers {
		ch <- true
		close(ch)
	}
	s.dataChangeNotifiers = make([]chan bool, 0)
}

func (s *Store) doKeyChanged(key interface{}) {
	for ch, active := range s.dataWatchers {
		if active {
			if len(ch) == cap(ch) {
				// Max buffered elements hit, spawn goroutines starting now.
				go func() {
					ch <- key
				}()
			} else {
				ch <- key
			}
		}
	}
}

// RemoveWatcher removes the specified data watcher; it must be an channel that
// was returned from the ChangeWatcher() method.
//
// See ChangeWatcher() for more information.
func (s *Store) RemoveWatcher(ch chan interface{}) {
	s.access.Lock()
	defer s.access.Unlock()

	// Mark inactive
	s.dataWatchers[ch] = false

	// Close channel, empty the channel buffer.
	close(ch)
	for len(ch) >= 1 {
		<-ch
	}

	// Remove watcher channel.
	delete(s.dataWatchers, ch)
}

// ChangeWatcher returns an channel over which the keys of data inside this
// store will be sent as they are added, removed, or changed.
//
// In order for the caller to not miss any data changes in this store, the
// channel will continue to have change events sent over it until an call to
// the RemoveWatcher() method.
func (s *Store) ChangeWatcher() chan interface{} {
	s.access.Lock()
	defer s.access.Unlock()

	ch := make(chan interface{}, 10)
	s.dataWatchers[ch] = true
	return ch
}

// ChangeNotify returns an channel over which true will be sent once the data
// inside this store has changed.
//
// After true is sent, the channel is closed. As such, if you intend to receive
// notifications forever, you'll need to constantly create a new channel with
// an call to this function.
func (s *Store) ChangeNotify() chan bool {
	s.access.Lock()
	defer s.access.Unlock()

	ch := make(chan bool, 1)
	s.dataChangeNotifiers = append(s.dataChangeNotifiers, ch)
	return ch
}

// Data returns an copy of this store's underlying data map.
func (s *Store) Data() map[interface{}]interface{} {
	s.access.RLock()
	defer s.access.RUnlock()

	cpy := make(map[interface{}]interface{}, len(s.data))
	for key, value := range s.data {
		cpy[key] = value
	}
	return cpy
}

// Keys returns an copy of this store's underlying data map's keys.
func (s *Store) Keys() []interface{} {
	s.access.RLock()
	defer s.access.RUnlock()

	cpy := make([]interface{}, len(s.data))
	i := 0
	for key, _ := range s.data {
		cpy[i] = key
		i++
	}
	return cpy
}

// Values returns an copy of this store's underlying data map's values.
func (s *Store) Values() []interface{} {
	s.access.RLock()
	defer s.access.RUnlock()

	cpy := make([]interface{}, len(s.data))
	i := 0
	for _, value := range s.data {
		cpy[i] = value
		i++
	}
	return cpy
}

// Len is short hand for the following, but doesn't have to create an copy of
// the data map as it cannot be modified by the caller:
//
//  return len(s.Data())
//
func (s *Store) Len() int {
	s.access.RLock()
	defer s.access.RUnlock()

	return len(s.data)
}

// Implements the gob encoding interface (see the encoding/gob package for more
// information)
func (s *Store) GobEncode() ([]byte, error) {
	s.access.RLock()
	defer s.access.RUnlock()

	buf := new(bytes.Buffer)

	encoder := gob.NewEncoder(buf)
	err := encoder.Encode(storeVersion)
	if err != nil {
		return nil, err
	}

	err = encoder.Encode(s.data)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// Implements the gob decoding interface (see the encoding/gob package for more
// information).
func (s *Store) GobDecode(data []byte) error {
	s.access.Lock()
	defer s.access.Unlock()

	buf := bytes.NewBuffer(data)

	decoder := gob.NewDecoder(buf)
	var version uint8
	err := decoder.Decode(&version)
	if err != nil {
		return err
	}

	err = decoder.Decode(&s.data)
	if err != nil {
		return err
	}

	for key, _ := range s.data {
		s.doKeyChanged(key)
	}

	return nil
}

// String returns an string representation of this Store
//
// Note that this prints all data inside this store -- as such if the store may
// contain sensitive data (I.e. passwords, credit card numbers, etc) then you
// might want to never print the store for security reasons.
func (s *Store) String() string {
	s.access.RLock()
	defer s.access.RUnlock()

	var b bytes.Buffer
	b.WriteString("Store(")
	ms := fmt.Sprint(s.data)
	ms = strings.TrimLeft(ms, "map[")
	ms = strings.TrimRight(ms, "]")
	b.WriteString(ms)
	b.WriteString(")")
	return b.String()
}

// Has tells weather this Store has the specified key.
func (s *Store) Has(key interface{}) bool {
	s.access.RLock()
	defer s.access.RUnlock()

	_, ok := s.data[key]
	return ok
}

// Set sets the specified key to the specified value
func (s *Store) Set(key, value interface{}) {
	s.access.Lock()
	defer s.access.Unlock()

	s.data[key] = value
	s.sendDataChanged()
	s.doKeyChanged(key)
}

// Get returns the specified key from this stores data, or if this store does
// not have the specified key then the key is set to the default value and the
// default value is returned.
func (s *Store) Get(key, defaultValue interface{}) interface{} {
	s.access.RLock()

	value, ok := s.data[key]
	if !ok {
		s.access.RUnlock()

		s.access.Lock()
		defer s.access.Unlock()

		s.data[key] = defaultValue
		s.sendDataChanged()
		s.doKeyChanged(key)
		return defaultValue
	}

	s.access.RUnlock()
	return value
}

// Delete deletes the specified key from this stores data, if there is no such
// key then this function is no-op.
func (s *Store) Delete(key interface{}) {
	s.access.Lock()
	defer s.access.Unlock()

	delete(s.data, key)
	s.sendDataChanged()
	s.doKeyChanged(key)
}

// Reset resets this store such that there is absolutely no data inside of it.
func (s *Store) Reset() {
	s.access.Lock()
	defer s.access.Unlock()

	for key, _ := range s.data {
		delete(s.data, key)
		s.sendDataChanged()
		s.doKeyChanged(key)
	}

	s.data = make(map[interface{}]interface{})
}

// Copy returns an new 1:1 copy of this store and it's data.
//
// The copy does not include data change notifiers returned by ChangeNotify().
func (s *Store) Copy() *Store {
	s.access.RLock()
	defer s.access.RUnlock()

	cpy := NewStore()
	for key, value := range s.data {
		cpy.data[key] = value
	}
	return cpy
}

// NewStore returns an new intialized *Store.
func NewStore() *Store {
	s := new(Store)
	s.data = make(map[interface{}]interface{})
	s.dataChangeNotifiers = make([]chan bool, 0)
	s.dataWatchers = make(map[chan interface{}]bool)
	return s
}
