
package main

import (
    //"bytes"
    "fmt"
    "net"
    "os"
    "os/signal"
)

func quitCleanup(c chan os.Signal, stop *bool, tcpListener *net.Listener) {
    _ = <-c

    *stop = true
    if *tcpListener != nil {
        (*tcpListener).Close()
    }
}

func main() {
    var ln net.Listener
    var signalTrap chan os.Signal
    var proxyConn chan []byte
    var err error
    var port int
    var stop bool

    port = 60000

    signalTrap = make(chan os.Signal, 1)
    go quitCleanup(signalTrap, &stop, &ln)

    signal.Notify(signalTrap, os.Interrupt)

    ln, err = net.Listen("tcp", fmt.Sprintf(":%d", port))
    if err != nil {
        panic(err)
    }
    defer func() {
        if ln != nil {
            ln.Close()
        }
        ln = nil
    }()

    proxyConn = make(chan []byte, 1)

    fmt.Println("Waiting...")
    for !stop {
        conn, err := ln.Accept()
        if err != nil {
            fmt.Printf("Failed to accept conn: %+v\n", err)
            continue
        }
        go wsServer(conn, &stop, proxyConn)
    }
}
