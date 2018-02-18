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


## Testing secure connections

I just had some [fun](http://dwarffortresswiki.org/images/4/40/FunComic.png)
time trying to setup a self-signed certificate that would be accepted by the
browser... Turns out you can't do that easily (or, at least, I failed horribly).

To save time from anyone wanting to test that, here are a few pointers:

* Use [minica](https://github.com/jsha/minica) to create your server's and the
  client's certificates.
    * But do not forget to add the **server CA certificate to the browser's
      trusted certificates.**
* Use the following OpenSSL command to create a PKCS#12 with the client's certificate:
    * `openssl pkcs12 -export -inkey <client_key>  -in <client_cert> -name <alias> -out <client_p12>`

And I must give some credits to
[Let's Encrypt](https://letsencrypt.org/docs/certificates-for-localhost#making-and-trusting-your-own-certificates),
for pointing me toward `minica` (mostly because I didn't have to re-implement
that).


## TODOs

* Add tests and examples
* Make the application conformant with [RFC 6455](https://tools.ietf.org/html/rfc6455)
