package base

import (
	"errors"
	"net"
	"time"
)

// const (
// CT_CLS = 0    //关闭连接
// CT_OPN = 1    //建立连接
// CT_DAT = 2    //数据传输
// CT_CMD = 3    //系统命令
// MTU    = 8190 //包头表示长度用了13bit，另外有2字节的包头
// )

type (
	Pipe struct {
		Session uint32
		Inner   net.Conn
		Outer   net.Conn
		LastUse time.Time
	}
	Pinger struct {
		Interval time.Duration
		Pipe
	}
)

func NewPinger(c net.Conn, interval int) *Pinger {
	return &Pinger{
		time.Duration(interval) * time.Second,
		Pipe{
			Inner:   c,
			LastUse: time.Now(),
		},
	}
}

func send(conn net.Conn, buf []byte) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = trace("base.send(): %v", e)
		}
	}()
	assert(conn.SetWriteDeadline(time.Now().Add(time.Minute)))
	defer func() {
		assert(conn.SetWriteDeadline(time.Time{}))
	}()
	_, err = conn.Write(buf)
	assert(err)
	return
}

//KeepAlive 一律由DKG发起，DKS不回复
func (p *Pinger) KeepAlive() (err error) {
	if p.Inner == nil {
		return errors.New("no connection")
	}
	if p.Interval <= 0 {
		return errors.New("ping deactivated")
	}
	for {
		p.Interval = time.Second //TODO：调试期间设为1秒
		time.Sleep(p.Interval)
		err = send(p.Inner, []byte{0xC0, 3, 0})
		if err != nil {
			return err
		}
	}
}
