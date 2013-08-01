// Copyright 2012 Lightpoke. All rights reserved.
// This source code is subject to the terms and
// conditions defined in the "License.txt" file.

// Package filesystem implements file-based session storage.
package filesystem

import (
	"code.google.com/p/organics"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

func stringSections(s string, n int) []string {
	sections := make([]string, 0)
	for len(s) > n {
		sections = append(sections, s[:n])
		s = s[n:]
	}
	sections = append(sections, s)
	return sections
}

func sessionKeyToPath(key string) string {
	keyPath := url.QueryEscape(key)
	sections := stringSections(keyPath, 10)
	return filepath.Join(sections...)
}

type provider struct {
	access           sync.RWMutex
	stores           map[string]*organics.Store
	fileWritingLocks map[string]*sync.Mutex
	directory        string
}

func (p *provider) setStore(key string, store *organics.Store) {
	p.access.Lock()
	defer p.access.Unlock()

	p.stores[key] = store
}

func (p *provider) store(key string) (store *organics.Store, ok bool) {
	p.access.RLock()
	defer p.access.RUnlock()

	store, ok = p.stores[key]
	return
}

func (p *provider) getWriteLock(key string) *sync.Mutex {
	p.access.Lock()
	defer p.access.Unlock()

	lock, ok := p.fileWritingLocks[key]
	if !ok {
		lock = new(sync.Mutex)
		p.fileWritingLocks[key] = lock
	}
	return lock
}

func (p *provider) Save(key string, whatChanged interface{}, s *organics.Store) error {
	p.setStore(key, s)

	lock := p.getWriteLock(key)
	lock.Lock()
	defer lock.Unlock()

	keyPath := filepath.Join(p.directory, sessionKeyToPath(key))

	if _, err := os.Stat(keyPath); err != nil {
		if os.IsNotExist(err) {
			keyPathDir := strings.Split(keyPath, string(os.PathSeparator))
			keyPathDir = keyPathDir[:len(keyPathDir)-1]
			dirPath := filepath.Join(keyPathDir...)

			err = os.MkdirAll(dirPath, os.ModeDir)
			if err != nil {
				return fmt.Errorf("Error saving session %v", err)
			}
		} else {
			return fmt.Errorf("Error saving session %v", err)
		}
	}

	data, err := s.GobEncode()
	if err != nil {
		return fmt.Errorf("Error saving session %v", err)
	}

	err = ioutil.WriteFile(keyPath, data, 0666)
	if err != nil {
		return fmt.Errorf("Error saving session %v", err)
	}

	return nil
}

func (p *provider) Load(key string) *organics.Store {
	store, ok := p.store(key)
	if ok {
		// We have it in memory already
		return store
	}

	// We don't have it in memory, so load it from file.
	keyPath := filepath.Join(p.directory, sessionKeyToPath(key))

	file, err := os.Open(keyPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		} else {
			log.Println("Error reading session file", err)
			return nil
		}
	}

	store = organics.NewStore()

	data, err := ioutil.ReadAll(file)
	if err != nil {
		log.Println("Error reading session file", err)
		return nil
	}

	err = store.GobDecode(data)
	if err != nil {
		log.Println("Error decoding session file", err)
		return nil
	}
	p.stores[key] = store
	return store
}

func Provider(directory string) (organics.SessionProvider, error) {
	p := new(provider)
	p.stores = make(map[string]*organics.Store)
	p.fileWritingLocks = make(map[string]*sync.Mutex)
	p.directory = directory

	if _, err := os.Stat(directory); err != nil {
		if os.IsNotExist(err) {
			err = os.MkdirAll(directory, os.ModeDir)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}
	return p, nil
}
