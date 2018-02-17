package newWebsocket

import (
    "bufio"
    "crypto/sha1"
    "encoding/base64"
    "fmt"
    "github.com/pkg/errors"
    "net"
    "net/http"
    "strings"
)

// Currently, only ignores if no |Host| was supplied in the header.
const Strict = false
// GUID used by every WebSocket server (as specified on the RFC).
const WS_SERVER_ID = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"

// Calculates the server response key, as defined by the RFC (see 1.3)
func calculateServerResponseKey(key string) string {
    s := sha1.Sum([]byte(key + WS_SERVER_ID))
    b64 := base64.StdEncoding.EncodeToString(s[:])

    return b64
}

// Handshake handles checking and upgrading a HTTP request into a WebSocket
// connection (as defined by RFC 6455 - The WebSocket Protocol).
func handshake(conn net.Conn, validUri string) (res http.Response, retErr error) {
    var b64ConnKey string

    // Properly set the status code text on exit
    defer func() {
        res.Status = http.StatusText(res.StatusCode)
    }()

    // Set initial response, just in case...
    res.StatusCode = http.StatusForbidden
    res.Proto = "HTTP/1.0"
    res.ProtoMajor = 1
    res.ProtoMinor = 0

    req, err := http.ReadRequest(bufio.NewReader(conn))
    if err != nil {
        retErr = errors.Wrap(err, "Failed to receive client handshake")
        return
    }

    res.Proto = req.Proto
    res.ProtoMajor = req.ProtoMajor
    res.ProtoMinor = req.ProtoMinor

    // 4.2.1 Reading the Client's Opening Handshake

    // 1. Check that it's a HTTP/1.1 or higher GET
    // (ignore "Request-URI" but check its presence)
    if !strings.HasPrefix(req.Proto, "HTTP/") {
        retErr = errors.New(fmt.Sprintf("Non HTTP request: '%s'", req.Proto))
        return
    } else if req.ProtoMajor == 1 && req.ProtoMinor < 1 ||
        req.ProtoMajor < 1 {

        retErr = errors.New("Non HTTP/1.1 or higher")
        return
    } else if req.Method != "GET" {
        retErr = errors.New("Non GET request")
        return
    } else if req.RequestURI == "" {
        retErr = errors.New("Missing RequestURI in request")
        return
    } else if req.RequestURI != validUri {
        retErr = errors.New("Invalid RequestURI")
        return
    }

    // 2. Check for a |Host| header with server's authority
    if host := req.Header.Get("Host"); len(host) == 0 {
        retErr = errors.New("Missing |Host| header field")
        if Strict {
            return
        } else {
            fmt.Printf("  Ignoring: %s\n", retErr.Error())
            retErr = nil
        }
    }

    // 3. Check for an |Upgrade| header == "websocket"
    if want, got := "websocket", req.Header.Get("Upgrade"); want != got {
        retErr = errors.New(fmt.Sprintf(
            "Invalid |Upgrade| header field: wanted '%s', got '%s'\n",
            want, got))
        return
    }

    // 4. Check for a |Connection| header == "Upgrade"
    if want, got := "Upgrade", req.Header.Get("Connection"); !strings.Contains(got, want) {
        retErr = errors.New(fmt.Sprintf(
            "Invalid |Connection| header field: wanted '%s', got '%s'\n",
            want, got))
        return
    }

    // 5. Check for a |Sec-WebSocket-Key| with the base-64 key
    if b64ConnKey = req.Header.Get("Sec-WebSocket-Key"); len(b64ConnKey) == 0{
        retErr = errors.New("Missing |Sec-WebSocket-Key|  header field")
        return
    }

    // 6..10 are optional (and, therefore, ignored)

    // 4.2.2 Sending the Server's Opening Handshake

    res.Header = make (http.Header)

    // 1. Connection isn't HTTPS, so ignore it
    // 2. Stuff that the server can do... ignore (as it isn't a MUST)
    // 3. "The server MAY...": NOPE
    // 4. Maybe later

    // 5. Set fields required to accept the connection

    // 5.1 Set status as Switching Protocols
    res.StatusCode = http.StatusSwitchingProtocols

    // 5.2 Set |Upgrade| = "websocket"
    res.Header.Set("Upgrade", "websocket")

    // 5.3 Set |Connection| = "Upgrade"
    res.Header.Set("Connection", "Upgrade")

    // 5.4 Set |Sec-WebSocket-Accept| with the base_64(SHA_1(key+UIGD))
    serverKey := calculateServerResponseKey(b64ConnKey)
    res.Header.Set("Sec-WebSocket-Accept", serverKey)

    // 5.5 Later
    // 5.6 Later

    return
}

// Updates a HTTP Response to have an response code of Service Unavailable.
func updateResponseUnavailable(res *http.Response) {
    res.StatusCode = http.StatusServiceUnavailable
    res.Status = http.StatusText(res.StatusCode)
    // Remove any previously set header
    res.Header = nil
}
