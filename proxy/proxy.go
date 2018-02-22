package proxy

import (
    "github.com/pkg/errors"
    "github.com/SirGFM/GoWebSocketProxy/websocket"
    "sync"
)

// Synchronizes access to chanA and chanB
var _chanMutex sync.Mutex
// One of the two channels usable by this server
var _chanA *closeableChannel
// One of the two channels usable by this server
var _chanB *closeableChannel

// Dumb channel-wrapper that allows avoiding writing to a closed channel
type closeableChannel struct {
    Mutex sync.Mutex
    IsClosed bool
    Channel chan []byte
}

// The proxy server
type server struct{
    // Connection to the end-point.
    conn websocket.ClientConnection
    // Channel connected to another server (from which that server sends
    // messages)
    recv *closeableChannel
    // Signals that this server should get closed
    stop chan struct{}
}

// Retrieves the template for the proxy server
func GetTemplate() websocket.Server {
    return &server{}
}


// Redirects messages received by another proxy server
func (p *server) redirect() {
    for {
        select {
        case <-p.stop:
            // Stops this server
            return
        case data := <-p.recv.Channel:
            // Received data from the other server: redirect to this' end-point
            p.conn.SendRaw(data)
        }
    }
}

// Creates a new server with a global reference (so other servers may redirect
// message to it, and vice-versa).
func (*server) Clone(conn websocket.ClientConnection) (websocket.Server, error) {
    _chanMutex.Lock()
    defer _chanMutex.Unlock()

    // Check if there's at least one free channel
    if _chanA != nil && _chanB != nil {
        return nil, errors.New("proxy: No available channel")
    }

    p := &server{
        conn: conn,
    }
    p.stop = make(chan struct{})
    p.recv = &closeableChannel{}
    p.recv.Channel = make(chan []byte)

    // Store the channel globally
    if _chanA == nil {
        _chanA = p.recv
    } else {
        _chanB = p.recv
    }

    // Start the redirecting goroutine
    go p.redirect()

    return p, nil
}

// Redirect the received message to any other listening server.
func (p *server) Do(msg []byte, offset int) error {
    var otherChan *closeableChannel

    // Retrieve the global channel owned by another server
    _chanMutex.Lock()
    if _chanA == p.recv {
        otherChan = _chanB
    } else {
        otherChan = _chanA
    }
    _chanMutex.Unlock()

    if otherChan == nil {
        return nil
    }

    // If it hasn't been closed yet (or since acquiring its address), send it
    // our data.
    otherChan.Mutex.Lock()
    if !otherChan.IsClosed {
        // Ensure that a safe buffer is used... This gave me a great deal of
        // headache, since I was passing the cached buffer through a channel
        // (in a proxy).
        b := make([]byte, len(msg))
        copy(b, msg)
        otherChan.Channel <- b
    }
    otherChan.Mutex.Unlock()

    return nil
}

// Makes this server stop running and clears its global reference.
func (p *server) Cleanup() {
    // Signal the server to stop
    p.stop <- struct{}{}

    // Releases the server's channel
    p.recv.Mutex.Lock()
    p.recv.IsClosed = true
    close(p.recv.Channel)
    p.recv.Mutex.Unlock()

    // Remove it from the global list
    _chanMutex.Lock()
    if _chanA == p.recv {
        _chanA = nil
    } else {
        _chanB = nil
    }
    _chanMutex.Unlock()
}
