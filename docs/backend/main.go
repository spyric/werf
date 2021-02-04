package main

import (
	"fmt"
	"github.com/gorilla/mux"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
)

func getPage(filename string) ([]byte, error) {
	body, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return body, nil
}

func inspectRequest(r *http.Request) string {
	var request []string

	url := fmt.Sprintf("%v %v %v", r.Method, r.URL, r.Proto)
	request = append(request, url)
	request = append(request, fmt.Sprintf("Host: %v", r.Host))
	for name, headers := range r.Header {
		name = strings.ToLower(name)
		for _, h := range headers {
			request = append(request, fmt.Sprintf("%v: %v", name, h))
		}
	}

	// If this is a POST, add post data
	if r.Method == "POST" {
		_ = r.ParseForm()
		request = append(request, "\n")
		request = append(request, r.Form.Encode())
	}

	return strings.Join(request, "\n")
}

func inspectRequestHTML(r *http.Request) string {
	var request []string

	request = append(request, "<p>")
	url := fmt.Sprintf("<b>%v</b> %v %v", r.Method, r.URL, r.Proto)
	request = append(request, url)
	request = append(request, fmt.Sprintf("<b>Host:</b> %v", r.Host))
	for name, headers := range r.Header {
		name = strings.ToLower(name)
		for _, h := range headers {
			request = append(request, fmt.Sprintf("<b>%v:</b> %v", name, h))
		}
	}

	// If this is a POST, add post data
	if r.Method == "POST" {
		_ = r.ParseForm()
		request = append(request, r.Form.Encode())
	}

	request = append(request, "</p>")
	return strings.Join(request, "<br />")
}

func newRouter() *mux.Router {
	r := mux.NewRouter()

	staticFileDirectory := http.Dir("./root/")

	r.PathPrefix("/backend/").HandlerFunc(ssiHandler).Methods("GET")
	r.PathPrefix("/").Handler(http.FileServer(staticFileDirectory))
	return r
}

func mainHandler(w http.ResponseWriter, r *http.Request) {
	_, _ = fmt.Fprintf(w, "Main handler (%s).", r.URL.Path[1:])
}

func ssiHandler(w http.ResponseWriter, r *http.Request) {
	_, _ = fmt.Fprintf(w, "<p>SSI handler (%s).</p>", r.URL.Path[1:])
	_, _ = fmt.Fprintf(w, inspectRequestHTML(r))
}

func main() {
	r := newRouter()
	log.Fatal(http.ListenAndServe(":8080", r))
}
