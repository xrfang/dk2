package base

import (
	"errors"
	"io"
	"net"
	"time"
)

type (
	ChunkType byte
	Conn      struct {
		conn net.Conn
		used time.Time
		data [][]byte
	}
)

const (
	ChunkCLS ChunkType = 0    //关闭连接
	ChunkOPN ChunkType = 1    //建立连接
	ChunkDAT ChunkType = 2    //数据传输
	ChunkCMD ChunkType = 3    //系统命令
	ChunkNUL ChunkType = 0xFF //保留内部使用
	MTU                = 8192 //包头表示长度用了13bit（含2字节的包头）
	TIMEOUT            = 60   //目前都使用默认值60秒
	backlog            = 1024 //最多缓存的包数，超过这个数字会丢包
)

var ErrInvalidChunk = errors.New("chunk size exceeds MTU")

func Encode(ct ChunkType, data []byte) ([]byte, error) {
	clen := len(data) + 2
	if clen > MTU {
		return nil, ErrInvalidChunk
	}
	buf := make([]byte, MTU)
	buf[0] = byte(clen / 0x100)
	buf[1] = byte(clen % 0x100)
	buf[0] = (buf[0] & 0x1F) | (byte(ct) << 6) //目前bit-5未用，所以左移6位
	buf = append(buf, data...)
	return buf[:clen], nil
}

func send(conn net.Conn, buf []byte) (err error) {
	deadline := time.Now().Add(time.Duration(TIMEOUT) * time.Second)
	if err = conn.SetWriteDeadline(deadline); err == nil {
		_, err = conn.Write(buf)
		if err == nil {
			err = conn.SetWriteDeadline(time.Time{})
		}
	}
	return err
}

func Recv(conn net.Conn) (ct ChunkType, data []byte, err error) {
	deadline := time.Now().Add(time.Duration(TIMEOUT) * time.Second)
	if err = conn.SetReadDeadline(deadline); err != nil {
		return
	}
	buf := make([]byte, MTU)
	if _, err = io.ReadFull(conn, buf[:2]); err != nil {
		return
	}
	ct = ChunkType((buf[0] & 0xC0) >> 6)             //目前bit-5未用，所以右移6位
	clen := int(buf[0]&0x1F)*0x100 + int(buf[1]) - 2 //byte-0的低5位（大端序），减去2字节包头
	if _, err = io.ReadFull(conn, buf[:clen]); err == nil {
		data = buf[:clen]
		err = conn.SetReadDeadline(time.Time{})
	}
	return
}

func (c *Conn) Send(ct ChunkType, data []byte) (err error) {
	if len(data) > 0 {
		buf, err := Encode(ct, data)
		if err != nil {
			return err
		}
		if len(c.data) < backlog {
			c.data = append(c.data, buf)
		}
	}
	if c.conn == nil {
		return nil
	}
	for _, buf := range c.data {
		if err = send(c.conn, buf); err != nil {
			return
		}
	}
	c.data = nil
	if ct == ChunkDAT {
		c.used = time.Now()
	}
	return nil
}

func (c *Conn) Close() error {
	c.data = nil
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

func (c *Conn) Connect(conn net.Conn) {
	if c.conn != nil {
		c.conn.Close()
	}
	c.conn = conn
}

func (c *Conn) Idle(unused int) bool {
	return time.Since(c.used).Seconds() >= float64(unused)
}

func NewConn(conn net.Conn) *Conn {
	return &Conn{conn: conn, used: time.Now()}
}
