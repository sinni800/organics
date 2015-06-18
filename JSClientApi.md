**Table of contents
  * [Variables](JSClientApi#Variables.md)
    * [Organics.Debug](JSClientApi#Organics.Debug.md)
    * [Organics.WebSocketSupported](JSClientApi#Organics.WebSocketSupported.md)
  * [Constants](JSClientApi#Constants.md)
    * [Organics.Connect](JSClientApi#Organics.Connect.md)
    * [Organics.Disconnect](JSClientApi#Organics.Disconnect.md)
    * [Organics.ErrNotConnected](JSClientApi#Organics.ErrNotConnected.md)
  * [Functions](JSClientApi#Functions.md)
    * [Organics.Connection](JSClientApi#Organics.Connection.md)
      * Variables
        * [Connection.URL](JSClientApi#Connection.URL.md)
        * [Connection.TLS](JSClientApi#Connection.TLS.md)
        * [Connection.Timeout](JSClientApi#Connection.Timeout.md)
      * Functions
        * [Connection.Close](JSClientApi#Connection.Close.md)
        * [Connection.Connected](JSClientApi#Connection.Connected.md)
        * [Connection.Connect](JSClientApi#Connection.Connect.md)
        * [Connection.Handle](JSClientApi#Connection.Handle.md)
        * [Connection.Request](JSClientApi#Connection.Request.md)**

# Variables #
### Organics.Debug ###
  * **Definition:**
    * `Organics.Debug = false;`

  * **Description:**
    * If true debug messages will be written to console.log, if false, they will not.

### Organics.WebSocketSupported ###
  * **Definition:**
    * `Organics.WebSocketSupported = "WebSocket" in window || "MozWebSocket" in window;`

  * **Description:**
    * If true, WebSocket will be used to connect to this Connection's URL when Connect() is called. If false, Organics falls back to Long Polling, and connects to this Connection's URL via standard HTTP requests when Connect() is called.

  * **Notes:**
    * Forcing this to false (`Organics.WebSocketSupported = false;`) will cause the browser to force use of Long Polling.

# Constants #
### Organics.Connect ###
  * **Definition:**
    * `Organics.Connect = "A!B@C#D$E%F^G&H*I(J)";`

  * **Description:**
    * An special, unique, Request which does not get sent over the network, instead it is sent to an Request handler when the client has connected successfully.

### Organics.Disconnect ###
  * **Definition:**
    * `Organics.Disconnect = "a1b2c3d4e5f6g7h8i9j0";`

  * **Description:**
    * An special, unique, Request which does not get sent over the network, instead it is sent to an Request handler when the client is disconnected.

### Organics.ErrNotConnected ###
  * **Definition:**
    * `Organics.ErrNotConnected = "not currently connected to server";`

  * **Description:**
    * The exception that will be thrown should an Request() call be made while this Connection is not Connected().

# Functions #
## Organics.Connection ##
  * **Definition:**
    * `Organics.Connection = function(URL, TLS, Timeout) {...`}

  * **Description:**
    * Returns an Organics.Connection object, used for communicating with an URL on an remote Organics server.

  * **Parameters:**
    * Parameter: URL
      * Type: String
      * Description: URL to connect to, without leading protocol ("http://", "https://" etc.)
    * Parameter: TLS
      * Type: Boolean
      * Optional: (default value: false)
      * Description: Weather to use WS/HTTP (false) or WSS/HTTPS (true).
    * Parameter: Timeout
      * Type: Number
      * Optional: (default value, 15 seconds: 15000)
      * Description: Time in milliseconds to wait before an connection attempt is assumed to be "timed out".

  * **Notes:**
    * Be sure to use the new keyword, like so:
    * `var Remote = new Organics.Connection(...);`

### Connection.URL ###
  * **Definition:**
    * As passed into [Organics.Connection](JSClientApi#Organics.Connection.md).

### Connection.TLS ###
  * **Definition:**
    * As passed into [Organics.Connection](JSClientApi#Organics.Connection.md).

### Connection.Timeout ###
  * **Definition:**
    * As passed into [Organics.Connection](JSClientApi#Organics.Connection.md).

### Connection.Close ###
  * **Definition:**
    * `Organics.Close = function() {...`}

  * **Description:**
    * Closes this Connection, ensuring that future calls to [Connection.Connected](JSClientApi#Connection.Connected.md) will return false.

### Connection.Connected ###
  * **Definition:**
    * `Organics.Connected = function() {...`}

  * **Description:**
    * Returns an Boolean representing weather this Connection is currently connected.

### Connection.Connect ###
  * **Definition:**
    * `Organics.Connect = function() {...`}

  * **Description:**
    * Connects this Connection to this Connection's URL, or if this Connection is already connected, then calling this function is no-op.

### Connection.Handle ###
  * **Definition:**
    * `Organics.Handle = function(requestName, handler) {...`}

  * **Description:**
    * Specifies the handler function to use for any requests whose name is requestName.

  * **Parameters:**
    * Parameter: requestName
      * Type: Any valid JSON data type.
      * Description: Unique identifier that will represent the type of Request to handle.
    * Parameter: handler
      * Type: function (or null)
      * Description: If this parameter is null, the Request handler function currently associated with the requestName parameter, is deleted. If this parameter is an function, the Request handler function currently associated with the requestName parameter is replaced by this function.

  * **Notes:**
    * JavaScript has no static typing, but Organics partially requires it (See [Type Safety](Type_Safety.md)).

### Connection.Request ###
  * **Definition:**
    * `Organics.Request = function(requestName, args..., onComplete) {...`}

  * **Description:**
    * Sends an Request with the specified requestName, and args..., and (optionally) calling the onComplete parameter when the request has completed.

  * **Parameters:**
    * Parameter: requestName
      * Type: Any valid JSON data type.
      * Description: Unique identifier that will represent the type of Request to send.
    * Parameter: args
      * Type: Any number of any JSON-compatible data type.
      * Description: Arguments to be sent along with this Request, that will be fed into the server's request handler function registered for this requestName.
    * Parameter: onComplete
      * Type: Any function which takes any number of JSON-compatible arguments, and returns nothing.
      * Optional
      * Description: function to be called when the request has completed. Any values returned by the server's corresponding request handler, will be passed in to this function as arguments.

  * **Notes:**
    * JavaScript has no static typing, but Organics partially requires it (See [Type Safety](Type_Safety.md)).