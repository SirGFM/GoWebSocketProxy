
package main

import (
    //"bufio"
    //"bytes"
    "encoding/binary"
    "fmt"
    "io"
    "net"
    //"net/http"
)

func control(conn net.Conn, stop *bool, proxyConn chan []byte) {
    for !*stop {
        //var buf bytes.Buffer
        var b [10240]byte

        n, err := conn.Read(b[:])
        if err == io.EOF {
            fmt.Println("Closing the controller!")
            return
        } else if err != nil {
            fmt.Printf("Controller failed to receive a request: %+v\n", err)
            continue
        }

        msg := b[:n]
        fmt.Printf("Received message: '%#x'\n", msg)

        // Check if message is masked
        if msg[1] & 0x80  == 0x80 {
            msg[1] &^= 0x80

            next := uint64(2)

            l := uint64(msg[1] & 0x7F)
            if l == 127 {
                next = 4
                l = uint64(binary.BigEndian.Uint16(msg[2:next]))
            } else if l == 128 {
                next = 10
                l = uint64(binary.BigEndian.Uint64(msg[2:next]))
            }

            key := msg[next:next+4]
            data := msg[next+4:]

            for dl, i := len(data), uint64(0); i < l && i < uint64(dl); i++ {
                msg[next+i] = data[i] ^ key[i & 0x3]
            }

            msg = msg[:next+l]
        }

        fmt.Printf("Forwarding message: '%#x'\n", msg)
        proxyConn <- msg

        /*
        req, err := http.ReadRequest(bufio.NewReader(conn))
        if netErr, ok := err.(*net.OpError); ok &&
            netErr.Err == net.ErrWriteToConnected {

            // Connection timed-out, continuing...
            continue
        } else if err == io.EOF {
            fmt.Println("Closing the controller!")
            return
        } else if err != nil {
            fmt.Printf("Controller failed to receive a request: %+v\n", err)
            continue
        }

        // Redirect the message to the view
        req.Write(&buf)
        b := buf.Bytes()
        fmt.Printf("control: Redirecting '%s'...\n", string(b))
        proxyConn <- b
        */
    }
}
