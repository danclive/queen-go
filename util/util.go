package util

import (
	"encoding/binary"
	"fmt"
)

func GetU32(b []byte, pos int) uint32 {
	return (uint32(b[pos+0])) |
		(uint32(b[pos+1]) << 8) |
		(uint32(b[pos+2]) << 16) |
		(uint32(b[pos+3]) << 24)
}

func UInt32ToBytes(i uint32) []byte {
	bytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(bytes, i)
	return bytes
}

func BytesToUInt32(b []byte) (uint32, error) {
	if len(b) != 4 {
		return 0, fmt.Errorf("[]byte(%v) len must == 4", b)
	}
	return binary.LittleEndian.Uint32(b), nil
}
