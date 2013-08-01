// Copyright 2012 Lightpoke. All rights reserved.
// This source code is subject to the terms and
// conditions defined in the "License.txt" file.

// Package memory implements in-memory session storage.
package memory

import (
	"code.google.com/p/organics"
	"sync"
)

type provider struct {
	access   sync.RWMutex
	sessions map[string]*organics.Store
}

func (p *provider) Save(key string, whatChanged interface{}, s *organics.Store) error {
	p.access.Lock()
	defer p.access.Unlock()

	theStore, ok := p.sessions[key]
	if !ok {
		p.sessions[key] = s.Copy()
		return nil
	}

	if !s.Has(whatChanged) {
		// It was removed
		theStore.Delete(whatChanged)
	} else {
		v := s.Get(whatChanged, nil)
		theStore.Set(whatChanged, v)
	}

	return nil
}

func (p *provider) Load(key string) *organics.Store {
	p.access.RLock()
	defer p.access.RUnlock()

	s, ok := p.sessions[key]
	if !ok {
		return nil
	}
	return s.Copy()
}

func Provider() organics.SessionProvider {
	p := new(provider)
	p.sessions = make(map[string]*organics.Store)
	return p
}
