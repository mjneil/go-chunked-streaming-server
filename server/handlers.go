package server

import (
	"io"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
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

	addCors(w, cors)
	addHeaders(w, f.headers)
	w.Header().Set("Transfer-Encoding", "chunked")

	w.WriteHeader(http.StatusOK)
	rc := f.NewReadCloser(basePath,w)
	defer rc.Close()
	io.Copy(ChunkedResponseWriter{w}, rc)
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

	addCors(w, cors)
	addHeaders(w, f.headers)
	w.Header().Set("Transfer-Encoding", "chunked")

	w.WriteHeader(http.StatusOK)
}

// PostHandler Writes a file
func PostHandler(onlyRAM bool, cors *Cors, basePath string, w http.ResponseWriter, r *http.Request) {
	name := r.URL.String()

	maxAgeS := getMaxAgeOr(r.Header.Get("Cache-Control"), -1)
	headers := getHeadersFiltered(r.Header)

	f := NewFile(name, headers, maxAgeS)

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
	// Add Content-Type & Cache-Control automatically
	// Some features depends on those
	allowedHeaders := cors.GetAllowedHeaders()
	allowedHeaders = append(allowedHeaders, "Content-Type")
	allowedHeaders = append(allowedHeaders, "Cache-Control")

	w.Header().Set("Access-Control-Allow-Origin", strings.Join(cors.GetAllowedOrigins(), ", "))
	w.Header().Set("Access-Control-Allow-Headers", strings.Join(allowedHeaders, ", "))
	w.Header().Set("Access-Control-Allow-Methods", strings.Join(cors.GetAllowedMethods(), ", "))
	w.Header().Set("Access-Control-Expose-Headers", strings.Join(allowedHeaders, ", "))
}

func addHeaders(w http.ResponseWriter, headersSrc http.Header) {
	// Copy all headers
	for name, values := range headersSrc {
		// Loop over all values for the name.
		for _, value := range values {
			w.Header().Set(name, value)
		}
	}
}
func getMaxAgeOr(s string, def int64) int64 {
	ret := def
	r := regexp.MustCompile(`max-age=(?P<maxage>\d*)`)
	match := r.FindStringSubmatch(s)
	for i, name := range r.SubexpNames() {
		if i > 0 && i <= len(match) {
			if name == "maxage" {
				valInt, err := strconv.ParseInt(match[i], 10, 64)
				if err == nil {
					ret = valInt
					break
				}
			}
		}
	}
	return ret
}

func getHeadersFiltered(headers http.Header) http.Header {
	ret := headers.Clone()

	// Clean up
	ret.Del("User-Agent")

	return ret
}
