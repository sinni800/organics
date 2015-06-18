  * **Table of contents**
    * [Why?](General#Why?.md)
      * [The Problem](General#The_Problem.md)
      * [The Solution](General#The_Solution.md)
    * [Who?](General#Who?.md)
      * [Creators](General#Creators.md)
    * [What next?](General#What_next?.md)

# Why? #
### The Problem ###
In modern day web sites and applications, an lot of dynamic things need to happen between the web browser and server.

Typical solutions to this problem are AJAX (Asynchronous JavaScript and XML) which basically means doing lots of in-the-background requests to your web server.

This solution is _only one-way_ communication. Using the HTTP POST requests, web browsers can send information to web servers, and using HTTP GET requests, they can retrieve data from web servers.

But how can an web server send information to an web browser, without the web browser specifically asking for it first? Typically, they cannot.

![https://organics.googlecode.com/files/SaveTheWeb.jpg](https://organics.googlecode.com/files/SaveTheWeb.jpg)

### The Solution ###
An **_naive solution_** is to issue an GET request every few seconds, but this means it will take an little while for new, updated information to arrive to users, in addition **_it will also increase the load on your web servers significantly_**.

An better solution is to use [Long Polling](http://en.wikipedia.org/wiki/Push_technology#Long_polling) HTTP requests, in this scenario the web server waits, as long as it likes, anywhere from seconds, to hours, to centuries, etc, and the web server only responds to that original HTTP request when it actually wants to send something back to the browser.

In 2011, the IETF standardized an new WebSocket protocol (RFC 6455), that today most popular web browsers support. The WebSocket protocol allows for two-way communication, the exact problem we where originally trying to solve. The great thing is it's even more efficient both bandwidth and latency wise, than [Long Polling](http://en.wikipedia.org/wiki/Push_technology#Long_polling). The down side is that not all web browsers support it just yet, and even once they do, only some people update their web browser, other people might just think their browser works fine, why update? It's only broken on your site. (You know the users we are talking about.)

To solve this problem, [we](http://www.lightpoke.com) decided to create Organics.

![https://organics.googlecode.com/files/organics_logo_wide.png](https://organics.googlecode.com/files/organics_logo_wide.png)

Organics provides **_two-way_** server/browser communication, by providing an JavaScript client library and Go server library, which fully solve this problem in (what we believe to be) the nicest way possible.

The JavaScript client library detects weather an browser supports the newer, better, and more efficient WebSocket protocol, and if it does it will use the WebSocket protocol to implement two-way messaging, or if the browser lacks WebSocket support, it will automatically fall back to using Long Polling (described above).

The Go server library is an [net/http](http://golang.org/pkg/net/http/) compatible server library which uses both the standard [net/http](http://golang.org/pkg/net/http/) and [go.net/websocket](https://code.google.com/p/go/source/browse/?repo=net#hg%2Fwebsocket) libraries to handle both incoming Long Polling HTTP requests, as well as WebSocket connections, under an single URL in an easy fashion.

# Who? #
### Creators ###
Organics was created at [Lightpoke](http://www.lightpoke.com) in 2013 by _Stephen Gutekanst_ for use in various large scale dynamic web based applications.

# What next? #
  1. Check out [some examples](Examples.md).
  1. Get cozy with [the API](API.md).
  1. `go get code.google.com/p/organics`