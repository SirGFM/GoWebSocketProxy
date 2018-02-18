package websocket

import (
    "github.com/pkg/errors"
    "io"
    "net"
    "time"
)

const defaultTimeout = time.Second * 10
// How long between 'Pong' are sent
var heartBeatFrequency = time.Second * 10

type runner struct {
    // The server that do stuff when it receives a message.
    server Server
    // The connection with the end-point.
    conn net.Conn
    // Signals the status of the last attempt to receive from conn.
    connSelect chan error
    // Signal from the main thread that this runner should exit.
    running *bool
    // Buffer used to receive messages from the connection.
    buf []byte
    // Time to send the next heart beat.
    heartBeatTime time.Time
    // Timeout for receiving messages.
    timeout time.Duration
    // If a connection closed was send but there was no response from the other
    // end-point yet.
    closed bool
}

// Setup a new runner that communicates with conn. When running is set to false
// (and after timeouts), the runner stops, as the caller should close the
// channel.
func setupRunner(conn net.Conn, running *bool, server Server,
    timeout time.Duration) (r *runner, err error) {

    r = &runner {
        server:  server,
        conn:    conn,
        running: running,
        timeout: timeout,
    }
    r.buf = make([]byte, MinHeaderLength)

    return
}

// checkConnectionError converts the various possible connection errors to the
// pre-defined ones.
func checkConnectionError(err error) error {
    if netErr, ok := err.(*net.OpError); ok && netErr != nil &&
        netErr.Err == net.ErrWriteToConnected {

        return receiveTimedOut
    } else if err == io.EOF {
        return connectionClosed
    } else {
        return errors.Wrap(err, "websocket: Unexpected error")
    }
}

// goWaitForMessage is the goroutine called on waitForMessage. See that
// function's documentation for details.
func (r *runner) goWaitForMessage() {
    r.conn.SetReadDeadline(time.Now().Add(r.timeout))

    _, err := r.conn.Read(r.buf[:MinHeaderLength])
    r.connSelect <- checkConnectionError(err)
}

// waitForMessage, sent from conn, and report the result on r.connSelect. The
// returned value may be one of:
//   * nil, if a message was received (and may have more bytes pending)
//   * connectionClosed, if the end-point was closed
//   * receiveTimedOut, if no message was received
//   * ???, if another error happened
// The received part of the message (i.e., its first MinHeaderLength
// bytes) shall be placed on r.buf.
func (r *runner) waitForMessage() {
    // There's no need to synchronize accesss to r.connSelect because it always
    // happen from the same goroutine/thread.
    if r.connSelect == nil {
        r.connSelect = make(chan error, 1)
        go r.goWaitForMessage()
    }
}

// processMessage finishes receiving a pending message and put it into r.buf.
// The message is already redirected through conn.send (if required).
func (r *runner) processMessage() (err error) {
    var msgLen, offset int

    r.buf, msgLen, offset, err = receiveFrame(r.conn, r.buf)
    err = checkConnectionError(err)
    if err != nil {
        return err
    }

    switch Opcode(r.buf[OpcodeIndex] & OpcodeMask) {
    case ConnectionClose:
        // 5.5.1 of RFC 6455 defines that when a 'ConnectionClose' is
        // received, the end-point must reply as soon as possible with its
        // own 'ConnectionClose'.
        if !r.closed {
            // The end-point started the closing procedure. In order to reply
            // and close the connection more easily, send the response and
            // return connectionClosed.
            r.conn.Write(r.buf[:offset+msgLen])
        }
        // The end-point replied to our 'ConnectionClose'. Simply exit.
        return connectionClosed
    case Ping:
        // 5.5.2 and 5.5.3 of RFC 6455 defines that a Ping request must be
        // answered with a Pong response. If it contained any
        // 'Payload Data', the same data must be sent on the response.
        //
        // Therefore, simply change the operation to 'Pong' and move on.
        r.buf[OpcodeIndex] &^= OpcodeMask
        r.buf[OpcodeIndex] |= byte(Pong)

        _, err = r.conn.Write(r.buf)
        err = checkConnectionError(err)
        return
    case Pong:
        // 5.5.3 of RFC 6455 defines that unsolicited 'Pong' requests acts as
        // heart-beat for the connection and no response is expected.

        // TODO Check if a 'Pong' was expected and clear that flag.
    }

    // If the connection was closed by this end-point, it must send a
    // 'ConnectionClose'. Ignore any other messages.
    if r.closed {
        return
    }

    // Ensure that a safe buffer is used... This gave me a great deal of
    // headache, since I was passing the cached buffer through a channel
    // (in a proxy).
    b := make([]byte, offset+msgLen)
    copy(b, r.buf[:offset+msgLen])

    err = r.server.Do(b, offset)

    return
}

// Run handles the connection the end-point.
func (r *runner) Run() (err error) {
    for *r.running {
        r.waitForMessage()

        // Instead of the usual time.After for avoiding running indefinitely,
        // r.connSelect gets signaled every r.timeout. Therefore, that is used
        // to avoid having the goroutine stuck waiting for messages.
        select {
        case <-time.After(r.timeout+time.Millisecond*10):
            // Nothing received within the expected time slice
        case err = <-r.connSelect:
            // If no error was detected, process the message. This allows
            // checking the error only once (regardless if from the channel or
            // the connection).
            if err == nil {
                err = r.processMessage()
            }

            if err == connectionClosed {
                // End-point was closed, nothing left to do here.
                err = errors.New(connectionClosed.Error())
                return
            } else if err == receiveTimedOut {
                // Receive timed out, ignore the error and continue.
                err = nil
            }

            r.connSelect = nil
        }

        // From time to time, send a 'Pong' message to check that the connection
        // is alive.
        if now := time.Now(); now.After(r.heartBeatTime) {
            r.conn.SetWriteDeadline(time.Now().Add(r.timeout))
            _, err = r.conn.Write(HeartBeatMessage)
            err = checkConnectionError(err)
            if err != nil {
                return
            }

            r.heartBeatTime = now.Add(heartBeatFrequency)
        }
    }

    err = nil
    return
}
