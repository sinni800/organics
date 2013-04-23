// Copyright 2012 Lightpoke. All rights reserved.
// This source code is subject to the terms and
// conditions defined in the "License.txt" file.

// Enable console debug messages
//
//Organics.Debug = true;
//
// Disable WebSocket support -- forcing long-polling method (GET/POST)
// Note: This is really only for debugging.
//
//Organics.WebSocketSupported = false;


var logMessage = function(msg) {
	var e = document.getElementById("log");
	e.innerHTML = e.innerHTML + msg + "<br/>";
}


var SetSessionTime = function(time) {
	document.getElementById("sessionTime").innerHTML = time;
}

var SetConnectionTime = function(time) {
	document.getElementById("connectionTime").innerHTML = time;
}


// If we connect without issues then this function will be called.
var OnConnect = function() {
	logMessage("Connected to server");

	// Make an request, now that we're connected to the server, we'll pass in two strings, and the
	// server will combine them and send us an "Message" request, the last (and optional) parameter
	// is an callback function, which gets called once the server has handled our request fully, it
	// can be fed any arguments from the server, as well.
	//
	var msg1 = "After this message, our request ";
	var msg2 = "completed function will be called!";
	Remote.Request("GiveBrowserMessage", msg1, msg2, function(error) {
		// This function gets called once our "GiveBrowserMessage" request completes!
		//
		// The server will give us an "error" which will either be an string, or null. (Think Go!)
		if(error != null) {
			logMessage("Our request failed: " + error);
			return
		}
		logMessage("Our request completed!");

		// If that request went okay, we'll send another one!
		Remote.Request("GiveBrowserMessage", "Hello from the browser ", "again!", function(error) {
			// Again this function gets called as long as soon as this request completes.
			if(error != null) {
				logMessage("Our request failed:" + error);
				return
			}
			logMessage("Our request completed (again)!");
		})
	})
}

// If we disconnect from the server, this function will be called and given an (string) reason as
// to why on earth that might have happened.
var OnDisconnect = function(reason) {
	logMessage("Disconnected from server: " + reason + " (reconnecting in 10 seconds)");

	// Use an Javascript timeout to try and connect again later.
	setTimeout(function() {
		logMessage("Reconnecting...")
		// This will try connecting again (only if not already connected)
		Remote.Connect();
	}, 10 * 1000); // time in milliseconds
}

// Create an remote connection to an Organics server (your application server).
var Remote = new Organics.Connection("/app");

Remote.Handle(Organics.Connect, OnConnect);
Remote.Handle(Organics.Disconnect, OnDisconnect);

Remote.Handle("SetSessionTime", SetSessionTime);
Remote.Handle("SetConnectionTime", SetConnectionTime);

// Add an handler:
Remote.Handle("Message", logMessage);

// Remove the handler:
Remote.Handle("Message", null);

// Add it back:
Remote.Handle("Message", logMessage);

Remote.Connect();

