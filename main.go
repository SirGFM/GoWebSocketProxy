package main

import (
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

    ctx := websocket.NewContext("", "/proxy", 60000, 2)
    defer ctx.Close()

    signalTrap = make(chan os.Signal, 1)
    go quitCleanup(signalTrap, ctx)
    signal.Notify(signalTrap, os.Interrupt)

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
