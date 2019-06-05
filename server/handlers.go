package server

import (
	"io"
	"log"
	"net/http"
)

func GetHandler(w http.ResponseWriter, r *http.Request) {
	FilesLock.RLock()
	defer FilesLock.RUnlock()

	f, ok := Files[r.URL.String()]
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", f.ContentType)
	w.Header().Set("Transfer-Encoding", "chunked")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(http.StatusOK)
	io.Copy(w, f.NewReader())
}

func HeadHandler(w http.ResponseWriter, r *http.Request) {
	FilesLock.RLock()
	defer FilesLock.RUnlock()

	f, ok := Files[r.URL.String()]
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", f.ContentType)
	w.Header().Set("Transfer-Encoding", "chunked")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(http.StatusOK)
}

func PostHandler(w http.ResponseWriter, r *http.Request) {
	name := r.URL.String()
	f := NewFile(name, r.Header.Get("Content-Type"))

	FilesLock.Lock()
	Files[name] = f
	FilesLock.Unlock()

	// Start writing to file without holding lock so that GET requests can read from it
	io.Copy(f, r.Body)
	r.Body.Close()
	f.Close()

	err := f.WriteToDisk()
	if err != nil {
		log.Fatalf("Error saving to disk: %v", err)
	}

	w.WriteHeader(http.StatusNoContent)
}

func PutHandler(w http.ResponseWriter, r *http.Request) {
	PostHandler(w, r)
}

func DeleteHandler(w http.ResponseWriter, r *http.Request) {
	FilesLock.RLock()
	f, ok := Files[r.URL.String()]
	FilesLock.RUnlock()

	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	FilesLock.Lock()
	defer FilesLock.Unlock()
	f.RemoveFromDisk()
	delete(Files, r.URL.String())

	w.WriteHeader(http.StatusNoContent)
}

func OptionsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Allow", http.MethodGet)
	w.Header().Add("Allow", http.MethodHead)
	w.Header().Add("Allow", http.MethodPost)
	w.Header().Add("Allow", http.MethodPut)
	w.Header().Add("Allow", http.MethodDelete)
	w.Header().Add("Allow", http.MethodOptions)
	w.Header().Set("Transfer-Encoding", "chunked")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(http.StatusNoContent)
}
