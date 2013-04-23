// Copyright 2012 Lightpoke. All rights reserved.
// This source code is subject to the terms and
// conditions defined in the "License.txt" file.

// Simple hello world server and client demonstration
package main

import(
	"code.google.com/p/organics"

	// For sessions being stored in-memory:
	"code.google.com/p/organics/provider/memory"

	// Or use this line for sessions being saved on-file:
	//"code.google.com/p/organics/provider/filesystem"

	"net/http"
	"time"
	"log"
	//"os"
)

func init() {
	// Not needed unless trying to debug things:
	//organics.SetDebugOutput(os.Stdout)
}

func HandleConnect(connection *organics.Connection) {
	// This timer will be used to display the connection's time on the browser
	go func() {
		deathNotify := connection.DeathNotify()

		// Initially, the page will load, so we want to display the time right away.
		intervalTime := 0 * time.Second
		for{
			select{
				case <-time.After(intervalTime):
					// Now that we've set the time once, future updates should be in intervals
					// of 1 second
					intervalTime = 1 * time.Second

					// We store the current time in their Connection's Store, and then add 1 to it
					currentTime := connection.Get("time", float64(0)).(float64)
					currentTime += 1
					connection.Set("time", currentTime)

					// Lets give the browser an request
					connection.Request("SetConnectionTime", currentTime)

				case <-deathNotify:
					// If they disconnect, we'll basically 'pause' this timer by killing the
					// goroutine, next time they connect we'll start an new timer (again).
					return
			}
		}
	}()

	// And this timer will be used to display their session's time on the browser
	session := connection.Session()

	// We'll store "hasTimer" inside their Session's Store, that way we know if another browser
	// tab (connection) has already created this timer.
	if !session.Get("hasTimer", false).(bool) {
		session.Set("hasTimer", true)

		go func() {
			// This timer is just starting, so we want it to send an update to the browser ASAP
			intervalTime := 0 * time.Second

			for{
				select{
					case <-time.After(intervalTime):
						// Now that we've set the time once, future updates should be in intervals
						// of 1 second
						intervalTime = 1 * time.Second

						// We store the current time in their Sessions's Store, and then add 1 to it
						currentTime := session.Get("time", float64(0)).(float64)
						currentTime += 1
						session.Set("time", currentTime)

						// Lets give each browser tab (connection) an request, we can do that by
						// using Session.Request, which performs an request to each connection that
						// is represented by that session.
						session.Request("SetSessionTime", currentTime)

					case <-session.QuietNotify():
						// If the session is quiet, it means there are no connections right now
						// that represent the session, so we can stop updating their session timer.

						// Be sure to clear this, otherwise the timer will never start again!
						session.Delete("hasTimer")
						return
				}
			}
		}()
	}
}

func HandleGiveBrowserMessage(msg1, msg2 string, connection *organics.Connection) error {
	// Lets give the browser an message
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
	// sessionProvider, err := filesystem.Provider("organics_sessions") // Folder for sessions
	// if err != nil {
	//     log.Fatal(err)
	// }
	//
	// Or for the memory session provider:
	//
	sessionProvider := memory.Provider()

	// We create our organics server, using our sessionProvider
	server := organics.Server(sessionProvider)

	// Special origin name to allow anyone to access this server from any origin:
	//
	// Note: If you do not use "*" to allow all origins, then ensure that you at least allow your
	// own origin, or else all access will be blocked from WebSockets (not from Long Polling, due
	// to the way CORS operates)
	server.SetOriginAccess("*", true)

	// Or specify allowed origins like so:
	//
	// Think like this: We are www.facebook.com, and we want to allow www.twitter.com to access
	// this server using cross-domain requests.
	//
	// server.SetOriginAccess("www.twitter.com", true)

	server.Handle(organics.Connect, HandleConnect)
	server.Handle("GiveBrowserMessage", HandleGiveBrowserMessage)

	// If you want to change the folder where Organics will place the session files, use this:
	//     Note: Only applies to provider/filesystem
	//
	// organics.InstallProvider(filesystem.NewProvider("organics_sessions"))

	http.Handle("/", http.FileServer(http.Dir(".")))

	http.Handle("/app", server)

	err = http.ListenAndServe(":80", nil)
	if err != nil {
		log.Fatal(err)
	}
}

