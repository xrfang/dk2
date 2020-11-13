//+build windows

package cli

import (
	"errors"
	"time"
)

func getMAC(ip string, timeout time.Duration) (mac string, err error) {
	return "", errors.New("cannot get MAC on windows")
}
