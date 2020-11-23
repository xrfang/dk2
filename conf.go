package main

import (
	"dk/ctrl"
	"dk/serv"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v2"
)

type config struct {
	Mode    string      `yaml:"mode"`
	Debug   bool        `yaml:"debug"`
	Gateway ctrl.Config `yaml:"server"`
	Backend serv.Config `yaml:"client"`
	ULimit  uint64      `yaml:"ulimit" json:"ulimit"`
	Logging struct {
		Path  string `yaml:"path" json:"path"`
		Split int    `yaml:"split" json:"split"`
		Keep  int    `yaml:"keep" json:"keep"`
	} `yaml:"logging" json:"logging"`
	file string
}

func (c config) absPath(p string) string {
	if filepath.IsAbs(p) {
		return p
	}
	dir := filepath.Dir(c.file)
	return filepath.Clean(filepath.Join(dir, p))
}

var (
	cf config
	nr *regexp.Regexp
)

func init() {
	nr = regexp.MustCompile(`(?i)^[a-z0-9.-]{1,32}$`)
}

func loadConfig(fn string) {
	unifyMap := func(item string, m map[string]string) map[string]string {
		um := make(map[string]string)
		for k, v := range m {
			k = strings.TrimSpace(strings.ToLower(k))
			v = strings.TrimSpace(v)
			if !nr.MatchString(k) {
				panic(fmt.Errorf("loadConfig: %s must be 1~32 chars of alphanum, . or - (invalid entry `%s`)", item, k))
			}
			um[k] = v
		}
		return um
	}
	f, err := os.Open(fn)
	assert(err)
	defer f.Close()
	assert(err)
	cf.file, err = filepath.Abs(fn)
	assert(yaml.NewDecoder(f).Decode(&cf))
	cf.Mode = strings.ToLower(cf.Mode)
	switch cf.Mode {
	case "backend":
		if !nr.MatchString(cf.Backend.Name) {
			panic(fmt.Errorf("loadConfig: client.name must be 1~32 chars of alphanum, . or -"))
		}
		cf.Backend.Name = strings.ToLower(cf.Backend.Name)
		if cf.Backend.SvrPort <= 0 || cf.Backend.SvrPort > 65535 {
			cf.Backend.SvrPort = 35357
		}
		if cf.Backend.MacScan < 100 {
			cf.Backend.MacScan = 1000
		}
		if cf.Backend.MacScan > 5000 {
			cf.Backend.MacScan = 5000
		}
	case "gateway":
		if cf.Gateway.CtrlPort <= 0 || cf.Gateway.CtrlPort > 65535 {
			cf.Gateway.CtrlPort = 3535
		}
		if cf.Gateway.ServPort <= 0 || cf.Gateway.ServPort > 65535 {
			cf.Gateway.ServPort = 35350
		}
		if cf.Gateway.Handshake <= 0 || cf.Gateway.Handshake > 60 {
			cf.Gateway.Handshake = 10
		}
		if cf.Gateway.MaxServes <= 0 || cf.Gateway.MaxServes > 99 {
			cf.Gateway.MaxServes = 9
		}
		if cf.Gateway.IdleClose <= 0 || cf.Gateway.IdleClose > 3600 {
			cf.Gateway.IdleClose = 600
		}
		if cf.Gateway.AuthTime <= 0 || cf.Gateway.AuthTime > 86400 {
			cf.Gateway.AuthTime = 3600
		}
		if cf.Gateway.OTPIssuer == "" {
			cf.Gateway.OTPIssuer = "Door Keeper"
		}
		if cf.Gateway.Users == nil {
			cf.Gateway.Users = make(map[string]string)
		} else {
			cf.Gateway.Users = unifyMap("server.users", cf.Gateway.Users)
		}
		if cf.Gateway.Auths == nil {
			cf.Gateway.Auths = make(map[string]string)
		} else {
			cf.Gateway.Auths = unifyMap("server.auths", cf.Gateway.Auths)
		}
	default:
		panic(fmt.Errorf(`loadConfig: mode must be "backend" or "gateway"`))
	}
	if cf.ULimit == 0 {
		cf.ULimit = 1024
	}
	if cf.Logging.Path == "" {
		cf.Logging.Path = "../log"
	}
	cf.Logging.Path = cf.absPath(cf.Logging.Path)
	if cf.Logging.Split == 0 {
		cf.Logging.Split = 1024 * 1024 //每个log文件1兆字节
	}
	if cf.Logging.Keep == 0 {
		cf.Logging.Keep = 10 //最多保留10个LOG文件
	}
}
