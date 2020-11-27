package ctrl

import (
	"math/rand"
	"time"
)

const chanLife = 9 * time.Second

type (
	replyChan struct {
		c chan interface{}
		t time.Time
	}
	replyChans map[uint32]*replyChan
)

var rc replyChans

func init() {
	rc = make(replyChans)
	rand.Seed(time.Now().UnixNano())
}

func setChan(c chan interface{}) uint32 {
	idx := rand.Uint32()
	rc[idx] = &replyChan{c, time.Now()}
	return idx
}

func getChan(idx uint32) chan interface{} {
	r := rc[idx]
	delete(rc, idx)
	for i, c := range rc {
		if time.Since(c.t) > chanLife {
			delete(rc, i)
		}
	}
	if r == nil {
		return nil
	}
	return r.c
}
