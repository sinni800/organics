#summary Organics Security Theory

  * **Table of contents**
    * [Understanding Go error handling](ErrorHandling#Understanding_Go_error_handling.md)
    * [Understanding JavaScript error handling](ErrorHandling#Understanding_JavaScript_error_handling.md)
    * [How Organics error handling works](ErrorHandling#How_Organics_error_handling_works.md)

# Understand Go error handling #
It's very important to understand how Go handles errors, as Organics tries it's very best to follow this model in it's most basic form.

In Go, `error` is an built-in interface described [here](http://golang.org/pkg/builtin/#error).

It's very basic, it must simply have an `Error()` method which returns an `string`.

If you want anything more, you must use your own type, then you could store other things (error code, etc).

# Understand JavaScript error handling #
Understanding how JavaScript's error handling differs from Go's is also very important too.

JavaScript uses exception based error handling, described [here](https://developer.mozilla.org/en-US/docs/JavaScript/Reference/Global_Objects/Error).

In Go, errors are simply returned, and in JavaScript errors are much more like Go's panic()/recover() facilities.

# How Organics error handling works #
The first question we had to ask ourselves when developing Organics, was where do we **want** errors?

There are two end-points to consider, both the client and the server.

Consider the example of an very large-scale social networking website.

If an request is made to the JavaScript client, and the JavaScript client for some reason runs into errors, should the server be told about it?

The answer is no, if we did something like this then you could be setting yourself up for an self-inflicted denial of service attack.

Now on the opposite side of that, if an request is made to the Go server, and the server is for some reason encounters errors, should the client be told about it?

The answer is yes, this could contain information that is valuable to the user in some way.

For this reason, you should always think of the following when designing an application using Organics:

  1. The server should give the client errors.
  1. The client shouldn't give the server errors.

Now on to the most daring question of all, how do you **actually** transport an error from the server, to the client? It's quite simple actually.

In the [Go Server API](GoServerApi.md) you may use the [Server.Handle](http://godoc.org/code.google.com/p/organics#Server.Handle) method, to handle an request. While handling this request, if you encounter some error that you would like to inform the user of, simply return it (as an string, using the `error` interface described before, which defines the `error.Error()` method, which returns an string).

On your JavaScript client, you'll simply have either an empty string, or an (non-empty) error string received in your (optional) onComplete parameter to your call to [Connection.Request](https://code.google.com/p/organics/wiki/JSClientApi#Connection.Request) .