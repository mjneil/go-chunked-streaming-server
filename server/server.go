package server

import (
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
)

// StartHTTPServer Starts the webserver
func StartHTTPServer(basePath string, port int, verboseLogging bool, certFilePath string, keyFilePath string) error {
	r := mux.NewRouter()

	r.PathPrefix("/").Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer w.(http.Flusher).Flush()
		if verboseLogging {
			log.Printf("%s %s {%s}", r.Method, r.URL.String(), getHeadersList(r))
		} else {
			log.Printf("%s %s", r.Method, r.URL.String())
		}
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

func getHeadersList(r *http.Request) string {
	ret := ""
	for name, values := range r.Header {
		if ret == "" {
			ret = name + ":"
		} else {
			ret = ret + "; " + name + ":"
		}

		// Loop over all values for the name.
		var valuesStr []string
		for _, value := range values {
			valuesStr = append(valuesStr, value)
		}
		ret = ret + strings.Join(valuesStr, ",")
	}

	return ret
}
