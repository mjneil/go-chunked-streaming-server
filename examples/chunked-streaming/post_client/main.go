package main

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
)

const (
	fileBase = "./content/examples/chunked-streaming/post_client"
	httpBase = "/segments"
	chunkLen = 4
)

func setupContentDir() {
	err := os.RemoveAll(fileBase)
	if err != nil {
		panic(err)
	}
	err = os.MkdirAll(fileBase, 0755)
	if err != nil {
		panic(err)
	}
}

// Allows use of io.TeeReader for http.Request.Body
type TeeReadCloser struct {
	r io.Reader
	c io.Closer
}

func (t *TeeReadCloser) Read(p []byte) (n int, err error) {
	return t.r.Read(p)
}

func (t *TeeReadCloser) Close() error {
	return t.c.Close()
}

type Segment struct {
	file  *os.File
	index int
	w     io.WriteCloser
	r     io.ReadCloser
}

func NewSegment(index int) (*Segment, error) {
	file, err := newSegmentFile(index)
	if err != nil {
		return nil, err
	}

	pipeR, pipeW := io.Pipe()

	// as the http client reads from the pipe to upload, tee will also write to file
	tee := io.TeeReader(pipeR, file)
	teerc := &TeeReadCloser{tee, pipeR}

	return &Segment{
		file:  file,
		index: index,
		w:     pipeW,
		r:     teerc,
	}, nil
}

func (s *Segment) Write(p []byte) (n int, err error) {
	fmt.Print(string(p))
	return s.w.Write(p)
}

func (s *Segment) Close() error {
	s.w.Close()
	fmt.Printf("\nEnd Upload: %s\n", segmentName(httpBase, s.index))
	return s.file.Close()
}

func (s *Segment) StartUpload(client *http.Client, errC chan error) {
	fmt.Printf("Starting Upload: %s\n", segmentName(httpBase, s.index))

	go func() {
		err := uploadSegment(client, s.index, s.r)
		if err != nil {
			errC <- err
		}
	}()
}

func main() {
	setupContentDir()

	tr := http.DefaultTransport

	client := &http.Client{
		Transport: tr,
		Timeout:   0,
	}

	var (
		inputStream    = bufio.NewReader(os.Stdin)
		chunk          = make([]byte, chunkLen)
		err            error
		index          int
		n              int
		currentSegment *Segment
		errC           = make(chan error)
	)

	endSegment := func() error {
		if currentSegment == nil {
			return nil
		}

		e := currentSegment.Close()
		currentSegment = nil
		return e
	}

	nextSegment := func() error {
		currentSegment, err = NewSegment(index)
		if err != nil {
			return err
		}
		currentSegment.StartUpload(client, errC)
		index++
		return nil
	}

	// cleanup potential segment opened
	defer func() {
		if err != nil {
			printErr(err)
		}
		err = endSegment()
		if err != nil {
			panic(err)
		}
	}()

	// errs in loop are printed in the defer
	for {
		select {
		case err = <-errC:
			return
		default:
		}

		// Read from STDIN one chunk at a time
		n, err = io.ReadFull(inputStream, chunk)
		if err != nil {
			if err == io.EOF {
				err = nil
			}
			if n > 0 {
				fmt.Printf("io.ReadFull bytes on error: %s\n", string(chunk[:n]))
			}
			return
		}

		// check for new segment delim
		if chunk[0] == '$' {
			err = endSegment()
			if err != nil {
				return
			}

			err = nextSegment()
			if err != nil {
				return
			}
		}

		_, err = currentSegment.Write(chunk)
		if err != nil {
			return
		}
	}
}

func segmentName(base string, index int) string {
	return path.Join(base, fmt.Sprintf("segment-%d", index))
}

func newSegmentFile(index int) (*os.File, error) {
	return os.OpenFile(segmentName(fileBase, index), os.O_APPEND|os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0644)
}

func uploadSegment(client *http.Client, index int, body io.ReadCloser) error {
	req := &http.Request{
		Method: "POST",
		URL: &url.URL{
			Scheme: "http",
			Host:   "localhost:9094",
			Path:   segmentName(httpBase, index),
		},
		ProtoMajor:    1,
		ProtoMinor:    1,
		ContentLength: -1,
		Body:          body,
	}
	_, err := client.Do(req)
	return err
}

func printErr(err error) {
	fmt.Printf("Error: %v\n", err)
}
