package main

import (
	"flag"
	"log"

	"github.com/mjneil/go-chunked-streaming-server/server"
)

var (
	certFilePath                 = flag.String("c", "", "Certificate file path (only for https)")
	keyFilePath                  = flag.String("k", "", "Key file path (only for https)")
	baseOutPath                  = flag.String("p", "./content", "Path used to store")
	port                         = flag.Int("i", 9094, "Port used for HTTP ingress/ egress")
	corsConfigFilePath           = flag.String("o", "", "JSON file path with the CORS headers definition")
	onlyRAM                      = flag.Bool("r", false, "Indicates DO NOT use disc as persistent/fallback storage (only RAM)")
	waitForDataToArrive          = flag.Bool("w", false, "Indicates to GET request to wait for some specific if data is NOT present yet")
	doCleanupBasedOnCacheHeaders = flag.Bool("d", false, "Indicates to remove files from the server based on original Cache-Control (max-age) header")
	urlTranslator                = flag.Bool("t", false, "Activates URL translation options that enables livestreaming features such as LATESTS, -30s")
)

func checkError(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}

func main() {
	flag.Parse()

	checkError(server.StartHTTPServer(*baseOutPath, *port, *certFilePath, *keyFilePath, *corsConfigFilePath, *onlyRAM, *doCleanupBasedOnCacheHeaders, *waitForDataToArrive, *urlTranslator))
}
