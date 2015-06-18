  * **Table of contents**
    * [Why it is important](Security#Why_it_is_important.md)
    * [Developing safe applications](Security#Developing_safe_applications.md)
      * [Cross-site scripting (XSS)](Security#Cross-site_scripting.md)
      * [Cross-site request forgery (CSRF)](Security#Cross-site_request_forgery.md)
    * [What Organics does](Security#What_Organics_does.md)
      * [Session management](Security#Session_management.md)
        * [Cookies](Security#Cookies.md)
      * [Connection management](Security#Connection_management.md)
        * [Pings](Security#Pings.md)
      * [Buffer limits](Security#Buffer_limits.md)
      * [HTML escaping](Security#HTML_escaping.md)

# Why it is important #

  1. **_An developer who compromises his own website is worse than an hacker compromising an website for fun or financial gain._**
  1. **_You are the only one who can protect your own users._**

Security in web applications has been over the past few years gaining an incredibly significant amount of acknowledgement, websites and applications run by big companies are being compromised every day by hackers and developers alike.

_It is crucial as an developer that you consider what types of data your application will utilize, how secure it needs to be, and what types of precautions you should make._

This document will give you an brief overview of some very common security holes in web applications today, and will inform you of what you can do to protect yourself and your users against them.

# Developing safe applications #
In developing safe web applications, two of the most important things to protect against are:
  1. Cross-site scripting (XSS)
  1. cross-site request forgery (CSRF)

In this section we will explain what these security holes mean, typical methods of protecting against them, and how Organics helps you in developing safe web applications.

## Cross-site scripting ##
### What is it? ###
Having an cross-site scripting (XSS) vulnerability on your website or web application means you're allowing other, mostly malicious users, to run malicious code on other users browsers, when they visit your website.

The code can be anything, from JavaScript, to HTML, CSS, anything you can use in the web.

[Wikipedia's very nice article on cross-site scripting](http://en.wikipedia.org/wiki/Cross-site_scripting) states that in 2007 Symantec determined that cross-site scripting accounted for roughly 84% of all web security vulnerabilities.

### How can I protect against it? ###
The answer is simple, yet needs to be considered when designing nearly any aspect of an web site or web application.

**_Never run untrusted, unvalidated, or otherwise user-inputted code of any kind on your web page._**

What this means is:
  * Never display an user's name, unless it has been properly escaped.
    * User types in an name of `George</p><malicious-code></malicious-code>` in the registration box.
    * Some other, victim user, visits your web site, your web site displays his name, inside of an paragraph tag, like so:
    * `...<body><p>George</p><malicious-code></malicious-code></p></body>`
    * His user name was not properly escaped, thus breaking out of the paragraph tag, and the malicious code will run on the victim's browser.
  * Never display HTML or CSS code, or run JavaScript code if it is hosted on an server that you do not control or trust.
    * Any person with access to that server, may change the content and insert malicious code at any point in time.

## Cross-site request forgery ##
### What is it? ###
An cross-site request forgery (CSRF) vulnerability means you're allowing an, potentially malicious, user to perform actions on behalf of another user (the victim).

[Wikipedia](http://en.wikipedia.org/wiki/Cross-site_request_forgery) has an great article on what cross-site request forgery is, very much so worth an read.

CSRF is an less well known security vulnerability, partly due to the fact that it is commonly misunderstood.

WebSockets use an Origin based security policy, that is, whenever an WebSocket connection is created, the browser informs the server what web page the connection was created from (similar to CORS), thus WebSockets do not suffer the same security problems that GET and POST HTTP requests do.

Organics uses only the HTTP POST method, so from here on we will only cover security about POST, if you wish to learn more about other HTTP methods, we recommend you look at the aforementioned wikipedia article, or do research online.

Here are some important bullet points about HTTP POST security
  * XMLHttpRequest cannot make cross-domain POST requests with custom headers, without an CORS preflight request being sent first.
    * Some plugin vulnerabilities have circumvented this in the past.
  * HTML forms can be submitted cross-domain, and even hidden from the victim.

### How can I protect against it? ###
Consider the following points.
  * Visiting an URL causes an action that only privileged users may perform. E.g. Visit www.bank.com/withdraw/5000 and, if you are logged in, then we withdraw $5000 from your account. Consider I send you (an logged in victim of this attack) that URL, you click it, and lose $5000.
    * Never allow an URL to perform an action for an user automatically. (E.g. www.social.net/deleteMyAccountForever)

### How Organics helps ###
An custom randomly generated session-based CSRF token is attached as an custom header to every XMLHttpRequest, which cannot be added to form submissions, and unless it is valid, the request fails. (Also known as double submit.)


# What Organics does #
Organics tries it's best to give you, the developer, an consistent and security oriented API.

This section covers what Organics does to help alleviate some security concerns you may have.

## Session management ##
Part of the concern over CSRF is session hijacking, where the unique session identifier, stored as an cookie in your browser, can be guessed or otherwise determine by an malicious user, whom in turn hijacks your session, becoming you.
### Cookies ###
Organics generates cryptographically random session cookies, so that they cannot be guessed, determined, or brute-forced easily.

First, Organics will fill an slice of Server.SessionKeySize() number of random bytes using Go's crypto/rand package.

And secondly, Organics will hash that array using Go's SHA256 hashing algorithm.

## Connection management ##
One concern would be keeping unused, likely unresponsive or closed, connections open and wasting memory or CPU time on them.

### Pings ###
Organics uses pings (AKA heartbeats) both over Long Polling and WebSocket methods, in order to ensure that the other end is still there.

It should be noted that these pings are empty messages, and only occur should no other interaction have occurred recently, thus pings are very light-weight with the great benefit of dropping unused HTTP or WebSocket connections, which might still be open due to outdated web browsers, or proxies that keep connections open forever.

## Buffer limits ##
An common security hole is allowing an user to allocate as much memory as they want, on your web server.

Organics limits the amount of data that an client may send to the server in an single Message before being disconnected.

This stops the primary security concern, and you can easily configure the maximum amount of memory that each message may allocate, to match your application's needs.

## HTML escaping ##
Before any strings are sent to the browser, they should be HTML escaped if you intend on displaying them as HTML output. You should use Go's html package for this. Otherwise an malicious user might be able to insert malicious code into your website (XSS).