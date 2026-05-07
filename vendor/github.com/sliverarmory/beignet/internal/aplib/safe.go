package aplib

import (
	"encoding/binary"
	"errors"
	"hash/crc32"
)

const (
	safeTag        = 0x32335041 // 'AP32'
	safeHeaderSize = 24
)

var ErrBadHeader = errors.New("aplib: bad safe header")

// PackSafe returns a safe-packed aPLib buffer (AP32 header + raw stream).
func PackSafe(src []byte) ([]byte, error) {
	if len(src) == 0 {
		return nil, ErrEmptyInput
	}
	if uint64(len(src)) > uint64(^uint32(0)) {
		return nil, ErrTooLarge
	}

	packed, err := Pack(src)
	if err != nil {
		return nil, err
	}
	if uint64(len(packed)) > uint64(^uint32(0)) {
		return nil, ErrTooLarge
	}

	hdr := make([]byte, safeHeaderSize)
	binary.LittleEndian.PutUint32(hdr[0:4], safeTag)
	binary.LittleEndian.PutUint32(hdr[4:8], safeHeaderSize)
	binary.LittleEndian.PutUint32(hdr[8:12], uint32(len(packed)))
	binary.LittleEndian.PutUint32(hdr[12:16], crc32.ChecksumIEEE(packed))
	binary.LittleEndian.PutUint32(hdr[16:20], uint32(len(src)))
	binary.LittleEndian.PutUint32(hdr[20:24], crc32.ChecksumIEEE(src))

	out := make([]byte, 0, len(hdr)+len(packed))
	out = append(out, hdr...)
	out = append(out, packed...)
	return out, nil
}

// DepackSafe depacks an AP32 header + aPLib buffer.
func DepackSafe(src []byte) ([]byte, error) {
	if len(src) < safeHeaderSize {
		return nil, ErrBadHeader
	}
	if binary.LittleEndian.Uint32(src[0:4]) != safeTag {
		return nil, ErrBadHeader
	}
	headerSize := binary.LittleEndian.Uint32(src[4:8])
	if headerSize < safeHeaderSize || headerSize > uint32(len(src)) {
		return nil, ErrBadHeader
	}
	packedSize := binary.LittleEndian.Uint32(src[8:12])
	if uint64(headerSize)+uint64(packedSize) > uint64(len(src)) {
		return nil, ErrBadHeader
	}
	origSize := binary.LittleEndian.Uint32(src[16:20])

	body := src[headerSize : headerSize+packedSize]
	out, err := Depack(body)
	if err != nil {
		return nil, err
	}
	if uint32(len(out)) != origSize {
		return nil, ErrCorrupt
	}
	return out, nil
}
