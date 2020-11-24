package base

import (
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"time"
)

type (
	ChunkType byte
	Conn      struct {
		conn net.Conn
		used time.Time
		sync.Mutex
	}
)

const (
	ChunkCLS ChunkType = 0    //关闭连接
	ChunkOPN ChunkType = 1    //建立连接
	ChunkDAT ChunkType = 2    //数据传输
	ChunkCMD ChunkType = 3    //系统命令
	MTU                = 8192 //包头表示长度用了13bit（含2字节的包头）
)

//TODO: 目前wait参数都使用默认值60秒
//TODO: 目前的Conn使用了mutex，观察一下性能有无影响
func (c *Conn) Send(wait int, ct ChunkType, data []byte) (err error) {
	if c.conn == nil {
		return errors.New("base.Send: not connected")
	}
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("base.Send: %v", e)
			return
		}
		if ct != ChunkCMD {
			c.Lock()
			c.used = time.Now()
			c.Unlock()
		}
	}()
	clen := len(data) + 2
	if clen > MTU {
		panic(errors.New("data length exceeds MTU"))
	}
	if wait == 0 {
		wait = 60
	}
	if wait > 0 {
		assert(c.conn.SetWriteDeadline(time.Now().Add(time.Minute)))
		defer func() {
			e1 := recover()
			e2 := c.conn.SetWriteDeadline(time.Time{})
			assert(e1)
			assert(e2)
		}()
	}
	buf := make([]byte, MTU)
	buf[0] = byte(clen / 0x100)
	buf[1] = byte(clen % 0x100)
	buf[0] = (buf[0] & 0x1F) | (byte(ct) << 6) //目前bit-5未用，所以左移6位
	buf = append(buf, data...)
	_, err = c.conn.Write(buf[:clen])
	assert(err)
	return
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
			c.Lock()
			c.used = time.Now()
			c.Unlock()
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
	c.Lock()
	defer c.Unlock()
	if c.conn == nil {
		return nil
	}
	return c.conn.Close()
}

func (c *Conn) Idle(unused int) bool {
	c.Lock()
	defer c.Unlock()
	return time.Since(c.used).Seconds() >= float64(unused)
}

//本函数仅供后端使用，控制端的conn在NewConn时已经连接了
func (c *Conn) Connect(conn net.Conn, millis int) error {
	if conn != nil { //建立后端连接
		c.Lock()
		c.conn = conn
		c.Unlock()
		return nil
	}
	//或者等待后端连接
	for {
		c.Lock()
		conn = c.conn
		c.Unlock()
		if conn != nil {
			return nil
		}
		if millis <= 0 {
			return errors.New("base.Conn: wait connection timed out")
		}
		time.Sleep(10 * time.Millisecond)
		millis -= 10
	}
}

func NewConn(conn net.Conn) *Conn {
	return &Conn{conn: conn, used: time.Now()}
}
