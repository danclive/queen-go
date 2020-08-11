package util

import (
	"bytes"
	"encoding/binary"
)

func WriteInt32(buf *bytes.Buffer, i int32) error {
	return binary.Write(buf, binary.LittleEndian, i)
}

func ReadInt32(rd *bytes.Buffer) (int32, error) {
	var i int32
	if err := binary.Read(rd, binary.LittleEndian, &i); err != nil {
		return 0, err
	}
	return i, nil
}

func GetI32(b []byte, pos int) int32 {
	return (int32(b[pos+0])) |
		(int32(b[pos+1]) << 8) |
		(int32(b[pos+2]) << 16) |
		(int32(b[pos+3]) << 24)
}
