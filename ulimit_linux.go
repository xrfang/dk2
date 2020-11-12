//+build linux

package main

import (
	"fmt"
	"syscall"
)

func ulimit(soft uint64) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = e.(error)
		}
	}()
	var L syscall.Rlimit
	assert(syscall.Getrlimit(syscall.RLIMIT_NOFILE, &L))
	if soft > L.Max {
		soft = L.Max
	}
	L.Cur = soft
	assert(syscall.Setrlimit(syscall.RLIMIT_NOFILE, &L))
	assert(syscall.Getrlimit(syscall.RLIMIT_NOFILE, &L))
	if L.Cur != soft {
		return fmt.Errorf("Setrlimit: %v; Getrlimit: %v", soft, L.Cur)
	}
	return
}
