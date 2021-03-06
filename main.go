package main

import (
	"bufio"
	"dk/base"
	"dk/ctrl"
	"dk/serv"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/mdp/qrterminal"
	"github.com/pquerna/otp/totp"
	"github.com/xrfang/go-res"
	"gopkg.in/yaml.v2"
)

func main() {
	ver := flag.Bool("version", false, "show version info")
	cfg := flag.String("conf", "", "configuration file")
	init := flag.Bool("init", false, "create sample configuration "+
		"(without -conf), or\nreset OTP key (with -conf)")
	flag.Usage = func() {
		fmt.Printf("DoorKeeper %s\n\n", verinfo())
		fmt.Printf("USAGE: %s [OPTIONS]\n\n", filepath.Base(os.Args[0]))
		fmt.Printf("OPTIONS:\n\n")
		flag.PrintDefaults()
	}
	flag.Parse()
	if *ver {
		fmt.Println(verinfo())
		return
	}
	if len(flag.Args()) > 0 {
		fmt.Printf("invalid command line arguments: %s\n\n", strings.Join(flag.Args(), " "))
		flag.Usage()
		os.Exit(1)
	}
	if *cfg == "" {
		if *init {
			f, err := ioutil.TempFile(".", "dk.yaml.")
			assert(err)
			defer f.Close()
			fmt.Fprintln(f, SAMPLE_CFG)
			fmt.Println("sample configuration:", f.Name())
		} else {
			fmt.Println("ERROR: missing configuration (-conf), try -h for help")
		}
		return
	}
	loadConfig(*cfg)
	if *init {
		if cf.Mode == "gateway" {
			nx := regexp.MustCompile(`^\w{1,32}$`)
			r := bufio.NewReader(os.Stdin)
			fmt.Print("username: ")
			login, _ := r.ReadString('\n')
			login = strings.TrimSpace(login)
			if !nx.MatchString(login) {
				fmt.Printf("ERROR: username should consit of 1~32 characters of alphanum or `_`")
				os.Exit(1)
			}
			gopts := totp.GenerateOpts{
				AccountName: login,
				Issuer:      cf.Gateway.OTPIssuer,
			}
			key, err := totp.Generate(gopts)
			assert(err)
			qrterminal.Generate(key.String(), qrterminal.L, os.Stdout)
			cf.Gateway.Users[login] = key.Secret()
			f, err := os.Create(*cfg)
			assert(err)
			defer f.Close()
			ye := yaml.NewEncoder(f)
			assert(ye.Encode(&cf))
		} else {
			fmt.Println("OTP key initialization is for DK gateway only (given backend config)")
		}
		return
	}
	base.InitLogger(cf.Logging.Path, cf.Logging.Split, cf.Logging.Keep, cf.Debug)
	if err := ulimit(cf.ULimit); err != nil {
		base.Log("ulimit(): %v", err)
	}
	switch cf.Mode {
	case "backend":
		serv.Start(cf.Backend)
	case "gateway":
		if len(cf.Gateway.Users) == 0 {
			fmt.Fprintln(os.Stderr, `ERROR: no user defined (gateway.users), use "-init" to generate`)
			return
		}
		if len(cf.Gateway.Auths) == 0 {
			fmt.Fprintln(os.Stderr, `ERROR: no auth defined (gateway.auths)`)
			return
		}
		policy := res.Verbatim
		if cf.Debug {
			policy = res.OverwriteIfNewer
		}
		assert(res.Extract(cf.Gateway.WebRoot, policy))
		ctrl.Start(cf.Gateway)
	}
}
