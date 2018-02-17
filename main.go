package main

import (
    "fmt"
    "github.com/SirGFM/GoWebSocketProxy/newWebsocket"
    "os"
    "os/signal"
)

type myServer struct{}

// Set anythin up required by the server's connection to a client.
func (*myServer) Setup() error { return nil }

// Do something with the received message.
func (*myServer) Do(msg []byte, offset int) error {
    msg = msg[offset:]
    fmt.Printf("Got message %#x (\"%s\")\n", msg, string(msg))
    return nil
}

func quitCleanup(c chan os.Signal, ctx *newWebsocket.Context) {
    _ = <-c

    ctx.Close()
}

func main() {
    var signalTrap chan os.Signal

    ctx := newWebsocket.NewContext("", "/proxy", 60000, 2)
    defer ctx.Close()

    signalTrap = make(chan os.Signal, 1)
    go quitCleanup(signalTrap, ctx)
    signal.Notify(signalTrap, os.Interrupt)

    err := ctx.Setup(&myServer{})
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
