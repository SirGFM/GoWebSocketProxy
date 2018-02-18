package main

import (
    "crypto/tls"
    "crypto/x509"
    "encoding/pem"
    "flag"
    "fmt"
    "github.com/pkg/errors"
    "github.com/SirGFM/GoWebSocketProxy/websocket"
    "github.com/SirGFM/GoWebSocketProxy/proxy"
    "io/ioutil"
    "os"
    "os/signal"
)

// Parses a PEM certificate.
func parseCertificate(path string) (cert *x509.Certificate, err error) {
    var pemData []byte

    pemData, err = ioutil.ReadFile(path)
    if err != nil {
        return
    }
    block, rest := pem.Decode(pemData)
    if block == nil || len(rest) != 0 {
        err = errors.New("Failed to decode the PEM certificate")
        return
    }
    cert, err = x509.ParseCertificate(block.Bytes)

    return
}

func quitCleanup(c chan os.Signal, ctx *websocket.Context) {
    _ = <-c

    ctx.Close()
}

func main() {
    var signalTrap chan os.Signal
    var cacert, cert, key, uri, url string
    var port int
    var forceCert bool

    // Parse command line arguments
    flag.StringVar(&url, "url", "", "URL accepted by the server. Empty defaults to any address/url.")
    flag.StringVar(&uri, "uri", "/proxy", "URI that references the WebSocket server.")
    flag.IntVar(&port, "port", 60000, "Port for the WebSocket server.")
    flag.StringVar(&cacert, "clientCacert", "", "Path to the CA certificate (that verifies clients' certificates).")
    flag.StringVar(&cert, "cert", "", "Path to the server's certificate.")
    flag.StringVar(&key, "key", "", "Path to the server's private key.")
    flag.BoolVar(&forceCert, "forceClientCert", false, "Requires users to supply a certificate (when using TLS).")
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

    // If desired, setup the connection over TLS
    if cert != "" && key != "" {
        tlsCert, err := tls.LoadX509KeyPair(cert, key)
        if err != nil {
            panic(err.Error())
        }

        config := &tls.Config{
            Certificates: []tls.Certificate{tlsCert},
            NextProtos:   []string{"wss"},
        }

        // Load the client CA, if supplied
        if cacert != "" {
            cert, err := parseCertificate(cacert)
            if err != nil {
                panic(err.Error())
            }

            pool := x509.NewCertPool()
            pool.AddCert(cert)
            config.ClientCAs = pool
            if !forceCert {
                config.ClientAuth = tls.VerifyClientCertIfGiven
            } else {
                config.ClientAuth = tls.RequireAndVerifyClientCert
            }
        }

        // Set the TLS configurations
        ctx.SetTlsConfig(config)
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
