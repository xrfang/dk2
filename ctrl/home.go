package ctrl

import (
	"net/http"
	"path/filepath"
	"strings"
)

func home(cf Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/":
			if allowed(r) {
				renderTemplate(w, "home.html", nil)
				return
			}
			user := r.URL.Query().Get("u")
			if user == "" {
				user = getCookie(r, "u")
			}
			if user == "" {
				return
			}
			renderTemplate(w, "login.html", struct{ User string }{user})
		case strings.HasPrefix(r.URL.Path, "/dk/"):
			http.Error(w, "Not Found", http.StatusNotFound)
		default:
			http.ServeFile(w, r, filepath.Join(cf.WebRoot, r.URL.Path))
		}
	}
}
