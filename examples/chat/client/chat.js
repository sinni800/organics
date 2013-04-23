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


var addEventListener = function(element, event, handler) {
	if(document.addEventListener) {
		element.addEventListener(event, handler, false);
	} else {
		element.attachEvent("on" + event, handler);
	}
}

var username = "";

while(username == "" || username == null) {
	username = prompt("Enter an username:");
}

var popSound = document.createElement('audio');
popSound.setAttribute('src', 'pop.wav');

window.onload = function() {
	var chatInput = document.getElementById('chat_input');
	chatInput.focus();

	addEventListener(chatInput, "blur", function() {
		chatInput.focus();
	})

	window.onblur = function() {
		window.isFocused = false;
	};
	window.onfocus = function() {
		window.isFocused = true;
	};

	addEventListener(chatInput, "keypress", function(e) {
		if(!e) {
			var e = window.event;
		}

		if(e.keyCode == 13 || e.which == 13) {
			if(e.preventDefault) {
				e.preventDefault();
			} else {
				e.returnValue = false;
			}

			Remote.Request("Message", chatInput.value);
			chatInput.value = "";
		}
	});
};

var DisplayMessage = function(msg) {
	var e = document.getElementById("messages");
	e.innerHTML = e.innerHTML + msg + "\r\n";
	e.scrollTop = e.scrollHeight;
	if(!window.isFocused) {
		popSound.play();
	}
}

// If we disconnect from the server, this function will be called and given an (string) reason as
// to why on earth that might have happened.
var OnDisconnect = function(reason) {
	DisplayMessage("Disconnected from server: " + reason + " (reconnecting in 10 seconds)");

	// Use an Javascript timeout to try and connect again later.
	setTimeout(function() {
		DisplayMessage("Reconnecting...")
		// This will try connecting again (only if not already connected)
		Remote.Connect();
	}, 10 * 1000); // time in milliseconds
}

// Create an remote connection to an Organics server (your application server).
var Remote = new Organics.Connection("/app");

Remote.Handle(Organics.Connect, function() {
	DisplayMessage("Connected to server");
	Remote.Request("SetUsername", username);
});

Remote.Handle(Organics.Disconnect, OnDisconnect);
Remote.Handle("DisplayMessage", DisplayMessage);

Remote.Connect();

