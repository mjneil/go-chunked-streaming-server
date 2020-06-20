package server

import (
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

// StartHTTPServer Starts the webserver
func StartHTTPServer(basePath string, port int, certFilePath string, keyFilePath string) error {
	r := mux.NewRouter()

	r.PathPrefix("/").Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer w.(http.Flusher).Flush()
		log.Printf("%s %s", r.Method, r.URL.String())
		switch r.Method {
		case http.MethodGet:
			GetHandler(basePath, w, r)
		case http.MethodHead:
			HeadHandler(w, r)
		case http.MethodPost:
			PostHandler(basePath, w, r)
		case http.MethodPut:
			PutHandler(basePath, w, r)
		case http.MethodDelete:
			DeleteHandler(basePath, w, r)
		case http.MethodOptions:
			OptionsHandler(w, r)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})).Methods(http.MethodGet, http.MethodHead, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodOptions)

	var err error
	if (certFilePath != "") && (keyFilePath != "") {
		// Try HTTPS
		log.Printf("HTTPS server running on port %d", port)
		err = http.ListenAndServeTLS(":"+strconv.Itoa(port), certFilePath, keyFilePath, r)
	} else {
		// Try HTTP
		log.Printf("HTTP server running on port %d", port)
		err = http.ListenAndServe(":"+strconv.Itoa(port), r)
	}

	return err
}
