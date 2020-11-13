package cli

import (
	"fmt"
	"net"
	"sync"
	"time"
)

func portScan(port int, cidrs []string, timeout int) (ips []string) {
	resc := make(chan string)
	go func() { //收集结果
		for {
			ip := <-resc
			if ip == "" {
				break
			}
			ips = append(ips, ip)
		}
	}()
	const (
		THREADS = 256 //并发线程数
		TIMEOUT = 500 * time.Millisecond
	)
	task := make(chan string, THREADS)
	var wg sync.WaitGroup
	tts := time.Duration(timeout) * time.Millisecond
	for i := 0; i < THREADS; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				ip := <-task
				if ip == "" {
					break
				}
				target := fmt.Sprintf("%s:%d", ip, port)
				conn, err := net.DialTimeout("tcp", target, TIMEOUT)
				if err == nil {
					conn.Close()
					ip += getOUI(ip, tts)
					resc <- ip
				}
			}
		}()
	}
	inc := func(ip net.IP) {
		for j := len(ip) - 1; j >= 0; j-- {
			ip[j]++
			if ip[j] > 0 {
				break
			}
		}
	}
	for _, cidr := range cidrs {
		ip, ipnet, err := net.ParseCIDR(cidr)
		if err != nil {
			continue
		}
		for ip := ip.Mask(ipnet.Mask); ipnet.Contains(ip); inc(ip) {
			task <- ip.String()
		}
	}
	for i := 0; i < THREADS; i++ {
		task <- ""
	}
	wg.Wait()  //等待所有工作线程结束
	resc <- "" //通知收集线程结束
	return
}
