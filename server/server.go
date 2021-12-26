package server

import (
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

var (
	cleanUpChannel = make(chan bool)
)

// StartHTTPServer Starts the webserver
func StartHTTPServer(basePath string, port int, certFilePath string, keyFilePath string, corsConfigFilePath string, onlyRAM bool, doCleanupBasedOnCacheHeaders bool) error {
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
			PostHandler(onlyRAM, cors, basePath, w, r)
		case http.MethodPut:
			PutHandler(onlyRAM, cors, basePath, w, r)
		case http.MethodDelete:
			DeleteHandler(onlyRAM, cors, basePath, w, r)
		case http.MethodOptions:
			OptionsHandler(cors, w, r)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})).Methods(http.MethodGet, http.MethodHead, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodOptions)

	if doCleanupBasedOnCacheHeaders {
		startCleanUp(1000)
	}

	if (certFilePath != "") && (keyFilePath != "") {
		// Try HTTPS
		log.Printf("HTTPS server running on port %d", port)
		err = http.ListenAndServeTLS(":"+strconv.Itoa(port), certFilePath, keyFilePath, r)
	} else {
		// Try HTTP
		log.Printf("HTTP server running on port %d", port)
		err = http.ListenAndServe(":"+strconv.Itoa(port), r)
	}

	if doCleanupBasedOnCacheHeaders {
		stopCleanUp()
	}

	return err
}

func startCleanUp(periodMs int64) {
	go runCleanupEvery(periodMs, cleanUpChannel)

	log.Printf("HTTP Started clean up thread")
}

func stopCleanUp() {
	// Send finish signal
	cleanUpChannel <- true

	// Wait to finish
	<-cleanUpChannel

	log.Printf("HTTP Stopped clean up thread")
}

func runCleanupEvery(periodMs int64, cleanUpChannelBidi chan bool) {
	timeCh := time.NewTicker(time.Millisecond * time.Duration(periodMs))
	exit := false

	for !exit {
		select {
		// Wait for the next tick
		case tm := <-timeCh.C:
			cacheCleanUp(tm)

		case <-cleanUpChannelBidi:
			exit = true
		}
	}
	// Indicates finished
	cleanUpChannelBidi <- true

	log.Printf("HTTP Exited clean up thread")
}

func cacheCleanUp(now time.Time) {
	//now := time.Now()
	toDelFilesKeys := []string{}

	// TODO: This is a brute force approach, optimization recommended

	FilesLock.Lock()
	defer FilesLock.Unlock()

	// Check for expired files
	for keyToDel, file := range Files {
		if file.maxAgeS >= 0 && file.eof {
			if file.receivedAt.Add(time.Second * time.Duration(file.maxAgeS)).Before(now) {
				toDelFilesKeys = append(toDelFilesKeys, keyToDel)
			}
		}
	}
	// Delete expired files
	for _, keyToDel := range toDelFilesKeys {
		delete(Files, keyToDel)
		log.Printf("CLEANUP expired, deleted: %s", keyToDel)
	}
}
