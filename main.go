package main

import (
	"bufio"
	"dk/base"
	"dk/cli"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/mdp/qrterminal"
	"github.com/pquerna/otp/totp"
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
		if cf.Mode == "server" {
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
				Issuer:      cf.Server.OTPIssuer,
			}
			key, err := totp.Generate(gopts)
			assert(err)
			qrterminal.Generate(key.String(), qrterminal.L, os.Stdout)
			cf.Server.Users[login] = key.Secret()
			f, err := os.Create(*cfg)
			assert(err)
			defer f.Close()
			ye := yaml.NewEncoder(f)
			assert(ye.Encode(&cf))
		} else {
			fmt.Println("OTP key initialization is for DK server only (given client config)")
		}
		return
	}
	base.InitLogger(cf.Logging.Path, cf.Logging.Split, cf.Logging.Keep, cf.Debug)
	if err := ulimit(cf.ULimit); err != nil {
		base.Log("ulimit(): %v", err)
	}
	switch cf.Mode {
	case "client":
		cli.Start(cf.Client)
	case "server":
		if len(cf.Server.Users) == 0 {
			fmt.Fprintln(os.Stderr, `ERROR: no user defined (server.users), use "-init" to generate`)
			return
		}
		if len(cf.Server.Auths) == 0 {
			fmt.Fprintln(os.Stderr, `ERROR: no client-auth defined (server.auths)`)
			return
		}
		fmt.Printf("todo: start server...")
		//	svr.Start(cf.Server)
	}
}
