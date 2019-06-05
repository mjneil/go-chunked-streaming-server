package main

import (
	"log"

	"github.com/mjneil/go-chunked-streaming-server/server"
)

func checkError(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}

func main() {
	checkError(server.StartHttpServer())
}
