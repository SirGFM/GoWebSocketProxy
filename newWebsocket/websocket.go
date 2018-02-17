package newWebsocket

import (
    "fmt"
    "github.com/pkg/errors"
    "net"
    "sync"
)

type Server interface {
    // Set anythin up required by the server's connection to a client.
    Setup() error
    // Do something with the received message.
    Do(msg []byte) error
}

type Context struct {
    // Local IP where connections are accept (blank for accepting globally).
    Ip string
    // URI for accepting connections (e.g., 'ws://Ip:Port/Uri').
    Uri string
    // Port where the WebSocket will listen to. Cannot be changed after starting
    // the server.
    Port int
    // How many simultaneous connections this server accepts. Anything less or
    // equal to 0 means unlimited.
    MaximumConnections int

    // Server that does something with the received messages.
    server Server
    // Connection listener.
    ln net.Listener
    // Synchronizes access to connectionCounter.
    counterMutex sync.Mutex
    // Number of active connections.
    connectionCounter int
    // Whether the server is running.
    running bool
}

// Creates a new context. The context may be manually instantiated as well (as
// long as all public fields are set).
func NewContext(ip, uri string, port, maxConn int) *Context {
    return &Context{
        Ip:                 ip,
        Uri:                uri,
        Port:               port,
        MaximumConnections: maxConn,
    }
}

// Closes and stops the listening server.
func (ctx *Context) Close() {
    ctx.running = false
    if ctx.ln != nil {
        ctx.ln.Close()
        ctx.ln = nil
        ctx.server = nil
    }
}

// Set everything up so the conectext may accept connections.
func (ctx *Context) Setup(server Server) error {
    if ctx.Port <= 0 {
        return errors.New("websocket: Invalid port")
    }

    ctx.ln, err = net.Listen("tcp", fmt.Sprintf("%s:%d", ctx.Ip, ctx.Port))
    if err != nil {
        return errors.Wrap(err, "websocket: Failed to listen to address")
    }

    ctx.server = server

    return nil
}

// Sends an error (if the channel is non-nil).
func sendError(cerr chan error, err error) {
    if cerr != nil {
        cerr <- err
    }
}

// Runs the websocket server and blocks execution until 'Stop' is called. Can
// (and should) safelly be called from/as a goroutine. If it fails to accept any
// connection, the error is reported through the supplied channel (if any).
//
// The server signals that it has exited by sending a nil error to the channel.
func (ctx *Context) Run(cerr chan error) {
    if ctx.ln == nil || ctx.server == nil {
        sendError(cerr, errors.New("websocket: Setup hasn't been configured yet"))
        return
    }

    for ctx.running {
        conn, err := ctx.ln.Accept()
        if err != nil {
            sendError(cerr, errors.Wrap(err, "websocket: Failed to accept connection"))
            continue
        }

        go ctx.serve(conn, cerr)
    }

    sendError(cerr, nil)
}

// Serves a websocket connection.
func (ctx *Context) serve(conn net.Conn, cerr chan error) {
    defer conn.Close()

    // Try to upgrade the connection to a WebSocket connection
    res, err := handshake(conn, ctx.Uri)
    if err == nil {
        // On success, check if the server may still accept connections. The
        // lock can't be defer'ed since it's quite short lived.
        ctx.counterMutex.Lock()
        if ctx.connectionCounter >= ctx.MaximumConnections &&
            ctx.MaximumConnections > 0 {

            // Set the error
            updateResponseUnavailable(&res)
            err = errors.New("websocket: Too many connections")
        } else {
            ctx.connectionCounter++
        }
        ctx.counterMutex.Unlock()
    }

    res.Write(conn)

    if err != nil {
        // On failure, simply return (since the response has already been sent).
        sendError(cerr, err)
        return
    }

    // Release the counter on exit
    defer func(ictx *Context) {
        ictx.counterMutex.Lock()
        ictx.connectionCounter--
        ictx.counterMutex.Unlock()
    } (ctx)

    // Runs until any error happens
    r, err := setupRunner(conn, &ctx.stop, ctx.server, defaultTimeout)
    if err != nil {
        sendError(cerr, err)
        return
    }
    err = r.Run()
    if err != nil {
        sendError(cerr, err)
    }
}
