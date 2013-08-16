// Copyright 2012 Lightpoke. All rights reserved.
// This source code is subject to the terms and
// conditions defined in the "License.txt" file.

package organics

// SessionProvider is the interface that an storage provider needs to fill in
// order to be accepted as an valid session provider.
//
// Conceptually, an session provider can be thought of as an thread-safe map
// implementation.
//
// Session providers Set() and Get() methods must be safe to call from multiple
// goroutines.
type SessionProvider interface {
	// Save should save the store's underlying data however the provider deems
	// proper. The key is guaranteed to be unique and will be the same key that
	// is passed into Get() for retreival.
	//
	// The channel returned should have either nil or an valid error sent over
	// it to respectively signal completion, or an error saving.
	Save(sessionKey string, whatChanged string, s *Store) error

	// Load should return an previously saved store object given the unique key
	// it was saved with.
	//
	// The store object need not be the same (I.e. it can be an new pointer or
	// object all-together), as long as the underlying data is identical.
	Load(key string) *Store
}
