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

// Store is an atomic data storage, which is used both by organics.Connection and organics.Session
// to provide session and connection based storage facilities.
type Store struct {
	access            sync.RWMutex
	data              map[interface{}]interface{}
	dataChangedNotify chan bool
}

func (s *Store) dataChanged() {
	if s.dataChangedNotify != nil {
		s.dataChangedNotify <- true
	}
}

// Implements encoding/gob.GobEncoder interface
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

// Implements encoding/gob.GobDecoder interface
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

	return nil
}

// String returns an string representation of this Store
func (s *Store) String() string {
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
	s.dataChanged()
}

// Get returns the specified key from this Store's data, or if this Store does not have the
// specified key, then the key is set to the default value, and the default value is returned.
func (s *Store) Get(key, defaultValue interface{}) interface{} {
	s.access.Lock()
	defer s.access.Unlock()

	value, ok := s.data[key]
	if !ok {
		s.data[key] = defaultValue
		s.dataChanged()
		return defaultValue
	}
	return value
}

// Delete deletes the specified key from this Store's data, if there is no such key, this function
// is no-op.
func (s *Store) Delete(key interface{}) {
	s.access.Lock()
	defer s.access.Unlock()

	delete(s.data, key)
	s.dataChanged()
}

// Reset resets this Store such that there is absolutely no data inside of it.
func (s *Store) Reset() {
	s.access.Lock()
	defer s.access.Unlock()

	s.data = make(map[interface{}]interface{})
	s.dataChanged()
}
