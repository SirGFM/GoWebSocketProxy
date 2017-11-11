
package main

import (
    "fmt"
    "github.com/SirGFM/GoWebSocketProxy/proxy"
    "net"
    "sync"
    "time"
)

const viewUri = "/view"
const ctrlUri = "/control"
const timeout = time.Second * 10
var uriLock = sync.Mutex{}

var a = make(chan []byte, 1)
var b = make(chan []byte, 1)

var validUris map[string]bool = map[string]bool {
    viewUri: true,
    ctrlUri: true,
}

func wsServer(conn net.Conn, stop *bool, proxyConn chan []byte) {
    //var buf bytes.Buffer

    defer conn.Close()

    // TODO Set conn.SetDeadline()

    // Lock to guarantee single access to validUris.
    // Do not defer since lock has a pretty well defined scope
    uriLock.Lock()

    res, uri, err := handshake(conn, validUris)
    if err != nil {
        fmt.Printf("%+v\n", err)
    }
    // Send the response
    res.Write(conn)

    if err != nil {
        uriLock.Unlock()
        return
    }

    // Someone connected to a URI, release the lock
    delete(validUris, uri)
    uriLock.Unlock()

    defer func() {
        uriLock.Lock()
        validUris[uri] = true
        uriLock.Unlock()
    }()

    if uri == viewUri {
        p := proxy.Setup(conn, stop, a, b, time.Second * 10)
        p.Run()
    } else if uri == ctrlUri {
        p := proxy.Setup(conn, stop, b, a, time.Second * 10)
        p.Run()
    }
}
