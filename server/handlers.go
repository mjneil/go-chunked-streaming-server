package server

import (
	"io"
	"log"
	"net/http"
)

// ChunkedResponseWriter Define a response writer
type ChunkedResponseWriter struct {
	w http.ResponseWriter
}

// Write Writes few bytes
func (rw ChunkedResponseWriter) Write(p []byte) (nn int, err error) {
	nn, err = rw.w.Write(p)
	rw.w.(http.Flusher).Flush()
	return
}

// GetHandler Sends file bytes
func GetHandler(basePath string, w http.ResponseWriter, r *http.Request) {
	FilesLock.RLock()
	f, ok := Files[r.URL.String()]
	FilesLock.RUnlock()

	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	log.Println("GET Content-Type " + f.ContentType)

	w.Header().Set("Content-Type", f.ContentType)
	w.Header().Set("Transfer-Encoding", "chunked")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(http.StatusOK)
	io.Copy(ChunkedResponseWriter{w}, f.NewReader(basePath, w))
}

// HeadHandler Sends if file exists
func HeadHandler(w http.ResponseWriter, r *http.Request) {
	FilesLock.RLock()
	f, ok := Files[r.URL.String()]
	FilesLock.RUnlock()

	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", f.ContentType)
	w.Header().Set("Transfer-Encoding", "chunked")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(http.StatusOK)
}

// PostHandler Writes a file
func PostHandler(basePath string, w http.ResponseWriter, r *http.Request) {
	name := r.URL.String()
	f := NewFile(name, r.Header.Get("Content-Type"))

	FilesLock.Lock()
	Files[name] = f
	FilesLock.Unlock()

	// Start writing to file without holding lock so that GET requests can read from it
	io.Copy(f, r.Body)
	r.Body.Close()
	f.Close()

	err := f.WriteToDisk(basePath)
	if err != nil {
		log.Fatalf("Error saving to disk: %v", err)
	}

	w.WriteHeader(http.StatusNoContent)
}

// PutHandler Writes a file
func PutHandler(basePath string, w http.ResponseWriter, r *http.Request) {
	PostHandler(basePath, w, r)
}

// DeleteHandler Deletes a file
func DeleteHandler(basePath string, w http.ResponseWriter, r *http.Request) {
	FilesLock.RLock()
	f, ok := Files[r.URL.String()]
	if ok {
		delete(Files, r.URL.String())
	}
	FilesLock.RUnlock()

	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	f.RemoveFromDisk(basePath)
	w.WriteHeader(http.StatusNoContent)
}

// OptionsHandler Returns CORS options
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
