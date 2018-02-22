package echo

import (
    "fmt"
    "github.com/SirGFM/GoWebSocketProxy/websocket"
    "math/rand"
)

type echoServer struct{
    // Store the connection to the client
    conn websocket.ClientConnection
    // Value that identifies this connection to the client
    uid  string
}

// Retrieves the template for the echo server
func GetTemplate() websocket.Server {
    return &echoServer{}
}

// Stores the connection (so we may echo the message) and create a unique
// identifier.
func (*echoServer) Clone(conn websocket.ClientConnection) (websocket.Server, error) {
    uid := make([]byte, 8)
    if _, err := rand.Read(uid); err != nil {
        return nil, err
    }

    ret := &echoServer{
        conn: conn,
        uid:  fmt.Sprintf("%#x - ", uid),
    }

    return ret, nil
}

// Echo the message back, prepending the connection identifier
func (es *echoServer) Do(msg []byte, offset int) (err error) {
    var payload []byte

    if string(msg[offset:]) == "close" {
        err = es.conn.Close(websocket.NormalClosure, []byte(es.uid))
    } else {
        payload = append(payload, es.uid...)
        // Skip the first 'offset' bytes, which contain the received WebSocket frame
        payload = append(payload, msg[offset:]...)

        // Wrap the message into a WebSocket frame
        err = es.conn.Send(payload, websocket.TextFrame)
    }

    return err
}

// No clean-up needed
func (*echoServer) Cleanup() {
}
