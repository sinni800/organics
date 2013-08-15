// Copyright 2012 Lightpoke. All rights reserved.
// This source code is subject to the terms and
// conditions defined in the "License.txt" file.


////////////////////////////////////////////////////////////////////////////////
// File: json2.js                                                             //
// Location: https://github.com/douglascrockford/JSON-js/blob/master/json2.js //
// License: Public Domain                                                     //
//                                                                            //
// Attribution: (Public Domain) Douglas Crockford - douglas@crockford.com     //
////////////////////////////////////////////////////////////////////////////////
/*
    json2.js
    2012-10-08

    Public Domain.

    NO WARRANTY EXPRESSED OR IMPLIED. USE AT YOUR OWN RISK.

    See http://www.JSON.org/js.html


    This code should be minified before deployment.
    See http://javascript.crockford.com/jsmin.html

    USE YOUR OWN COPY. IT IS EXTREMELY UNWISE TO LOAD CODE FROM SERVERS YOU DO
    NOT CONTROL.


    This file creates a global JSON object containing two methods: stringify
    and parse.

        JSON.stringify(value, replacer, space)
            value       any JavaScript value, usually an object or array.

            replacer    an optional parameter that determines how object
                        values are stringified for objects. It can be a
                        function or an array of strings.

            space       an optional parameter that specifies the indentation
                        of nested structures. If it is omitted, the text will
                        be packed without extra whitespace. If it is a number,
                        it will specify the number of spaces to indent at each
                        level. If it is a string (such as '\t' or '&nbsp;'),
                        it contains the characters used to indent at each level.

            This method produces a JSON text from a JavaScript value.

            When an object value is found, if the object contains a toJSON
            method, its toJSON method will be called and the result will be
            stringified. A toJSON method does not serialize: it returns the
            value represented by the name/value pair that should be serialized,
            or undefined if nothing should be serialized. The toJSON method
            will be passed the key associated with the value, and this will be
            bound to the value

            For example, this would serialize Dates as ISO strings.

                Date.prototype.toJSON = function (key) {
                    function f(n) {
                        // Format integers to have at least two digits.
                        return n < 10 ? '0' + n : n;
                    }

                    return this.getUTCFullYear()   + '-' +
                         f(this.getUTCMonth() + 1) + '-' +
                         f(this.getUTCDate())      + 'T' +
                         f(this.getUTCHours())     + ':' +
                         f(this.getUTCMinutes())   + ':' +
                         f(this.getUTCSeconds())   + 'Z';
                };

            You can provide an optional replacer method. It will be passed the
            key and value of each member, with this bound to the containing
            object. The value that is returned from your method will be
            serialized. If your method returns undefined, then the member will
            be excluded from the serialization.

            If the replacer parameter is an array of strings, then it will be
            used to select the members to be serialized. It filters the results
            such that only members with keys listed in the replacer array are
            stringified.

            Values that do not have JSON representations, such as undefined or
            functions, will not be serialized. Such values in objects will be
            dropped; in arrays they will be replaced with null. You can use
            a replacer function to replace those with JSON values.
            JSON.stringify(undefined) returns undefined.

            The optional space parameter produces a stringification of the
            value that is filled with line breaks and indentation to make it
            easier to read.

            If the space parameter is a non-empty string, then that string will
            be used for indentation. If the space parameter is a number, then
            the indentation will be that many spaces.

            Example:

            text = JSON.stringify(['e', {pluribus: 'unum'}]);
            // text is '["e",{"pluribus":"unum"}]'


            text = JSON.stringify(['e', {pluribus: 'unum'}], null, '\t');
            // text is '[\n\t"e",\n\t{\n\t\t"pluribus": "unum"\n\t}\n]'

            text = JSON.stringify([new Date()], function (key, value) {
                return this[key] instanceof Date ?
                    'Date(' + this[key] + ')' : value;
            });
            // text is '["Date(---current time---)"]'


        JSON.parse(text, reviver)
            This method parses a JSON text to produce an object or array.
            It can throw a SyntaxError exception.

            The optional reviver parameter is a function that can filter and
            transform the results. It receives each of the keys and values,
            and its return value is used instead of the original value.
            If it returns what it received, then the structure is not modified.
            If it returns undefined then the member is deleted.

            Example:

            // Parse the text. Values that look like ISO date strings will
            // be converted to Date objects.

            myData = JSON.parse(text, function (key, value) {
                var a;
                if (typeof value === 'string') {
                    a =
/^(\d{4})-(\d{2})-(\d{2})T(\d{2}):(\d{2}):(\d{2}(?:\.\d*)?)Z$/.exec(value);
                    if (a) {
                        return new Date(Date.UTC(+a[1], +a[2] - 1, +a[3], +a[4],
                            +a[5], +a[6]));
                    }
                }
                return value;
            });

            myData = JSON.parse('["Date(09/09/2001)"]', function (key, value) {
                var d;
                if (typeof value === 'string' &&
                        value.slice(0, 5) === 'Date(' &&
                        value.slice(-1) === ')') {
                    d = new Date(value.slice(5, -1));
                    if (d) {
                        return d;
                    }
                }
                return value;
            });


    This is a reference implementation. You are free to copy, modify, or
    redistribute.
*/

/*jslint evil: true, regexp: true */

/*members "", "\b", "\t", "\n", "\f", "\r", "\"", JSON, "\\", apply,
    call, charCodeAt, getUTCDate, getUTCFullYear, getUTCHours,
    getUTCMinutes, getUTCMonth, getUTCSeconds, hasOwnProperty, join,
    lastIndex, length, parse, prototype, push, replace, slice, stringify,
    test, toJSON, toString, valueOf
*/


// Create a JSON object only if one does not already exist. We create the
// methods in a closure to avoid creating global variables.

if (typeof JSON !== 'object') {
    JSON = {};
}

(function () {
    'use strict';

    function f(n) {
        // Format integers to have at least two digits.
        return n < 10 ? '0' + n : n;
    }

    if (typeof Date.prototype.toJSON !== 'function') {

        Date.prototype.toJSON = function (key) {

            return isFinite(this.valueOf())
                ? this.getUTCFullYear()     + '-' +
                    f(this.getUTCMonth() + 1) + '-' +
                    f(this.getUTCDate())      + 'T' +
                    f(this.getUTCHours())     + ':' +
                    f(this.getUTCMinutes())   + ':' +
                    f(this.getUTCSeconds())   + 'Z'
                : null;
        };

        String.prototype.toJSON      =
            Number.prototype.toJSON  =
            Boolean.prototype.toJSON = function (key) {
                return this.valueOf();
            };
    }

    var cx = /[\u0000\u00ad\u0600-\u0604\u070f\u17b4\u17b5\u200c-\u200f\u2028-\u202f\u2060-\u206f\ufeff\ufff0-\uffff]/g,
        escapable = /[\\\"\x00-\x1f\x7f-\x9f\u00ad\u0600-\u0604\u070f\u17b4\u17b5\u200c-\u200f\u2028-\u202f\u2060-\u206f\ufeff\ufff0-\uffff]/g,
        gap,
        indent,
        meta = {    // table of character substitutions
            '\b': '\\b',
            '\t': '\\t',
            '\n': '\\n',
            '\f': '\\f',
            '\r': '\\r',
            '"' : '\\"',
            '\\': '\\\\'
        },
        rep;


    function quote(string) {

// If the string contains no control characters, no quote characters, and no
// backslash characters, then we can safely slap some quotes around it.
// Otherwise we must also replace the offending characters with safe escape
// sequences.

        escapable.lastIndex = 0;
        return escapable.test(string) ? '"' + string.replace(escapable, function (a) {
            var c = meta[a];
            return typeof c === 'string'
                ? c
                : '\\u' + ('0000' + a.charCodeAt(0).toString(16)).slice(-4);
        }) + '"' : '"' + string + '"';
    }


    function str(key, holder) {

// Produce a string from holder[key].

        var i,          // The loop counter.
            k,          // The member key.
            v,          // The member value.
            length,
            mind = gap,
            partial,
            value = holder[key];

// If the value has a toJSON method, call it to obtain a replacement value.

        if (value && typeof value === 'object' &&
                typeof value.toJSON === 'function') {
            value = value.toJSON(key);
        }

// If we were called with a replacer function, then call the replacer to
// obtain a replacement value.

        if (typeof rep === 'function') {
            value = rep.call(holder, key, value);
        }

// What happens next depends on the value's type.

        switch (typeof value) {
        case 'string':
            return quote(value);

        case 'number':

// JSON numbers must be finite. Encode non-finite numbers as null.

            return isFinite(value) ? String(value) : 'null';

        case 'boolean':
        case 'null':

// If the value is a boolean or null, convert it to a string. Note:
// typeof null does not produce 'null'. The case is included here in
// the remote chance that this gets fixed someday.

            return String(value);

// If the type is 'object', we might be dealing with an object or an array or
// null.

        case 'object':

// Due to a specification blunder in ECMAScript, typeof null is 'object',
// so watch out for that case.

            if (!value) {
                return 'null';
            }

// Make an array to hold the partial results of stringifying this object value.

            gap += indent;
            partial = [];

// Is the value an array?

            if (Object.prototype.toString.apply(value) === '[object Array]') {

// The value is an array. Stringify every element. Use null as a placeholder
// for non-JSON values.

                length = value.length;
                for (i = 0; i < length; i += 1) {
                    partial[i] = str(i, value) || 'null';
                }

// Join all of the elements together, separated with commas, and wrap them in
// brackets.

                v = partial.length === 0
                    ? '[]'
                    : gap
                    ? '[\n' + gap + partial.join(',\n' + gap) + '\n' + mind + ']'
                    : '[' + partial.join(',') + ']';
                gap = mind;
                return v;
            }

// If the replacer is an array, use it to select the members to be stringified.

            if (rep && typeof rep === 'object') {
                length = rep.length;
                for (i = 0; i < length; i += 1) {
                    if (typeof rep[i] === 'string') {
                        k = rep[i];
                        v = str(k, value);
                        if (v) {
                            partial.push(quote(k) + (gap ? ': ' : ':') + v);
                        }
                    }
                }
            } else {

// Otherwise, iterate through all of the keys in the object.

                for (k in value) {
                    if (Object.prototype.hasOwnProperty.call(value, k)) {
                        v = str(k, value);
                        if (v) {
                            partial.push(quote(k) + (gap ? ': ' : ':') + v);
                        }
                    }
                }
            }

// Join all of the member texts together, separated with commas,
// and wrap them in braces.

            v = partial.length === 0
                ? '{}'
                : gap
                ? '{\n' + gap + partial.join(',\n' + gap) + '\n' + mind + '}'
                : '{' + partial.join(',') + '}';
            gap = mind;
            return v;
        }
    }

// If the JSON object does not yet have a stringify method, give it one.

    if (typeof JSON.stringify !== 'function') {
        JSON.stringify = function (value, replacer, space) {

// The stringify method takes a value and an optional replacer, and an optional
// space parameter, and returns a JSON text. The replacer can be a function
// that can replace values, or an array of strings that will select the keys.
// A default replacer method can be provided. Use of the space parameter can
// produce text that is more easily readable.

            var i;
            gap = '';
            indent = '';

// If the space parameter is a number, make an indent string containing that
// many spaces.

            if (typeof space === 'number') {
                for (i = 0; i < space; i += 1) {
                    indent += ' ';
                }

// If the space parameter is a string, it will be used as the indent string.

            } else if (typeof space === 'string') {
                indent = space;
            }

// If there is a replacer, it must be a function or an array.
// Otherwise, throw an error.

            rep = replacer;
            if (replacer && typeof replacer !== 'function' &&
                    (typeof replacer !== 'object' ||
                    typeof replacer.length !== 'number')) {
                throw new Error('JSON.stringify');
            }

// Make a fake root object containing our value under the key of ''.
// Return the result of stringifying the value.

            return str('', {'': value});
        };
    }


// If the JSON object does not yet have a parse method, give it one.

    if (typeof JSON.parse !== 'function') {
        JSON.parse = function (text, reviver) {

// The parse method takes a text and an optional reviver function, and returns
// a JavaScript value if the text is a valid JSON text.

            var j;

            function walk(holder, key) {

// The walk method is used to recursively walk the resulting structure so
// that modifications can be made.

                var k, v, value = holder[key];
                if (value && typeof value === 'object') {
                    for (k in value) {
                        if (Object.prototype.hasOwnProperty.call(value, k)) {
                            v = walk(value, k);
                            if (v !== undefined) {
                                value[k] = v;
                            } else {
                                delete value[k];
                            }
                        }
                    }
                }
                return reviver.call(holder, key, value);
            }


// Parsing happens in four stages. In the first stage, we replace certain
// Unicode characters with escape sequences. JavaScript handles many characters
// incorrectly, either silently deleting them, or treating them as line endings.

            text = String(text);
            cx.lastIndex = 0;
            if (cx.test(text)) {
                text = text.replace(cx, function (a) {
                    return '\\u' +
                        ('0000' + a.charCodeAt(0).toString(16)).slice(-4);
                });
            }

// In the second stage, we run the text against regular expressions that look
// for non-JSON patterns. We are especially concerned with '()' and 'new'
// because they can cause invocation, and '=' because it can cause mutation.
// But just to be safe, we want to reject all unexpected forms.

// We split the second stage into 4 regexp operations in order to work around
// crippling inefficiencies in IE's and Safari's regexp engines. First we
// replace the JSON backslash pairs with '@' (a non-JSON character). Second, we
// replace all simple value tokens with ']' characters. Third, we delete all
// open brackets that follow a colon or comma or that begin the text. Finally,
// we look to see that the remaining characters are only whitespace or ']' or
// ',' or ':' or '{' or '}'. If that is so, then the text is safe for eval.

            if (/^[\],:{}\s]*$/
                    .test(text.replace(/\\(?:["\\\/bfnrt]|u[0-9a-fA-F]{4})/g, '@')
                        .replace(/"[^"\\\n\r]*"|true|false|null|-?\d+(?:\.\d*)?(?:[eE][+\-]?\d+)?/g, ']')
                        .replace(/(?:^|:|,)(?:\s*\[)+/g, ''))) {

// In the third stage we use the eval function to compile the text into a
// JavaScript structure. The '{' operator is subject to a syntactic ambiguity
// in JavaScript: it can begin a block or an object literal. We wrap the text
// in parens to eliminate the ambiguity.

                j = eval('(' + text + ')');

// In the optional fourth stage, we recursively walk the new structure, passing
// each name/value pair to a reviver function for possible transformation.

                return typeof reviver === 'function'
                    ? walk({'': j}, '')
                    : j;
            }

// If the text is not JSON parseable, then a SyntaxError is thrown.

            throw new SyntaxError('JSON.parse');
        };
    }
}());
///////////////////////////////////////////////////////////////////////////////
// Endfile: json2.js                                                         //
///////////////////////////////////////////////////////////////////////////////

///////////////////////////////////////////////////////////////////////////////
// *** End of in file imports                                                //
///////////////////////////////////////////////////////////////////////////////






var Organics = new function() {
	this.Debug = false;
	this.WebSocketSupported = "WebSocket" in window || "MozWebSocket" in window;

	// Special 'unique' messages.
	this.Connect = "A!B@C#D$E%F^G&H*I(J)";
	this.Disconnect = "a1b2c3d4e5f6g7h8i9j0";

	// Constants (save extra bytes by making them acronyms)
	this.__rtLongPollEstablishConnection  = "lpec"; // long-poll-establish-connection
	this.__rtLongPoll                     = "lp";   // long-poll
	this.__rtMessage                      = "m";    // message

	this.ErrNotConnected = "not currently connected to server";

//	this.ErrorDisconnected = "disconnected from server or connection closed";
//	this.ErrorConnectionFailed = "unable to connect to server";
//	this.ErrorBadData = "server sent corrupted, bad, or unreadable data";

	this.__addEventListener = function(element, event, handler) {
		if(document.addEventListener) {
			element.addEventListener(event, handler, false);
		} else {
			element.attachEvent("on" + event, handler);
		}
	}

	this.__isPageBeingRefreshed = false;
	this.__addEventListener(window, "beforeunload", function() {
		Organics.__isPageBeingRefreshed = true;
	})

	this.__Log = function(msg) {
		if(Organics.Debug) {
			if(window.console) {
				console.log("Organics: " + msg);
			}
		}
	}

	if(this.WebSocketSupported) {
		var ws = null;
		if("WebSocket" in window) {
			ws = window.WebSocket;
		} else {
			ws = window.MozWebSocket;
		}
		if(ws.CLOSED <= 2) {
			// < hybi 07
			// Old WebSocket versions not supported (IOS, etc).
			this.WebSocketSupported = false;

			if(window.console) {
				console.log("Organics: Browser only supports < hybi 07 WebSockets; falling back to long polling.");
			}
		}
	}

	this.__StackTrace = function() {
		var e = new Error("StackTrace");
		var stack = e.stack.replace(/^[^\(]+?[\n$]/gm, '')
			.replace(/^\s+at\s+/gm, '')
			.replace(/^Object.<anonymous>\s*\(/gm, '{anonymous}()@')
			.split('\n');
		return stack;
	}

	this.__LogStackTrace = function() {
		if(Organics.Debug) {
			var lines = Organics.__StackTrace();
			for(var i = 0; i < lines.length; i++) {
				if(window.console) {
					console.log(lines[i]);
				}
			}
			if(window.console) {
				console.log("\n");
			}
		}
	}

	// Tells weather (str) starts with (w)
	this.__StringStartsWith = function(str, w) {
		return str.slice(0, w.length) == w;
	}

	// Tells weather (str) ends with (w)
	this.__StringEndsWith = function(str, w) {
		return str.slice(-w.length) == w;
	}

	// Returns an new XMLHttpRequest object for this browser
	this.__xhr = function() {
		if(window.XMLHttpRequest) {
			return new XMLHttpRequest();

		} else if(window.createRequest) {
			// ICEbrowser uses window.createRequest()
			// http://support.icesoft.com/jive/entry.jspa?entryID=471&categoryID=21
			return window.createRequest();

		} else if(window.ActiveXObject) {
			// Older versions of IE use ActiveXObject's, so try and get one of those now.
			var modes = ["Msxml3.XMLHTTP", "Msxml2.XMLHTTP.6.0", "Msxml2.XMLHTTP.3.0", "Msxml2.XMLHTTP", "Microsoft.XMLHTTP"];
			for(var i = 0; i < modes.length; i++) {
				try{
					return new ActiveXObject(modes[i]);
				} catch(e) {
					// Ignore errors creating activex objects, we'll fallback to
					// another activex version defined above.
				}
			}
		}
	}

	this.__translateWindowsError = function(code) {
		// ActiveX error codes.. http://msdn.microsoft.com/en-us/library/aa383770%28VS.85%29.aspx
		switch(code) {
			case 12001:
				return "Internet handle could not be generated at this time.";
			case 12002:
				return "Request timed out.";
			case 12004:
				return "An internal internet error has occured.";
			case 12005:
				return "URL is invalid.";
			case 12006:
				return "URL scheme could not be recognized or is not supported.";
			case 12007:
				return "Server name could not be resolved.";
			case 12008:
				return "The requested protocol could not be found.";
			case 12013:
				return "Failed to log on to FTP server, user name is incorrect.";
			case 12014:
				return "Failed to log on to FTP server, password is incorrect.";
			case 12015:
				return "Failed to coonect and log on to FTP server.";
			case 12023:
				return "Direct network access cannot be made at this time.";
			case 12029:
				return "Unable to connect to server.";
			case 12030:
				return "Connection terminated.";
			case 12031:
				return "Connection reset.";
			case 12037:
				return "Date on server's SSL certificate is bad expired.";
			case 12038:
				return "Host name on server's SSL certificate is incorrect.";
			case 12040:
				return "Moving from non-SSL to SSL due to redirect.";
			case 12042:
				return "Attempt to post and change data on server that is not secure.";
			case 12043:
				return "Attempt to post data on server that is not secure.";
			case 12110:
				return "FTP operation failed, operation is already in progress.";
			case 12111:
				return "FTP operation failed, the session was aborted.";
			case 12150:
				return "Requested HTTP header could not be found.";
			case 12151:
				return "Server returned no HTTP headers.";
			case 12152:
				return "Server response could not be parsed.";
			case 12156:
				return "HTTP redirect failed, scheme changed or all attempts failed.";
		}
	}

	// Performs ajax request on our behalf, do not use externally.
	this.__ajax = function(url, method, handlers, data, timeout, headers) {
		var xhr = new Organics.__xhr();
		if(!xhr) {
	 		handlers["error"](null, "Browser has no support for AJAX");
			return
		}

		var timer = null;
		if(timeout) {
			timer = setTimeout(function() {
				xhr.abort();
				handlers["error"](xhr, "timed out");
			}, timeout);
		}

		xhr.onreadystatechange = function() {
			if(xhr.readyState == 4) {
				if(timer) {
					clearTimer(timer);
				}
				if(xhr.status == 200) {
					handlers["complete"](xhr);
				} else {
					var windowsErr = Organics.__translateWindowsError(xhr.status);

					if(Organics.__isPageBeingRefreshed && xhr.status == 0) {
						// No actual error
						return;
					} else if(xhr.status == 0) {
						handlers["error"](xhr, "network error")
					} else if(windowsErr != null) {
						handlers["error"](xhr, xhr.status + " " + windowsErr);
					} else {
				 		handlers["error"](xhr, xhr.status + " " + xhr.statusText);
					}
				}
			}
		}

		//var timeStampUrl = url + ((/\?/).test(url) ? "&" : "?") + (new Date()).getTime();
		try{
			xhr.open(method, url, true);
		} catch(actualError) {
			handlers["error"](xhr, "XMLHttpRequest.open failed: " + actualError);
			return;
		}

		if(headers == null) {
			headers = {};
		}
		if(headers["Content-Type"] == null) {
			headers["Content-Type"] = "text/plain;charset=UTF-8";
		}

		if(headers) {
			for(var key in headers) {
				xhr.setRequestHeader(key, headers[key]);
			}
		}


		try{
			xhr.send(data);
		} catch(actualError) {
			handlers["error"](xhr, "XMLHttpRequest.send failed: " + actualError);
			return;
		}
	}

	// parameters
	//     URL
	//         (description): URL to connect to, without starting method ('http://' etc)
	//         (type):        string
	//         (optional):    no
	//
	//     TLS
	//         (description): weather to use WS/HTTP (false) or WSS/HTTPS (true)
	//         (type):        boolean
	//         (optional):    yes
	//         (default):     false
	//
	//     Timeout
	//         (description): Time in milliseconds to wait before an connection attempt is guessed
	//                        to be 'timed out'
	//
	//         (type):        number
	//         (optional):    yes
	//         (default):     15000 (15 seconds)
	//
	this.Connection = function(URL, TLS, Timeout) {
		var self = this;

		self.__requestCounter = -1;
		self.__requestHandlers = {};

		self.__URL = URL;
		if(typeof self.__URL != "string") {
			throw TypeError("URL parameter must be an string!");
		}
		self.URL = self.__URL;

		self.TLS = TLS;
		if(self.TLS == null) {
			self.TLS = false;
		}
		if(typeof self.TLS != "boolean") {
			throw TypeError("TLS optional parameter must be bool!");
		}

		self.Timeout = Timeout;
		if(self.Timeout == null) {
			self.Timeout = 15 * 1000; // milliseconds
		}
		self.Timeout = Timeout;

		self.__handlers = {};
		self.__connected = false;
		self.__connecting = false;

		// If they use /something then it should be relative to document.domain
		if(Organics.__StringStartsWith(self.__URL, "/")) {
			self.__URL = document.location.host + self.__URL;
		}

		// Strip current method from URL in case they put it in there on accident
		var methodWs    = "ws://";
		var methodWss   = "wss://";
		var methodHttp  = "http://";
		var methodHttps = "https://";
		if(Organics.__StringStartsWith(self.__URL, methodWs)) {
			self.__URL = self.__URL.slice(methodWs.length)
		} else if(Organics.__StringStartsWith(self.__URL, methodWss)) {
			self.__URL = self.__URL.slice(methodWss.length)
		} else if(Organics.__StringStartsWith(self.__URL, methodHttp)) {
			self.__URL = self.__URL.slice(methodHttp.length)
		} else if(Organics.__StringStartsWith(self.__URL, methodHttps)) {
			self.__URL = self.__URL.slice(methodHttps.length)
		}


		// Create URL with proper method depending on WebSocket support in browser, and TLS option.
		if(self.TLS) {
			self.__HTTP_URL = methodHttps + self.__URL;
		} else {
			self.__HTTP_URL = methodHttp + self.__URL;
		}

		if(Organics.WebSocketSupported) {
			if(self.TLS) {
				self.__URL = methodWss + self.__URL;
			} else {
				self.__URL = methodWs + self.__URL;
			}
		} else {
			if(self.TLS) {
				self.__URL = methodHttps + self.__URL;
			} else {
				self.__URL = methodHttp + self.__URL;
			}
		}

		self.__logMessage = function(msg) {
			Organics.__Log("(" + self.__URL + "): " + msg);
		}

		self.__handleDisconnect = function(err, later) {
			if(self.__connected == true || self.__connecting == true) {
				self.__connected = false;
				self.__connecting = false;

				self.__logMessage("-> Disconnected: \"" + err + "\"");
				var fn = self.__handlers[Organics.Disconnect];
				if(fn) {
					fn(err)
				}
			}
			self.__connected = false;
			self.__connecting = false;
		}

		self.__handleConnect = function() {
			self.__logMessage("-> Connected");
			var fn = self.__handlers[Organics.Connect];
			if(fn) {
				fn()
			}
		}
	}

	// Connected tells weather this Connection is currently connected (boolean).
	this.Connection.prototype.Connected = function() {
		var self = this;

		if(self.__connecting == true) {
			return false;
		}
		return self.__connected;
	}

	// Close closes this connection, causing Connected() to return false.
	this.Connection.prototype.Close = function() {
		var self = this;

		if(self.__connected == true) {
			self.__connected = false;
			if(Organics.WebSocketSupported) {
				self.__webSocket.close()
			}
		}
	}

	// Connect tries to connect this Connection if it is currently not connected.
	this.Connection.prototype.Connect = function() {
		var self = this;

		if(Organics.WebSocketSupported) {
			self.__logMessage("WebSocket is supported");
		} else {
			self.__logMessage("No WebSocket support; using long polling");
		}

		if(self.__connected || self.__connecting) {
			return;
		}
		self.__logMessage("-> Connect()");
		self.__connecting = true;

		var doConnect = function() {
			if(Organics.WebSocketSupported) {
				self.__connectWebSocket()
				return
			}

			// Perform the create session request
			Organics.__ajax(self.__HTTP_URL, "POST", {
				complete: function(xhr) {
					self.__connectionId = xhr.responseText;
					self.__logMessage("-> Create session request successful: connected to server");
					self.__connected = true;
					self.__connecting = false;
					setTimeout(function() {
						self.__doLongPolling();
					}, 0);
					self.__handleConnect();
				},
				error: function(xhr, msg) {
					// Handle the disconnection error, use an delay of 0.25 seconds to ensure that
					// an error handler is assigned before the error is dispatched
					self.__handleDisconnect("Create session request failed (" + msg + ")");
				}
			}, null, self.Timeout, {
				// This informs the server this is an session creation request, and they it should
				// respond immedietly after ensuring we have an request.
				"X-Organics-Req": Organics.__rtLongPollEstablishConnection
			});
		};


		if(!Organics.__hasAlreadyLoaded) {
			Organics.__addEventListener(window, "load", doConnect);
			Organics.__hasAlreadyLoaded = true;
		} else {
			doConnect();
		}
	}

	this.Connection.prototype.__handleMessage = function(msg) {
		var self = this;

		var doClose = function() {
			if(Organics.webSocketSupported) {
				self.__webSocket.close();
			}
		}

		if(msg.length > 0) {
			// It's an request OR an response

			// Firstly, it must be valid JSON data, so try to parse it as JSON first.
			try{
				var json = JSON.parse(msg);
			} catch(parseError) {
				self.__handleDisconnect("Server sent bad JSON data: " + parseError);
				doClose();
				return;
			}

			// If we made it this far, it's at least JSON. Now check if it's an array, it must be.
			if(json.length == 3) {
				// It's an request: [id, requestName, args]
				var id = json[0];
				var requestName = json[1];
				var args = json[2];

				var responseArgs = null;
				var fn = self.__handlers[requestName];
				if(fn) {
					try{
						var responseArgs = fn.apply(undefined, args);
						// Function could return null, if it does, it's "no args"
						if(responseArgs == null) {
							responseArgs = [];
						}
					} catch(e) {
						Organics.__Log("Request handler exception:\n" + e);
						return;
					}
				} else {
					Organics.__Log("Ignoring request \"" + requestName + "\", no handler.");
					return
				}

				if(id !== -1) {
					try{
						return JSON.stringify([id, responseArgs]);
					} catch(e) {
						Organics.__Log("Error encoding response:\n" + e);
						return;
					}
				}

			} else if(json.length == 2 || json.length == 1) {
				// It's an response: [id, args], or [id]
				var id = json[0];
				if(json.length == 1) {
					var args = [];
				} else {
					var args = json[1];
				}

				var onComplete = self.__requestHandlers[id];
				if(onComplete) {
					try{
						onComplete.apply(undefined, args);
					} catch(e) {
						Organics.__Log("Request handler onComplete exception:\n" + e);
						return;
					}
				} else {
					Organics.__Log("Got invalid response; id is invalid; ignored.")
					return
				}

			} else {
				self.__handleDisconnect("Server sent bad JSON data: Must be array of length 3");
				doClose();
				return;
			}

		} else {
			// It's an ping
			if(Organics.WebSocketSupported) {
				// For WebSockets, an ping is an empty message, and we send an empty message back
				// to respond to their ping.
				return "";

			} else {
				// For long-polling, all we need to do is request again ASAP, to respond to their
				// ping (which means just return here).
				return;
			}
		}
	}

	this.Connection.prototype.__connectWebSocket = function() {
		var self = this;

		if(window.WebSocket) {
			self.__webSocket = new WebSocket(self.__URL);
		} else if(window.MozWebSocket) {
			self.__webSocket = new MozWebSocket(self.__URL);
		}

		self.__webSocket.onopen = function(evt) {
			self.__connected = true;
			self.__connecting = false;
			self.__handleConnect();
		}
		self.__webSocket.onclose = function(evt) {
			self.__handleDisconnect("connection closed");
		}

		self.__webSocket.onmessage = function(evt) {
			var response = self.__handleMessage(evt.data);
			if(response != null) {
				self.__webSocket.send(response);
			}
		}
		self.__webSocket.onerror = function(evt) {
			self.__handleDisconnect("disconnected" + evt);
		}
	}

	this.Connection.prototype.__doLongPolling = function() {
		var self = this;

		Organics.__ajax(self.__HTTP_URL, "POST", {
			complete: function(xhr) {
				// Handle the data
				var response = self.__handleMessage(xhr.responseText);
				if(response != null) {
					Organics.__ajax(self.__HTTP_URL, "POST", {
						complete: function(xhr) {
						},
						error: function(xhr, msg) {
							self.__handleDisconnect("POST request failed (" + msg + ")", 0);
						}
					}, response, null, {
						"X-Organics-Req": Organics.__rtMessage,
						"X-Organics-Conn": self.__connectionId
					});
				}

				// Go back to long-polling again
				setTimeout(function() {
					self.__doLongPolling();
				}, 0);
			},

			error: function(xhr, msg) {
				self.__handleDisconnect("long-polling request failed (" + msg + ")", 0);
			}
		}, null, null, {
			"X-Organics-Req": Organics.__rtLongPoll,
			"X-Organics-Conn": self.__connectionId
		});
	}

	// Request makes an request to the server, using jsonData, and (optionally) calling the
	// OnComplete function parameter.
	//
	// This function fails if the jsonData parameter is invalid JSON data for the global
	// JSON.stringify function (provided by json2.js), in which case an exception is thrown.
	//
	// This function fails if this Connection is currently not connected; in which case
	// an Organics.ErrNotConnected exception is thrown.
	//
	// This function fails if the (optional) OnComplete parameter is not an function, and an
	// TypeError exception is thrown.
	//
	this.Connection.prototype.Request = function() {
		var self = this;

		var args = Array.prototype.slice.call(arguments);

		if(args.length == 0) {
			return;
		}

		var requestName = args[0];

		if(args.length > 1) {
			var onComplete = args[args.length - 1];
			if(typeof onComplete != "function") {
				onComplete = null;
			}
		} else {
			var onComplete = null;
		}

		if(onComplete != null) {
			var sequence = args.slice(1, args.length - 1);
		} else {
			var sequence = args.slice(1, args.length);
		}

		if(!self.Connected()) {
			self.__logMessage("-> Ignoring Request() call (Not connected)");
			throw Organics.ErrNotConnected;
		}

		//var encodedData = JSON.stringify([requestName].concat(sequence));

		self.__requestCounter++;
		if(self.__requestCounter == -1) {
			self.__requestCounter++; // Special case, since -1 is special
		}

		var id = self.__requestCounter;
		if(onComplete) {
			self.__requestHandlers[self.__requestCounter] = onComplete;
		} else {
			id = -1; // Never respond to this request, please.
		}
		var encoded = JSON.stringify([id, requestName, sequence])

		if(Organics.WebSocketSupported) {
			self.__webSocket.send(encoded);

		} else {
			Organics.__ajax(self.__HTTP_URL, "POST", {
				complete: function(xhr) {
				},

				// If we are unable to POST data to the server; then this means either the server
				// had an internal error, OR we where disconnected somehow.
				//
				// The best thing to do in this situation is to say we where disconnected, because
				// in either of those two cases, our session state could be invalidated.
				error: function(xhr, msg) {
					if(xhr && xhr.status == 413) {
						// Data too large
						self.__handleDisconnect("JSON request data exceeded server's MaxBufferSize property.", 0);
						return;
					} else {
						self.__handleDisconnect("Request failed: " + msg);
					}
				}
			}, encoded, self.Timeout, {
				"X-Organics-Req": Organics.__rtMessage,
				"X-Organics-Conn": self.__connectionId
			});
		}
	}

	this.Connection.prototype.Handle = function(requestName, handler) {
		var self = this;

		if(handler === null) {
			// Remove the handler for this requestName key
			delete self.__handlers[requestName];

		} else {
			// Add the handler for this requestName key
			if(typeof handler !== "function") {
				throw TypeError("Handle() parameter \"handler\" must be function!");
			}

			self.__handlers[requestName] = handler;
		}
	}
}

