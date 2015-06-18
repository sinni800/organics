  * **Table of contents**
    * [Important considerations](Bandwidth#Important_considerations.md)
      * [Application usage](Bandwidth#Application_usage.md)
      * [Size and rate relativity](Bandwidth#Size_and_rate_relativity.md)
      * [Caching](Bandwidth#Caching.md)
    * [Under the hood](Bandwidth#Under_the_hood.md)
      * [JSON](Bandwidth#JSON.md)
      * [Cookies](Bandwidth#Cookies.md)
      * [Web Socket Cookies](Bandwidth#Web_Socket_Cookies.md)

# Important considerations #

## Application usage ##
Organics only has an small overhead on top of the overhead you inherently get from TCP, HTTP, and/or WebSocket protocols. What this means is that if you're seeing large amounts of bandwidth usage, it's most likely your application doing it.

**_Ask yourself what bandwidth problems you will face in your application._** (_avoid [premature optimization](http://en.wikipedia.org/wiki/Program_optimization#Time_taken_for_optimization)._)

## Size and rate relativity ##
Organics only makes things easy to do -- there is no black magic compression included.

**_Think of the relationship between the size of data you are sending and the rate at which you are sending it._**

## Caching ##
Organics provides no means of caching anything, period. In our opinion, the best method of caching things is on an per-application basis.

**_Always ask yourself if you should cache something._**

# Under the hood #
There are some underlying things that occur in order to get two-way communication between Go and Javascript. These are described below.

### JSON ###
Organics uses JSON, even under the hood (see [Messages](Messages.md)). So all of your data will be restricted to JSON data types. Since Organics is all about communicating between JavaScript and Go, we think this makes sense.

We are however open to your objections to this, and if you have some valid reason for wanting something else then please [create an issue](https://code.google.com/p/organics/issues/entry).

### Cookies ###
Browser cookies are used to identify user sessions. This means an few things, users will need to have cookies enabled(1), the session cookie is ~44 bytes(_2_)(_3_).

  1. Most modern websites require cookies to be enabled, if an user was to disable cookies, most modern websites that they visit daily would break horribly. Turning off cookies is the most grave mistake an user could make aside from uninstalling their browser or unplugging their LAN cable.
  1. Cookies are sent with all HTTP requests (both Long Polling and Request HTTP POST requests), whereas with WebSockets they will only be sent for the initial connection-establish HTTP requests, therefor less bandwidth is used from WebSocket connections.
  1. Long Polling clients require an additional connection cookies, of the same size as the session cookie, to be sent along with each request. This cookies allows us to emulate WebSocket's connection-based behavior, in addition to also providing anti-CSRF protection.

### Web Socket Cookies ###
The WebSocket library we use on the Go server side ([go.net/websocket](https://code.google.com/p/go/source/browse/?repo=net#hg%2Fwebsocket)) doesn't provide any way to send cookies back on the initial WebSocket upgrade request.

  * Because of this, we have to send an single additional HTTP request to establish an Organics connection.

  * In order to fix this, go.net/websocket must first be fixed. (see [issue 2](https://code.google.com/p/organics/issues/detail?id=2))