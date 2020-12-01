package ctrl

import (
	"fmt"
	"net/http"
	"strings"
)

func notFound(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Not Found", http.StatusNotFound)
}

func setupRoutes(cf Config) {
	http.HandleFunc("/", home)
	http.HandleFunc("/dk/login", apiLogin(cf))
	http.HandleFunc("/dk/auth", apiAuth)
	http.HandleFunc("/dk/site", apiSite)
	http.HandleFunc("/dk/port", notFound)
	http.HandleFunc("/dk/port/", apiScan)
	http.HandleFunc("/dk/conn", notFound)
	http.HandleFunc("/dk/conn/", apiConn)
}

func home(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.URL.Path == "/":
		fmt.Fprintln(w, "TODO: show admin web page")
	case strings.HasPrefix(r.URL.Path, "/dk/"):
		http.Error(w, "Not Found", http.StatusNotFound)
	default:
		fmt.Fprintln(w, "TODO: load resource "+r.URL.Path)
	}
}
