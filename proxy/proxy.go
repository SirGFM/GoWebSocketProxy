
package proxy

import (
    "github.com/pkg/errors"
    "github.com/SirGFM/GoWebSocketProxy/websocket"
    "io"
    "net"
    "time"
)

type proxy struct {
    // The connection with the end-point.
    conn net.Conn
    // Signals the status of the last attempt to receive from conn.
    connSelect chan error
    // Channel used to redirect messages to another proxy.
    send chan []byte
    // Channel used to receive redirected messages by another proxy.
    recv chan []byte
    // Signal from the main thread that this proxy should exit.
    stop *bool
    // Buffer used to receive messages from the connection.
    buf  []byte
    // Timeout for receiving messages
    timeout time.Duration
}

// Signals that the asynchronous conn.Read timed out
var receiveTimedOut = errors.New(
    "Timed out waiting for the first few bytes of a message")
// Signals that the connection to the end-point was closed.
var connectionClosed = errors.New("Connection closed")

// Setup a new proxy that communicates with conn. It redirects messages received
// from conn into send and sends messages received from recv into conn. When
// stop is set to true (and after timeout), the proxy stops running and closes
// its channel.
func Setup(conn net.Conn, stop *bool, send chan []byte, recv chan []byte,
    timeout time.Duration) *proxy {

    return &proxy {
        conn:           conn,
        send:           send,
        recv:           recv,
        stop:           stop,
        timeout:        timeout,
    }
}

// goWaitForMessage is the goroutine called on waitForMessage. See that
// function's documentation for details.
func (p *proxy) goWaitForMessage() {
    p.conn.SetReadDeadline(time.Now().Add(p.timeout))

    _, err := p.conn.Read(p.buf[:websocket.MinHeaderLength])
    if netErr, ok := err.(*net.OpError); ok && netErr != nil &&
        netErr.Err == net.ErrWriteToConnected {

        p.connSelect <- receiveTimedOut
    } else if err == io.EOF {
        p.connSelect <- connectionClosed
    } else {
        p.connSelect <- err
    }
}

// waitForMessage, sent from conn, and report the result on p.connSelect. The
// returned value may be one of:
//   * nil, if a message was received (and may have more bytes pending)
//   * connectionClosed, if the end-point was closed
//   * receiveTimedOut, if no message was received
//   * ???, if another error happened
// The received part of the message (i.e., its first websocket.MinHeaderLength
// bytes) shall be placed on p.buf.
func (p *proxy) waitForMessage() {
    // There's no need to synchronize accesss to p.connSelect because it always
    // happen from the same goroutine/thread.
    if p.connSelect == nil {
        p.connSelect = make(chan error, 1)
        go p.goWaitForMessage()
    }
}

// processMessage finishes receiving a pending message and put it into p.buf.
// The message is already redirected through conn.send (if required).
func (p *proxy) processMessage() (err error) {
    var msgLen, offset int

    p.buf, msgLen, offset, err = websocket.ReceiveFrame(p.conn, p.buf)
    if netErr, ok := err.(*net.OpError); ok && netErr != nil &&
        netErr.Err == net.ErrWriteToConnected {

        return receiveTimedOut
    } else if err == io.EOF {
        return connectionClosed
    }

    // TODO Check whether the Frame was actually directed at this proxy (e.g., Ping/Pong)
    p.send <- p.buf[:offset+msgLen]

    return nil
}

func (p *proxy) Run() {
    defer p.conn.Close()

    p.buf = make([]byte, websocket.MinHeaderLength)

    for !*p.stop {
        p.waitForMessage()

        // Instead of the usual time.After for avoiding running indefinitely,
        // p.connSelect gets signaled every p.timeout. Therefore, that is used
        // to avoid having the goroutine stuck waiting for messages.
        select {
        case <-time.After(p.timeout+time.Millisecond*10):
            // Nothing received within the expected time slice
        case msg := <- p.recv:
            // Received a message from another proxy. Send it to our end-point.
            p.conn.Write(msg)
        case err := <-p.connSelect:
            // If no error was detected, process the message. This allows
            // checking the error only once (regardless if from the channel or
            // the connection).
            if err == nil {
                err = p.processMessage()
            }

            if err == connectionClosed {
                // End-point was closed, nothing left to do here.
                return
            } else if err == receiveTimedOut {
                // Receive timed out, ignore the error and continue.
                err = nil
            } else if err == nil {
                err = p.processMessage()
            }

            p.connSelect = nil
        }

        // TODO Heartbeat (i.e., PONG)
    }
}
