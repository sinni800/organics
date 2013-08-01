// Copyright 2012 Lightpoke. All rights reserved.
// This source code is subject to the terms and
// conditions defined in the "License.txt" file.

// Organics simple hello world server/client example.
package main

import (
	"code.google.com/p/organics"

	// For sessions being stored in-memory:
	"code.google.com/p/organics/provider/memory"

	// Or use this line for sessions being saved on-file:
	//"code.google.com/p/organics/provider/filesystem"

	"log"
	"net/http"
	"os"
	"os/signal"
	"time"
)

func init() {
	// Not needed unless trying to debug things:
	organics.SetDebugOutput(os.Stdout)
}

func HandleConnect(connection *organics.Connection) {
	// This timer will be used to display the connection's time on the browser.
	go func() {
		deathNotify := connection.DeathNotify()

		// Initially, the page will load, so we want to display the time right
		// away.
		intervalTime := 0 * time.Second
		for {
			select {
			case <-time.After(intervalTime):
				// Now that we've set the time once, future updates should be
				// in intervals of 1 second.
				intervalTime = 1 * time.Second

				// Get current timestamp
				currentTime := time.Now().Unix()

				// Determine timer start time
				startTime := connection.Get("startTime", currentTime).(int64)

				// Subtract the two -- that's how long their connection has
				// been alive.
				t := currentTime - startTime

				// Give the browser an request.
				connection.Request("SetConnectionTime", t)

			case <-deathNotify:
				// If they disconnect, we'll basically 'pause' this timer by
				// killing the goroutine, next time they connect we'll start an
				// new timer.
				return
			}
		}
	}()

	// And this timer will be used to display their session's time on the
	// browser.
	session := connection.Session()

	go func() {
		// This timer is just starting, so we want it to send an update to
		// the browser ASAP
		intervalTime := 0 * time.Second

		for {
			select {
			case <-time.After(intervalTime):
				// Now that we've set the time once, future updates should
				// be in intervals of 1 second
				intervalTime = 1 * time.Second

				// Get current timestamp
				currentTime := time.Now().Unix()

				// Determine timer start time
				startTime := session.Get("startTime", currentTime).(int64)

				// Subtract the two -- that's how long their connection has
				// been alive.
				t := currentTime - startTime

				// Lets give each browser tab (connection) an request, we
				// can do that by using Session.Request, which performs an
				// request to each connection that is represented by that
				// session.
				session.Request("SetSessionTime", t)

			case <-connection.DeathNotify():
				// If they disconnect, we'll basically 'pause' this timer by
				// killing the goroutine, next time they connect we'll start an
				// new timer.
				return
			}
		}
	}()
}

func HandleGiveBrowserMessage(msg1, msg2 string, connection *organics.Connection) error {
	// Lets give the browser an message.
	response := "Message from server: " + msg1 + msg2

	connection.Request("Message", response, func() {
		log.Println("Our message got to the browser! We said:\n\t", response)
	})

	return nil
}

func main() {
	var err error

	// For the filesystem session provider:
	//
	//  sessionProvider, err := filesystem.Provider("organics_sessions")
	//  if err != nil {
	//      log.Fatal(err)
	//  }
	//
	// (where 'organics_sessions' is the folder to store sessions).

	// We use the in-memory session provider:
	sessionProvider := memory.Provider()

	// We create our organics server, using our sessionProvider
	server := organics.NewServer(sessionProvider)

	// Special origin name to allow anyone to access this server from any
	// origin domain is "*".
	//‏w
	// Note: If you do not use "*" to allow all origins, then ensure that you
	// at least allow your own origin, or else all access will be blocked from
	// WebSockets (not from Long Polling, due to the way CORS operates).

	// We allow connections from any origin:
	server.SetOriginAccess("*", true)

	// Or specify allowed origins like so:
	//
	// Think like this: We are www.facebook.com, and we want to allow
	// www.twitter.com to access this server using cross-domain requests.
	//
	// server.SetOriginAccess("www.twitter.com", true)

	server.Handle(organics.Connect, HandleConnect)
	server.Handle("GiveBrowserMessage", HandleGiveBrowserMessage)

	// Server up the client directory containing the JavaScript client.
	http.Handle("/", http.FileServer(http.Dir("src/code.google.com/p/organics/examples/organics_hello_world/client")))
	http.Handle("/javascript/", http.FileServer(http.Dir("src/code.google.com/p/organics/")))

	// And finally handle the /app url using our Organics server.
	http.Handle("/app", server)

	// Start an goroutine to listen for ctrl+c or exit signal so that we
	// properly shut down the server (otherwise super-super-recent session data
	// could be lost).
	go func() {
		exitSignal := make(chan os.Signal, 1)
		signal.Notify(exitSignal, os.Interrupt, os.Kill)

		// Block until ctrl+c or exit signal is received
		<-exitSignal

		// Kill the server
		server.Kill()

		// Exit program
		os.Exit(1)
	}()

	err = http.ListenAndServe(":80", nil)
	if err != nil {
		log.Fatal(err)
	}
}
