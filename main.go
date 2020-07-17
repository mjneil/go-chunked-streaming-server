package main

import (
	"flag"
	"log"

	"github.com/mjneil/go-chunked-streaming-server/server"
)

var (
	verbose      = flag.Bool("verbose", false, "enable to get verbose logging")
	certFilePath = flag.String("c", "", "Certificate file path (only for https)")
	keyFilePath  = flag.String("k", "", "Key file path (only for https)")
	baseOutPath  = flag.String("p", "./content", "Path used to store")
	port         = flag.Int("i", 9094, "Port used for HTTP ingress/ egress")
)

func checkError(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}

func main() {
	flag.Parse()

	checkError(server.StartHTTPServer(*baseOutPath, *port, *verbose, *certFilePath, *keyFilePath))
}
