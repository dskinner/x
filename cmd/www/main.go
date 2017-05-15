// Command www provides vanity imports and redirects.
package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"
)

const meta = `<html><head>
<meta name="go-import" content="dasa.cc/%[1]s git https://github.com/dskinner/%[1]s.git">
</head></html>`

func exists(repo string) bool {
	resp, err := http.Head(fmt.Sprintf("https://api.github.com/repos/dskinner/%s", repo))
	return err == nil && 200 <= resp.StatusCode && resp.StatusCode < 400
}

func handle(w http.ResponseWriter, r *http.Request) {
	repo := strings.Split(r.URL.Path, "/")[1]
	switch {
	case repo == "":
		http.Redirect(w, r, "https://github.com/dskinner", http.StatusMovedPermanently)
	case r.FormValue("go-get") == "1":
		fmt.Fprintf(w, meta, repo)
	case exists(repo):
		http.Redirect(w, r, fmt.Sprintf("https://godoc.org/dasa.cc/%s", repo), http.StatusMovedPermanently)
	default:
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
	}
}

func health(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "ok")
}

func main() {
	http.HandleFunc("/", handle)
	http.HandleFunc("/_ah/health", health)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
