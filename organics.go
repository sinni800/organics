// Copyright 2012 Lightpoke. All rights reserved.
// This source code is subject to the terms and
// conditions defined in the "License.txt" file.

// Organics provides two-way communication between an browser and web server.
package organics

import (
	"io"
	"io/ioutil"
	"log"
	"sync"
)

var (
	loggerAccess sync.RWMutex
	theLogger    = log.New(ioutil.Discard, "organics ", 0)
)

// SetDebugOutput specifies an io.Writer which Organics will write debug
// information to.
func SetDebugOutput(w io.Writer) {
	loggerAccess.Lock()
	defer loggerAccess.Unlock()

	theLogger = log.New(w, "organics ", log.Ltime)
}

func logger() *log.Logger {
	loggerAccess.RLock()
	defer loggerAccess.RUnlock()

	return theLogger
}
