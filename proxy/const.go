
package proxy

import (
    "github.com/pkg/errors"
    "time"
)

// How long between 'Pong' are sent
const heartBeatFrequency = time.Second * 10

// Signals that the asynchronous conn.Read timed out
var receiveTimedOut = errors.New(
    "Timed out waiting for the first few bytes of a message")
// Signals that the connection to the end-point was closed.
var connectionClosed = errors.New("Connection closed")
