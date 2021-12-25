package server

import (
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

// StartHTTPServer Starts the webserver
func StartHTTPServer(basePath string, port int, certFilePath string, keyFilePath string, corsConfigFilePath string) error {
	var err error

	cors := NewCors()
	if corsConfigFilePath != "" {
		// Loads CORS config
		err = cors.LoadFromDisc(corsConfigFilePath)
		if err != nil {
			return err
		}
	} else {
		log.Printf("CORS default policy applied")
	}
	log.Printf("CORS: %s", cors.String())

	r := mux.NewRouter()

	r.PathPrefix("/").Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer w.(http.Flusher).Flush()
		log.Printf("%s %s", r.Method, r.URL.String())
		switch r.Method {
		case http.MethodGet:
			GetHandler(cors, basePath, w, r)
		case http.MethodHead:
			HeadHandler(cors, w, r)
		case http.MethodPost:
			PostHandler(cors, basePath, w, r)
		case http.MethodPut:
			PutHandler(cors, basePath, w, r)
		case http.MethodDelete:
			DeleteHandler(cors, basePath, w, r)
		case http.MethodOptions:
			OptionsHandler(cors, w, r)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})).Methods(http.MethodGet, http.MethodHead, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodOptions)

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
