package server

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"sync"
)

var (
	Files      = map[string]*File{}
	FilesLock  = new(sync.RWMutex)
	contentDir = "./content"
)

type File struct {
	Name        string
	ContentType string
	lock        *sync.RWMutex
	buffer      []byte
	eof         bool
	onDisk      bool
}

func NewFile(name, contentType string) *File {
	return &File{
		Name:        name,
		ContentType: contentType,
		lock:        new(sync.RWMutex),
		buffer:      []byte{},
		eof:         false,
		onDisk:      false,
	}
}

type FileReader struct {
	offset int
	*File
}

func (f *File) NewReader() io.Reader {
	f.lock.RLock()
	defer f.lock.RUnlock()

	if f.onDisk {
		name := path.Join(contentDir, f.Name)
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
		File:   f,
	}
}

func (r *FileReader) Read(p []byte) (int, error) {
	r.File.lock.RLock()
	defer r.File.lock.RUnlock()
	if r.offset >= len(r.File.buffer) {
		if r.File.eof {
			return 0, io.EOF
		} else {
			return 0, nil
		}
	}
	n := copy(p, r.File.buffer[r.offset:])
	r.offset += n
	return n, nil
}

func (f *File) Write(p []byte) (int, error) {
	f.lock.Lock()
	defer f.lock.Unlock()
	f.buffer = append(f.buffer, p...)
	return len(p), nil
}

func (f *File) Close() {
	f.lock.Lock()
	defer f.lock.Unlock()
	f.eof = true
}

func (f *File) WriteToDisk() error {
	f.lock.Lock()
	defer f.lock.Unlock()
	name := path.Join(contentDir, f.Name)
	err := ioutil.WriteFile(name, f.buffer, 0644)
	if err != nil {
		return err
	}
	f.onDisk = true
	f.buffer = nil
	return nil
}

func (f *File) RemoveFromDisk() error {
	f.lock.Lock()
	defer f.lock.Unlock()

	name := path.Join(contentDir, f.Name)
	err := os.Remove(name)

	// even if we get an error, lets act as if the file is completely removed
	f.onDisk = false
	f.buffer = nil

	return err
}
