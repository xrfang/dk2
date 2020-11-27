package ctrl

import (
	"fmt"
	"net/http"
)

func setupRoutes() {
	http.HandleFunc("/", home)
	http.HandleFunc("/dk/site", apiSite)
	http.HandleFunc("/dk/port/", apiScan)
}

func home(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "TODO...")
}
