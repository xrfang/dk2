package ctrl

import (
	"fmt"
	"net/http"
	"strings"
)

func setupRoutes() {
	http.HandleFunc("/", home)
	http.HandleFunc("/dk/site", apiSite)
	http.HandleFunc("/dk/port/", apiScan)
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
