//+build windows

package main

func ulimit(soft uint64) error {
	L.Log("ulimit() not supported on windows")
	return nil
}
