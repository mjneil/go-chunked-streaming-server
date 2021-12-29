package server

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sync"
	"time"
)

var (
	// Files Array of on the fly files
	Files = map[string]*File{}

	// FilesLock Lock used to write / read files
	FilesLock = new(sync.RWMutex)
)

// File Definition of file
type File struct {
	Name       string
	headers    http.Header
	lock       *sync.RWMutex
	buffer     []byte
	eof        bool
	onDisk     bool
	receivedAt time.Time
	maxAgeS    int64
}

// NewFile Creates a new file
func NewFile(name string, headers http.Header, maxAgeS int64) *File {
	f := File{
		Name:       name,
		headers:    headers,
		lock:       new(sync.RWMutex),
		buffer:     []byte{},
		eof:        false,
		onDisk:     false,
		receivedAt: time.Now(),
		maxAgeS:    maxAgeS,
	}

	contentType := f.GetContentType()

	log.Println("NEW File Content-Type " + contentType)

	return &f
}

func (f *File) GetContentType() string {
	return f.headers.Get("Content-Type")
}

// FileReader Defines a reader
type FileReader struct {
	offset int
	w      http.ResponseWriter
	*File
}

// NewReader Crates a new filereader from a file
func (f *File) NewReader(baseDir string, w http.ResponseWriter) io.Reader {
	f.lock.RLock()
	defer f.lock.RUnlock()

	if f.onDisk {
		name := path.Join(baseDir, f.Name)
		file, err := os.Open(name)
		if err != nil {
			panic(err)
		}
		fmt.Println("Skipping file reading and reading from disk")
		return file
	}

	fmt.Println("Reading from memory")
	return &FileReader{
		offset: 0,
		w:      w,
		File:   f,
	}
}

// Read Reads bytes from filereader
func (r *FileReader) Read(p []byte) (int, error) {
	r.File.lock.RLock()
	defer r.File.lock.RUnlock()
	if r.offset >= len(r.File.buffer) {
		if r.File.eof {
			return 0, io.EOF
		}

		return 0, nil
	}
	n := copy(p, r.File.buffer[r.offset:])
	r.offset += n
	// r.w.(http.Flusher).Flush()
	return n, nil
}

// Write Write bytes to a file
func (f *File) Write(p []byte) (int, error) {
	f.lock.Lock()
	defer f.lock.Unlock()
	f.buffer = append(f.buffer, p...)
	return len(p), nil
}

// Close Closes a file
func (f *File) Close() {
	f.lock.Lock()
	defer f.lock.Unlock()
	f.eof = true
}

// WriteToDisk Writes a file to disc
func (f *File) WriteToDisk(baseDir string) error {
	f.lock.Lock()
	defer f.lock.Unlock()
	name := path.Join(baseDir, f.Name)

	if _, err := os.Stat(filepath.Dir(name)); os.IsNotExist(err) {
		err := os.MkdirAll(filepath.Dir(name), 0755)
		if err != nil {
			return err
		}
	}

	err := ioutil.WriteFile(name, f.buffer, 0644)
	if err != nil {
		return err
	}
	f.onDisk = true
	f.buffer = nil
	return nil
}

// RemoveFromDisk Removes file from disc
func (f *File) RemoveFromDisk(baseDir string) error {
	f.lock.Lock()
	defer f.lock.Unlock()

	name := path.Join(baseDir, f.Name)
	err := os.Remove(name)

	// even if we get an error, lets act as if the file is completely removed
	f.onDisk = false
	f.buffer = nil

	return err
}
