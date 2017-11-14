
package proxy

import (
    "encoding/json"
    "fmt"
)

type cmdStruct struct {
    Command string
    Name    string
    Data    []byte
}

func exec(buf []byte) {
    var err error
    var cmd cmdStruct

    fmt.Printf("Cmd: %#x (%s)\n", buf, string(buf))

    err = json.Unmarshal(buf, &cmd)
    if err != nil {
        fmt.Printf("Failed to received cmd: '%+v'\n", err)
    } else {
        fmt.Printf("Received cmd: '%+v'\n", cmd)
    }
}
