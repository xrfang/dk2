package ctrl

import (
	"dk/base"
	"encoding/json"
	"html/template"
	"net/http"
	"path"
	"path/filepath"
	"time"
)

var (
	version string
	webroot string
)

func setEnv(cf Config) {
	version = cf.Version
	webroot = cf.WebRoot
}

func getCookie(r *http.Request, name string) string {
	c, err := r.Cookie(name)
	if err != nil {
		return ""
	}
	return c.Value
}

func setCookie(w http.ResponseWriter, name, value string, age int) {
	exp := time.Now().Add(time.Duration(age) * time.Second)
	http.SetCookie(w, &http.Cookie{
		Name:    name,
		Value:   value,
		Path:    "/",
		MaxAge:  age,
		Expires: exp,
		Secure:  false,
	})
}

func httpTrap(w http.ResponseWriter) func() {
	return func() {
		if e := recover(); e != nil {
			base.Err("%v", e)
			http.Error(w, e.(error).Error(), http.StatusInternalServerError)
		}
	}
}

func renderTemplate(w http.ResponseWriter, tpl string, args interface{}) {
	defer httpTrap(w)()
	helper := template.FuncMap{
		"ver": func() string { return version },
	}
	tDir := path.Join(webroot, "templates")
	t, err := template.New("body").Funcs(helper).ParseFiles(path.Join(tDir, tpl))
	assert(err)
	sfs, err := filepath.Glob(path.Join(tDir, "shared/*"))
	if len(sfs) > 0 {
		t, err = t.ParseFiles(sfs...)
		assert(err)
	}
	w.Header().Add("Content-Type", "text/html; charset=utf-8")
	w.Header().Add("Cache-Control", "no-store")
	assert(t.Execute(w, args))
}

func jsonReply(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Add("Cache-Control", "no-store")
	je := json.NewEncoder(w)
	je.SetIndent("", "    ")
	assert(je.Encode(data))
}
