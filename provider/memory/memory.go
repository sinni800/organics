// Copyright 2012 Lightpoke. All rights reserved.
// This source code is subject to the terms and
// conditions defined in the "License.txt" file.

// Package memory implements in-memory session storage.
package memory

import "code.google.com/p/organics"

type provider struct {
	sessions map[string]*organics.Session
}

func (p *provider) Store(key string, s *organics.Session) error {
	p.sessions[key] = s
	return nil
}

func (p *provider) Get(key string) *organics.Session {
	return p.sessions[key]
}

func Provider() organics.SessionProvider {
	p := new(provider)
	p.sessions = make(map[string]*organics.Session)
	return p
}
