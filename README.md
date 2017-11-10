# Go WebSocket Proxy

Go WebSocket Proxy is a small, simple and still quite bare-bones proxy that
enables sending/receiving requests between two client browsers.

Its advantages over pure WebSockets is the lack of necessity of a full-fledged
server (as Apache, or even using an Electron App). It can be used to allow
communication between to locally-running web applications, even if they are
running on different devices.

## TODOs

* Make both routines symmetrical and more robust:
    * Ping the client from time-to-time
    * Fix receiving of non-text messages
    * Allow communication in any direction
* Add tests and examples
* Make the application conformant with [RFC 6455](https://tools.ietf.org/html/rfc6455)
* Allow communication between arbitrary numbers of clients
* Allow authentication over HTTPS
