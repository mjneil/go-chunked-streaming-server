package server

import (
	"io"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
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
func GetHandler(urlTranslator *UrlTranslator, waitingRequests *WaitingRequests, cors *Cors, basePath string, w http.ResponseWriter, r *http.Request) {
	name := r.URL.String()

	FilesLock.RLock()

	if urlTranslator != nil {
		name = urlTranslator.GetTranslated(name)
	}
	f, ok := Files[name]

	FilesLock.RUnlock()

	if !ok {
		isFound := false
		waited := 0 * time.Millisecond
		if waitingRequests != nil {
			// Wait and return
			isFound, waited = waitingRequests.AddWaitingRequest(name, getHeadersFiltered(r.Header))
			w.Header().Set("Waited-For-Data-Ms", strconv.FormatInt(int64(waited/time.Millisecond), 10))
			if isFound {
				// Refresh file
				FilesLock.RLock()
				fnew, ok := Files[name]
				FilesLock.RUnlock()
				if !ok {
					// This should be very rare, file arrived but it is not in Files. It can happen if it expired just between arrived and this line
					isFound = false
				} else {
					f = fnew
				}
			}
		}
		if !isFound {
			addCors(w, cors)
			w.WriteHeader(http.StatusNotFound)
			return
		}
	}
	addCors(w, cors)
	addHeaders(w, f.headers)

	// Add chunked only if the file is not yet complete
	if !f.eof {
		w.Header().Set("Transfer-Encoding", "chunked")
	}

	w.WriteHeader(http.StatusOK)
	io.Copy(ChunkedResponseWriter{w}, f.NewReadCloser(basePath, w))
}

// HeadHandler Sends if file exists
func HeadHandler(cors *Cors, w http.ResponseWriter, r *http.Request) {
	FilesLock.RLock()
	f, ok := Files[r.URL.String()]
	FilesLock.RUnlock()

	addCors(w, cors)
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	addHeaders(w, f.headers)
	w.Header().Set("Transfer-Encoding", "chunked")

	w.WriteHeader(http.StatusOK)
}

// PostHandler Writes a file
func PostHandler(urlTranslator *UrlTranslator, waitingRequests *WaitingRequests, onlyRAM bool, cors *Cors, basePath string, w http.ResponseWriter, r *http.Request) {
	name := r.URL.String()

	maxAgeS := getMaxAgeOr(r.Header.Get("Cache-Control"), -1)
	chunkCreatedAt := getGenerationTimeOr(r.Header.Get("Joc-First-Frame-Clk"), time.Now())
	headers := getHeadersFiltered(r.Header)

	f := NewFile(name, headers, maxAgeS)

	FilesLock.Lock()

	if urlTranslator != nil {
		urlTranslator.AddNewEntry(name, chunkCreatedAt)
	}
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

	// Awake GET requests waiting (if there are any)
	if waitingRequests != nil {
		waitingRequests.ReceivedDataFor(name)
	}
}

// PutHandler Writes a file
func PutHandler(urlTranslator *UrlTranslator, waitingRequests *WaitingRequests, onlyRAM bool, cors *Cors, basePath string, w http.ResponseWriter, r *http.Request) {
	PostHandler(urlTranslator, waitingRequests, onlyRAM, cors, basePath, w, r)
}

// DeleteHandler Deletes a file
func DeleteHandler(urlTranslator *UrlTranslator, onlyRAM bool, cors *Cors, basePath string, w http.ResponseWriter, r *http.Request) {
	name := r.URL.String()

	FilesLock.RLock()

	if urlTranslator != nil {
		urlTranslator.RemoveEntry(name)
	}
	f, ok := Files[name]

	FilesLock.RUnlock()

	addCors(w, cors)
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

func getGenerationTimeOr(valStr string, def time.Time) time.Time {
	ret := def
	timeMs, err := strconv.ParseInt(valStr, 10, 64)
	if err == nil {
		timeSecs := timeMs / 1000
		timeNs := (timeMs - (timeSecs * 1000)) * 1000 * 1000
		ret = time.Unix(timeSecs, timeNs)
	}
	return ret
}

func getHeadersFiltered(headers http.Header) http.Header {
	ret := headers.Clone()

	// Clean up
	ret.Del("User-Agent")

	return ret
}
