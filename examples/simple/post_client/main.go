package main

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
)

func main() {
	tr := http.DefaultTransport

	client := &http.Client{
		Transport: tr,
		Timeout:   0,
	}

	r := os.Stdin
	req := &http.Request{
		Method: "POST",
		URL: &url.URL{
			Scheme: "http",
			Host:   "localhost:9094",
			Path:   "/post-client.txt",
		},
		ProtoMajor:    1,
		ProtoMinor:    1,
		ContentLength: -1,
		Body:          r,
	}
	fmt.Printf("Doing request\n")
	_, err := client.Do(req)
	fmt.Printf("Done request. Err: %v\n", err)
}
