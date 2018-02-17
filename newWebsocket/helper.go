package newWebsocket

import (
    "encoding/binary"
    "github.com/pkg/errors"
    "net"
    "time"
)

// expandBuffer in so it has at least min bytes and place it in out.
func expandBuffer(in []byte, min int) (out []byte) {
    if len(in) < min {
        out = make([]byte, min)
        copy(out, in)
    } else {
        out = in
    }

    return
}

// readFullLength from conn, expanding buf as needed (only so length more bytes
// fit into it).
func readFullLength(conn net.Conn, buf []byte, length int) (retBuf []byte,
    err error) {

    var n int
    var gerr error

    retBuf = expandBuffer(buf, MinHeaderLength+length)

    conn.SetReadDeadline(time.Now().Add(time.Second))
    n, gerr = conn.Read(retBuf[MinHeaderLength:MinHeaderLength+length])
    if gerr != nil {
        err = errors.Wrap(err, "websocket: Failed to read message's length")
    } else if n != length {
        err = errors.New("websocket: Failed to read the entire message's length")
    }

    return
}

// receiveFrame from conn on buf. MinHeaderLength bytes should have been read into
// buf before calling this function, so frame length may be read.
//
// buf will be expand as needed and later returned into retBuf. msgLen is the
// length of "PayloadData", which may be read starting on offset. (i.e., the
// payload should be read as retBuf[offset:msgLen])
func receiveFrame(conn net.Conn, buf []byte) (retBuf []byte, msgLen, offset int,
    err error) {

    var gerr error
    var key [MaskLength]byte
    var n int

    if buf[MaskIndex] & MaskBit == 0 {
        err = errors.New("websocket: Message is missing its mask bits")
        return
    }
    // Remove the mask
    buf[MaskIndex] &^= MaskBit

    // By default, "Payload Data" starts after the basic header (and
    // "Masking-key" will be removed from the message).
    offset = MinHeaderLength

    // Read "Payload Data" length, expanding the buffer as necessary.
    msgLen = int(buf[LengthIndex] & LengthMask)
    switch msgLen {
    case Extended16BitLength:
        retBuf, err = readFullLength(conn, buf, 2)
        if err != nil {
            return
        }

        offset += 2
        msgLen = int(binary.BigEndian.Uint16(retBuf[MinHeaderLength:offset]))
    case Extended64BitLength:
        retBuf, err = readFullLength(conn, buf, 8)
        if err != nil {
            return
        }

        offset += 8
        len64 := uint64(binary.BigEndian.Uint64(retBuf[MinHeaderLength:offset]))
        if len64 >= 0x100000000 {
            err = errors.New("websocket: Message bigger than max int32")
            return
        }
        msgLen = int(len64 & 0x7fffffff)
    default:
        retBuf = buf
    }

    retBuf = expandBuffer(retBuf, offset+msgLen)

    // Read "Masking-key"
    conn.SetReadDeadline(time.Now().Add(time.Second))
    n, gerr = conn.Read(key[:])
    if gerr != nil {
        err = errors.Wrap(gerr, "websocket: Failed to read masking-key")
        return
    } else if n != MaskLength {
        err = errors.New( "websocket: Failed to read the entire masking-key")
        return
    }

    // Finish reading the message
    conn.SetReadDeadline(time.Now().Add(time.Second))
    n, gerr = conn.Read(retBuf[offset:offset+msgLen])
    if gerr != nil {
        err = errors.Wrap(gerr, "websocket: Failed to read the message")
        return
    } else if n != msgLen {
        err = errors.New( "websocket: Failed to read the entire message")
        return
    }

    // Unmask the message
    payload := retBuf[offset:]
    for i := 0; i < msgLen; i++ {
        payload[i] = payload[i] ^ key[i & 0x3]
    }

    return
}
