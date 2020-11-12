package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"dk/cli"
	"dk/svr"

	"gopkg.in/yaml.v2"
)

type config struct {
	Mode    string     `yaml:"mode"`
	Debug   bool       `yaml:"debug"`
	Server  svr.Config `yaml:"server"`
	Client  cli.Config `yaml:"client"`
	ULimit  uint64     `yaml:"ulimit" json:"ulimit"`
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
	case "client":
		if !nr.MatchString(cf.Client.Name) {
			panic(fmt.Errorf("loadConfig: client.name must be 1~32 chars of alphanum, . or -"))
		}
		cf.Client.Name = strings.ToLower(cf.Client.Name)
		if cf.Client.SvrPort <= 0 || cf.Client.SvrPort > 65535 {
			cf.Client.SvrPort = 35357
		}
		if cf.Client.MacScan < 100 {
			cf.Client.MacScan = 1000
		}
		if cf.Client.MacScan > 5000 {
			cf.Client.MacScan = 5000
		}
	case "server":
		if cf.Server.CtrlPort <= 0 || cf.Server.CtrlPort > 65535 {
			cf.Server.CtrlPort = 3535
		}
		if cf.Server.ServPort <= 0 || cf.Server.ServPort > 65535 {
			cf.Server.ServPort = 35350
		}
		if cf.Server.Handshake <= 0 || cf.Server.Handshake > 60 {
			cf.Server.Handshake = 10
		}
		if cf.Server.IdleClose <= 0 || cf.Server.IdleClose > 3600 {
			cf.Server.IdleClose = 600
		}
		if cf.Server.AuthTime <= 0 || cf.Server.AuthTime > 86400 {
			cf.Server.AuthTime = 3600
		}
		if cf.Server.OTPIssuer == "" {
			cf.Server.OTPIssuer = "Door Keeper"
		}
		if cf.Server.Users == nil {
			cf.Server.Users = make(map[string]string)
		} else {
			cf.Server.Users = unifyMap("server.users", cf.Server.Users)
		}
		if cf.Server.Auths == nil {
			cf.Server.Auths = make(map[string]string)
		} else {
			cf.Server.Auths = unifyMap("server.auths", cf.Server.Auths)
		}
	default:
		panic(fmt.Errorf(`loadConfig: mode must be "client" or "server"`))
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
