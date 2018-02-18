package main

import (
    "flag"
    "fmt"
    "github.com/SirGFM/GoWebSocketProxy/websocket"
    "github.com/SirGFM/GoWebSocketProxy/proxy"
    "os"
    "os/signal"
)

func quitCleanup(c chan os.Signal, ctx *websocket.Context) {
    _ = <-c

    ctx.Close()
}

func main() {
    var signalTrap chan os.Signal
    var url string
    var uri string
    var port int

    // Parse command line arguments
    flag.StringVar(&url, "url", "", "URL accepted by the server. Empty defaults to any address/url.")
    flag.StringVar(&uri, "uri", "/proxy", "URI that references the WebSocket server.")
    flag.IntVar(&port, "port", 60000, "Port for the WebSocket server.")
    flag.Parse()

    // Create the WebSocket context
    ctx := websocket.NewContext(url, uri, port, 2)
    defer ctx.Close()

    // Detect keyboard interrupts (Ctrl+C) and exit gracefully.
    signalTrap = make(chan os.Signal, 1)
    go quitCleanup(signalTrap, ctx)
    signal.Notify(signalTrap, os.Interrupt)

    // Start the server
    err := ctx.Setup(&proxy.Server{})
    if err != nil {
        panic(err.Error())
    }

    cerr := make(chan error)
    go ctx.Run(cerr)

    for {
        err = <-cerr
        if err == nil {
            break
        }

        fmt.Printf("Got error from server: %+v\n", err)
    }
}
