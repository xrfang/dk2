package base

import (
	"encoding/binary"
	"net"
)

func Ping(conn net.Conn) error {
	buf, _ := Encode(ChunkCMD, []byte{0, 0, 0, 0, 0})
	return Send(conn, buf)
}

func Close(conn net.Conn, session uint32) error {
	id := make([]byte, 4)
	binary.BigEndian.PutUint32(id, session)
	buf, _ := Encode(ChunkCLS, id)
	return Send(conn, buf)
}

func Reply(conn net.Conn, session uint32, data []byte) error {
	id := make([]byte, 4)
	binary.BigEndian.PutUint32(id, session)
	buf, err := Encode(ChunkCMD, append(id, data...))
	if err != nil {
		return err
	}
	return Send(conn, buf)
}