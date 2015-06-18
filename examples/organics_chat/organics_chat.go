// Copyright 2012 Lightpoke. All rights reserved.
// This source code is subject to the terms and
// conditions defined in the "License.txt" file.

// Simple chat server and client example
package main

import (
	"github.com/sinni800/organics"

	// For sessions being stored in-memory:
	"github.com/sinni800/organics/provider/memory"

	// Or use this line for sessions being saved on-file:
	//"github.com/sinni800/organics/provider/filesystem"

	"html"
	"log"
	"net/http"
	"os"
	"sync"
)

var (
	server               *organics.Server
	recentMessages       []string
	recentMessagesAccess sync.RWMutex
)

func init() {
	// Not needed unless trying to debug things:
	organics.SetDebugOutput(os.Stdout)
}

func sendMessageToAll(msg string) {
	recentMessagesAccess.Lock()
	if len(recentMessages)+1 > 25 {
		recentMessages = recentMessages[len(recentMessages)-25 : len(recentMessages)]
	}
	recentMessages = append(recentMessages, msg)
	recentMessagesAccess.Unlock()

	log.Println(msg)
	for _, conn := range server.Connections() {
		conn.Request("DisplayMessage", msg)
	}
}

func doConnect(connection *organics.Connection) {
	// Send them the most recent 25 messages
	recentMessagesAccess.RLock()
	for _, msg := range recentMessages {
		connection.Request("DisplayMessage", msg)
	}
	recentMessagesAccess.RUnlock()

	// Wait untill they disconnect
	go func() {
		<-connection.DeathNotify()

		username := connection.Get("username", "").(string)
		sendMessageToAll(username + " has left.")
	}()
}

func doMessage(msg string, connection *organics.Connection) {
	msg = html.EscapeString(msg)

	username := connection.Get("username", "").(string)

	if username == "admin" {
		log.Println(msg)
		for _, conn := range server.Connections() {
			conn.Request("DisplayMessage", "("+username+"): "+msg)
		}
		return
	}
	sendMessageToAll("(" + username + "): " + msg)
}

func doSetUsername(username string, connection *organics.Connection) {
	username = html.EscapeString(username)

	connection.Set("username", username)
	sendMessageToAll(username + " is now online.")
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
	server = organics.NewServer(sessionProvider)

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

	server.Handle(organics.Connect, doConnect)
	server.Handle("SetUsername", doSetUsername)
	server.Handle("Message", doMessage)

	http.Handle("/", http.FileServer(http.Dir("src/code.google.com/p/organics/examples/organics_chat/client")))
	http.Handle("/javascript/", http.FileServer(http.Dir("src/code.google.com/p/organics/")))

	http.Handle("/app", server)

	err = http.ListenAndServe(":80", nil)
	if err != nil {
		log.Fatal(err)
	}
}
