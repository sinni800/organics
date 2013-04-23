// Copyright 2012 Lightpoke. All rights reserved.
// This source code is subject to the terms and
// conditions defined in the "License.txt" file.

// Package filesystem implements file-based session storage.
package filesystem

import(
	"code.google.com/p/organics"
	"path/filepath"
	"io/ioutil"
	"strings"
	"net/url"
	"sync"
	"log"
	"os"
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
	sessions map[string]*organics.Session
	fileWritingLocks map[string]*sync.Mutex
	directory string
}

func (p *provider) getWriteLock(key string) *sync.Mutex {
	lock, ok := p.fileWritingLocks[key]
	if !ok {
		lock = new(sync.Mutex)
		p.fileWritingLocks[key] = lock
	}
	return lock
}

func (p *provider) saveSession(key string, s *organics.Session) {
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
				log.Println("Error saving session", err)
				return
			}
		} else {
			log.Println("Error saving session", err)
			return
		}
	}

	data, err := s.GobEncode()
	if err != nil {
		log.Println("Error saving session", err)
		return
	}

	err = ioutil.WriteFile(keyPath, data, 0666)
	if err != nil {
		log.Println("Error saving session", err)
		return
	}
}

func (p *provider) Store(key string, s *organics.Session) error {
	p.sessions[key] = s

	go p.saveSession(key, s)

	return nil
}

func (p *provider) Get(key string) *organics.Session {
	session, ok := p.sessions[key]
	if ok {
		// We have it in memory already
		return session
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

	session = organics.NewSession()

	data, err := ioutil.ReadAll(file)
	if err != nil {
		log.Println("Error reading session file", err)
		return nil
	}

	err = session.GobDecode(data)
	if err != nil {
		log.Println("Error decoding session file", err)
		return nil
	}
	p.sessions[key] = session
	return session
}

func Provider(directory string) (organics.SessionProvider, error) {
	p := new(provider)
	p.sessions = make(map[string]*organics.Session)
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

