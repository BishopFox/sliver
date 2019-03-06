package encoders

import "io"

// Encoder - Can losslessly encode arbitrary binary data
type Encoder func(io.Writer, []byte) error

// Decoder - Can decode a losslessly encoded buffer
type Decoder func([]byte) ([]byte, error)
