package ctrl

import (
	"fmt"
	"net/http"
	"os"
	"time"
)

const httpSvrTimeout = time.Minute

func startAdminInterface(cf Config) {
	setEnv(cf)
	setWatchdog(cf)
	go func() {
		defer func() {
			fmt.Fprintf(os.Stderr, "startAdminInterface: %v", recover())
			os.Exit(1)
		}()
		setupRoutes(cf)
		svr := http.Server{
			Addr:         fmt.Sprintf(":%d", cf.MgmtPort),
			ReadTimeout:  httpSvrTimeout,
			WriteTimeout: httpSvrTimeout,
		}
		assert(svr.ListenAndServe())
	}()
}
