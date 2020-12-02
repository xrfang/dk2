package ctrl

import (
	"net/http"
)

func notFound(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Not Found", http.StatusNotFound)
}

func setupRoutes(cf Config) {
	http.HandleFunc("/", home(cf))
	http.HandleFunc("/dk/login", apiLogin(cf))
	http.HandleFunc("/dk/auth", apiAuth)
	http.HandleFunc("/dk/site", apiSite)
	http.HandleFunc("/dk/port", notFound)
	http.HandleFunc("/dk/port/", apiScan)
	http.HandleFunc("/dk/conn", notFound)
	http.HandleFunc("/dk/conn/", apiConn)
}
