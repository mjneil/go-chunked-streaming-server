package main

import (
	"flag"
	"log"

	"github.com/mjneil/go-chunked-streaming-server/server"
)

var (
	baseOutPath = flag.String("p", "./content", "Path used to store ")
	port        = flag.Int("i", 9094, "Port used for HTTP ingress/ egress")
)

func checkError(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}

func main() {
	flag.Parse()

	checkError(server.StartHTTPServer(*baseOutPath, *port))
}
