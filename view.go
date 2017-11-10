
package main

import (
    "fmt"
    "net"
    "time"
)

func view(conn net.Conn, stop *bool, proxyConn chan []byte) {
    for !*stop {
        select {
        case <-time.After(timeout):
            // Channel timed-out, continuing...
            continue
        case msg := <-proxyConn:
            // Received a message.
            fmt.Printf("view: Redirecting '%#x'...\n", msg)
            // Simply redirect it to the WebSocket
            conn.Write(msg)
        }
    }
}
