package base

import (
	"bytes"
	"encoding/binary"
	"io"
	"net"
	"time"
)

const (
	CT_CLS = 0    //关闭连接
	CT_DAT = 1    //数据传输
	CT_QRY = 2    //询问开放端口
	CT_PNG = 3    //通道心跳检测
	MTU    = 4000 //传输块长度必须小于4096，因为包头表示长度用了12bit
)

type Chunk struct {
	Type byte
	Src  *net.TCPAddr
	Dst  *net.TCPAddr
	Buf  []byte
}

func (c Chunk) Serialize() []byte {
	var tag byte
	switch c.Type {
	case CT_DAT:
		tag = 0x40 //0100 0000
	case CT_QRY:
		tag = 0x80 //1000 0000
	case CT_PNG:
		tag = 0xC0 //1100 0000
	}
	buf := bytes.NewBuffer(nil)
	putIP := func(ip net.IP, v6Mark byte) {
		p := ip.To4()
		if p == nil {
			buf.Write([]byte(ip))
			tag |= v6Mark
			return
		}
		buf.Write([]byte(p))
	}
	buf.Write([]byte{0, 0})
	binary.Write(buf, binary.BigEndian, uint16(c.Src.Port))
	putIP(c.Src.IP, 0x20)
	binary.Write(buf, binary.BigEndian, uint16(c.Dst.Port))
	putIP(c.Dst.IP, 0x10)
	buf.Write(c.Buf)
	res := buf.Bytes()
	binary.BigEndian.PutUint16(res, uint16(len(c.Buf)))
	res[0] = res[0] | tag
	return res
}

func (c *Chunk) Recv(conn *net.TCPConn) (err error) {
	defer func() {
		if e := recover(); e != nil {
			ne, ok := e.(net.Error)
			if ok && ne.Timeout() {
				err = ne
			} else if e != io.EOF {
				err = trace("%v", e)
			}
		}
	}()
	assert(conn.SetReadDeadline(time.Now().Add(time.Minute)))
	defer func() {
		assert(conn.SetReadDeadline(time.Time{}))
	}()
	buf := make([]byte, MTU)
	getAddr := func(isV6 bool) *net.TCPAddr {
		var addr net.TCPAddr
		alen := 6 //port + IPv4长度
		if isV6 {
			alen = 18 //port + IPv6长度
		}
		_, err := io.ReadFull(conn, buf[:alen])
		assert(err)
		addr.Port = (int(buf[0]) << 8) + int(buf[1])
		ip := make([]byte, alen-2)
		copy(ip, buf[2:alen])
		addr.IP = net.IP(ip)
		return &addr
	}
	_, err = io.ReadFull(conn, buf[:2])
	assert(err)
	tag := buf[0] >> 4
	blen := (uint16(buf[0]&0xF) << 8) + uint16(buf[1])
	c.Type = tag >> 2
	c.Src = getAddr((tag & 2) != 0)
	c.Dst = getAddr((tag & 1) != 0)
	_, err = io.ReadFull(conn, buf[:blen])
	assert(err)
	c.Buf = buf[:blen]
	return
}

func (c Chunk) Send(conn *net.TCPConn) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = trace("base.Chunk.Send(): %v", e)
		}
	}()
	assert(conn.SetWriteDeadline(time.Now().Add(time.Minute)))
	defer func() {
		assert(conn.SetWriteDeadline(time.Time{}))
	}()
	_, err = conn.Write(c.Serialize())
	assert(err)
	return
}
