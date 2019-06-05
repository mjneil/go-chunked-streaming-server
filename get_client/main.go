package main

import (
	"bufio"
	"fmt"
	"io"
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

	req := &http.Request{
		Method: "GET",
		URL: &url.URL{
			Scheme: "http",
			Host:   "localhost:9094",
			Path:   "/post-client.txt",
		}
	}
	fmt.Printf("Doing request\n")
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	fmt.Println(resp.Body)
	reader := bufio.NewReader(resp.Body)
	for {
		line, err := reader.ReadBytes('\n')

		if err != nil {
			if err == io.EOF {
				fmt.Println("EOF")
				break
			}
			fmt.Printf("Error: %v\n", err)
			break
		}

		fmt.Println(string(line))
	}
}
