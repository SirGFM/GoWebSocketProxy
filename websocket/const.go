
package websocket

import (
    "github.com/pkg/errors"
)

// Error returned when a message is bigger than supported.
var MessageToBig = errors.New("Message bigger than max int32")

// Minimal length (in bytes) of a message from a WebSocket.
const MinHeaderLength = 2

// Length of the "Masking-key".
const MaskLength = 4

// Index of the opcode within a Frame.
const OpcodeIndex = 0

// Mask to retrieve the opcode from its byte in the Frame.
const OpcodeMask = 0x0F

// Index of the mask bit within a Frame.
const MaskIndex = 1

// Bit used to check whether "Payload data" is masked.
const MaskBit = 0x80

// Index of the opcode within a Frame.
const LengthIndex = 1

// Mask to retrieve the opcode from its byte in the Frame.
const LengthMask = 0x7F

// Constant used on the Length field if the length actually has 16 bits.
const Extended16BitLength = 126

// Constant used on the Length field if the length actually has 64 bits.
const Extended64BitLength = 127

// Defines the interpretation of the "Payload data". Only the lowest 4 bits are
// used.
type Opcode uint8
const (
    ContinuationFrame Opcode = 0x0
    TextFrame         Opcode = 0x1
    BinaryFrame       Opcode = 0x2
    // 0x3...0x7: reserved for further non-control frames
    ConnectionClose Opcode = 0x8
    Ping            Opcode = 0x9
    Pong            Opcode = 0xA
    // 0xB...0xF: reserved for further control frames
)

// Default message to be used for heart-beats.
var HeartBeatMessage = []byte{0x80 | byte(Pong), 0x00}
