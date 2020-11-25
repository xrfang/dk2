package base

import (
	"errors"
	"fmt"
	"io"
	"net"
	"time"
)

type (
	ChunkType byte
	chunk     struct {
		cls ChunkType
		buf []byte
	}
	Conn struct {
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
	TIMEOUT            = 60   //TODO: 目前都使用默认值60秒
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

func Send(conn net.Conn, buf []byte) (err error) {
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
		if err = Send(c.conn, buf); err != nil {
			return
		}
	}
	c.data = nil
	if ct == ChunkDAT {
		c.used = time.Now()
	}
	return nil
}

func (c *Conn) Recv(wait int) (ct ChunkType, data []byte, err error) {
	if c.conn == nil {
		return 0, nil, errors.New("base.Recv: not connected")
	}
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("base.Recv: %v", e)
			return
		}
		if ct != ChunkCMD {
			c.used = time.Now()
		}
	}()
	if wait == 0 {
		wait = 60
	}
	if wait > 0 {
		assert(c.conn.SetReadDeadline(time.Now().Add(time.Duration(wait) * time.Second)))
		defer func() {
			e1 := recover()
			e2 := c.conn.SetReadDeadline(time.Time{})
			assert(e1)
			assert(e2)
		}()
	}
	buf := make([]byte, MTU)
	_, err = io.ReadFull(c.conn, buf[:2])
	assert(err)
	ct = ChunkType((buf[0] & 0xC0) >> 6)             //目前bit-5未用，所以右移6位
	clen := int(buf[0]&0x1F)*0x100 + int(buf[1]) - 2 //byte-0的低5位（大端序），减去2字节包头
	_, err = io.ReadFull(c.conn, buf[:clen])
	assert(err)
	return ct, buf[:clen], nil
}

func (c *Conn) Close() error {
	c.data = nil
	var err error
	if c.conn != nil {
		err = c.conn.Close()
	}
	return err
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

func (c *Conn) Connected() bool {
	return c.conn != nil
}

func NewConn(conn net.Conn) *Conn {
	c := &Conn{conn: conn, used: time.Now()}
	/*
		go func() {
			t := time.NewTicker(time.Second)
			ic := float64(idleClose)
			wait := time.Duration(0)
			switch {
			case TIMEOUT == 0:
				wait = time.Minute
			case TIMEOUT > 0:
				wait = time.Duration(TIMEOUT) * time.Second
			}
			var err error
			for {
				if c.conn
				select {
				case p := <-c.data:
					err = func() (err error) {
						defer func() {
							if e := recover(); e != nil {
								err = e.(error)
								return
							}
							if p.cls != ChunkCMD {
								c.used = time.Now()
							}
						}()
						if wait > 0 {
							assert(c.conn.SetWriteDeadline(time.Now().Add(wait)))
							defer func() {
								e1 := recover()
								e2 := c.conn.SetWriteDeadline(time.Time{})
								assert(e1)
								assert(e2)
							}()
						}
						clen := len(p.buf) + 2
						buf := make([]byte, MTU)
						buf[0] = byte(clen / 0x100)
						buf[1] = byte(clen % 0x100)
						buf[0] = (buf[0] & 0x1F) | (byte(p.cls) << 6) //目前bit-5未用，所以左移6位
						buf = append(buf, p.buf...)
						_, err = c.conn.Write(buf[:clen])
						assert(err)
						return
					}()
					if err != nil {
						break
					}
				case <-t.C:
					if ic > 0 && time.Since(c.used).Seconds() >= ic {
						err = errors.New("idle close")
						break
					}
				}
			}
			Log("base.Conn: %v", err)
			c.conn.Close()
			close(c.data)
		}()
	*/
	return c
}
