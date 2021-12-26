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
func GetHandler(cors *Cors, basePath string, w http.ResponseWriter, r *http.Request) {
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

	addCors(w, cors)
	w.WriteHeader(http.StatusOK)
	io.Copy(ChunkedResponseWriter{w}, f.NewReader(basePath, w))
}

// HeadHandler Sends if file exists
func HeadHandler(cors *Cors, w http.ResponseWriter, r *http.Request) {
	FilesLock.RLock()
	f, ok := Files[r.URL.String()]
	FilesLock.RUnlock()

	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", f.ContentType)
	w.Header().Set("Transfer-Encoding", "chunked")

	addCors(w, cors)
	w.WriteHeader(http.StatusOK)
}

// PostHandler Writes a file
func PostHandler(onlyRAM bool, cors *Cors, basePath string, w http.ResponseWriter, r *http.Request) {
	name := r.URL.String()
	f := NewFile(name, r.Header.Get("Content-Type"))

	FilesLock.Lock()
	Files[name] = f
	FilesLock.Unlock()

	// Start writing to file without holding lock so that GET requests can read from it
	io.Copy(f, r.Body)
	r.Body.Close()
	f.Close()

	if !onlyRAM {
		err := f.WriteToDisk(basePath)
		if err != nil {
			log.Fatalf("Error saving to disk: %v", err)
		}
	}
	addCors(w, cors)
	w.WriteHeader(http.StatusNoContent)
}

// PutHandler Writes a file
func PutHandler(onlyRAM bool, cors *Cors, basePath string, w http.ResponseWriter, r *http.Request) {
	PostHandler(onlyRAM, cors, basePath, w, r)
}

// DeleteHandler Deletes a file
func DeleteHandler(onlyRAM bool, cors *Cors, basePath string, w http.ResponseWriter, r *http.Request) {
	FilesLock.RLock()
	f, ok := Files[r.URL.String()]
	FilesLock.RUnlock()

	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	FilesLock.Lock()
	delete(Files, r.URL.String())
	FilesLock.Unlock()

	if !onlyRAM {
		f.RemoveFromDisk(basePath)
	}

	addCors(w, cors)
	w.WriteHeader(http.StatusNoContent)
}

// OptionsHandler Returns CORS options
func OptionsHandler(cors *Cors, w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Transfer-Encoding", "chunked")

	addCors(w, cors)
	w.WriteHeader(http.StatusNoContent)
}

func addCors(w http.ResponseWriter, cors *Cors) {
	w.Header().Set("Access-Control-Allow-Origin", cors.GetAllowedOriginsStr())
	w.Header().Set("Access-Control-Allow-Headers", cors.GetAllowedHeadersStr())
	w.Header().Set("Access-Control-Allow-Methods", cors.GetAllowedMethodsStr())
}
