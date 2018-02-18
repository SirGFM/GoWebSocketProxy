# Go WebSocket Proxy

Go WebSocket Proxy is a small, simple and still quite bare-bones proxy that
enables sending/receiving requests between two client browsers.


## Package Features

* Stand-alone WebSocket proxy that allows two connections
* Package/interface for implementing WebSocket servers

Go to the wiki for more info on how to use the `websocket` package.


## Reasoning

For one of my side projects ([stream overlay](https://github.com/SirGFM/Roa-Stream-Skin)),
I needed some way to make two locally-running web applications talk with each
other. From what I've read, I would need a full-fledged web server...

So, I decided to write my own proxy server.


## TODOs

* Add tests and examples
* Make the application conformant with [RFC 6455](https://tools.ietf.org/html/rfc6455)
* Allow communication between arbitrary numbers of clients
* Allow authentication over HTTPS
