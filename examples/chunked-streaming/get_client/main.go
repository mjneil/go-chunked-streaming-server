package main

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"time"
)

const (
	httpBase = "/segments"
	max      = 5
)

func main() {
	tr := http.DefaultTransport

	client := &http.Client{
		Transport: tr,
		Timeout:   0,
	}

	var (
		index         int
		err           error
		currentBody   io.Closer
		currentReader *bufio.Reader
	)

	endSegment := func() error {
		if currentBody == nil {
			return nil
		}
		fmt.Printf("\nSegment finished: %s\n", segmentName(httpBase, index-1))
		e := currentBody.Close()
		currentReader = nil
		currentBody = nil
		return e
	}

	nextSegment := func() error {
		req := &http.Request{
			Method: "GET",
			URL: &url.URL{
				Scheme: "http",
				Host:   "localhost:9094",
				Path:   segmentName(httpBase, index),
			},
		}

		fmt.Printf("Requesting segment: %s\n", segmentName(httpBase, index))

		resp, err := client.Do(req)
		if err != nil {
			return err
		}

		if resp.StatusCode == http.StatusNotFound {
			currentBody = nil
			currentReader = nil
			return nil
		}

		index++

		currentBody = resp.Body
		currentReader = bufio.NewReader(resp.Body)
		return nil
	}

	// cleanup potential request body
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
		if currentReader == nil {
			// sleep until the segment is available
			time.Sleep(1 * time.Second)
			err = nextSegment()
			if err != nil {
				return
			}
			continue
		}

		b, err := currentReader.ReadByte()
		fmt.Print(string(b))

		if err == io.EOF {
			err = endSegment()
			if err != nil {
				return
			}

			if index == max {
				// done
				break
			}
			err = nextSegment()
		}

		if err != nil {
			return
		}
	}
}

func segmentName(base string, index int) string {
	return path.Join(base, fmt.Sprintf("segment-%d", index))
}

func printErr(err error) {
	fmt.Printf("Error: %v\n", err)
}
