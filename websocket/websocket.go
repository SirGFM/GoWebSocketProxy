package websocket

import (
    "crypto/tls"
    "fmt"
    "github.com/pkg/errors"
    "net"
    "sync"
)

// Ugly (and probably non-convetional) interface that exports the only public
// functions of connections to clients.
// None of these are thread-safe. However, since they should only be called from
// the goroutine handling the client's connection, this shouldn't be an issue...
type ClientConnection interface {
    // Send buf through the web-socket, wrapping it into a WebSocket frame and
    // as text data. Use 'Send' if you need to specify the type.
    Write(buf []byte) (int, error)
    // Send buf through the web-socket, wrapping it into a WebSocket frame and
    // as the specified type.
    Send(buf []byte, dataType Opcode) (error)
    // Send data throught the WebSocket as-is. Shouldn't be used unless you know
    // what you are doing!
    SendRaw(buf []byte) error
    // Requests to close the connection with the client.
    Close(code CloseReason, reason []byte) error
    // Release cached memory. Should be used for long-living connections that
    // occasionally sends really big messages.
    Flush()
}

type Server interface {
    // Returns a new instance of this server, passed to the runner.
    Clone(conn ClientConnection) (Server, error)
    // Do something with the received message. Offset points to the actual
    // payload within the WebSocket frame.
    Do(msg []byte, offset int) error
    // Called after the connection to this client was closed.
    Cleanup()
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

    // Configuration used when accepting connections over TLS.
    tlsConfig *tls.Config
    // Synchronizes access to tlsMutext.
    tlsMutex sync.Mutex
    // Server that does something with the received messages.
    serverTempl Server
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
        ctx.serverTempl = nil
    }
}

// Set everything up so the conectext may accept connections.
func (ctx *Context) Setup(serverTempl Server) error {
    var err error

    if ctx.Port <= 0 {
        return errors.New("websocket: Invalid port")
    }

    ctx.ln, err = net.Listen("tcp", fmt.Sprintf("%s:%d", ctx.Ip, ctx.Port))
    if err != nil {
        return errors.Wrap(err, "websocket: Failed to listen to address")
    }

    ctx.serverTempl = serverTempl

    return nil
}

// Set the new TLS configuration. Open connections will still use the old
// credentials, but new connections will automatically use this new one.
func (ctx *Context) SetTlsConfig(config *tls.Config) {
    ctx.tlsMutex.Lock()
    if config != nil {
        // Stores a copy of the TLS configuration (to avoid unsynchronized
        // access)
        cpy := *config
        ctx.tlsConfig = &cpy
    } else {
        ctx.tlsConfig = nil
    }
    ctx.tlsMutex.Unlock()
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
    if ctx.ln == nil || ctx.serverTempl == nil {
        sendError(cerr, errors.New("websocket: Setup hasn't been configured yet"))
        return
    }

    ctx.running = true
    for ctx.running {
        conn, err := ctx.ln.Accept()
        if err != nil {
            sendError(cerr, errors.Wrap(err, "websocket: Failed to accept connection"))
            continue
        }

        // Wrap the connection before continuing
        ctx.tlsMutex.Lock()
        if ctx.tlsConfig != nil {
            go ctx.serveTls(tls.Server(conn, ctx.tlsConfig), cerr)
        } else {
            go ctx.serve(conn, cerr)
        }
        ctx.tlsMutex.Unlock()
    }

    sendError(cerr, nil)
}

// Serves a websocket connection over TLS.
func (ctx *Context) serveTls(conn *tls.Conn, cerr chan error) {
    gerr := conn.Handshake()
    if gerr != nil {
        err := errors.Wrap(gerr, "websocket: TLS handshake failed")
        sendError(cerr, err)
        return
    }

    ctx.serve(conn, cerr)
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

    // Clone the server (if needed)
    r := getNewRunner()
    server, err := ctx.serverTempl.Clone(r)
    if err != nil {
        sendError(cerr, err)
        return
    }

    // Runs until any error happens
    err = r.setupRunner(conn, &ctx.running, server, defaultTimeout)
    if err != nil {
        sendError(cerr, err)
        return
    }
    err = r.Run()
    if err != nil {
        sendError(cerr, err)
    }

    server.Cleanup()
}
