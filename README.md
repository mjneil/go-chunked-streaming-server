# go-chunked-streaming-server

This is simple webserver written in [GoLang](https://golang.org/) that allows you to chunk transfer from the ingress and egress sides. In other words:
- If it receives a GET request for object A, and object A is still being ingested, we send all we have of object A and we keep that GET request open sending on real time (and in memory) all the chunks that we received for object A, until that object is closed.
- If we receives a GET request for object B and this object is already written (and closed), we send the object B as any other static web server.

Ideal for low latency media processing pipelines.

# Usage
## Installation
1. Just download GO in your computer. See [GoLang](https://golang.org/)
2. Create a Go directory to be used as a workspace for all go code, i.e.
```
mkdir ~/MYDIR/go
```
3. Add `GOPATH` to your `~/.bash_profile` or equivalent for your shell
```
export GOPATH="$HOME/MYDIR/go"
```
4. Add `GOPATH/bin` to your path in `~/.bash_profile` or equivalent for your shell
```
export PATH="$PATH:$GOPATH/bin
```
5. Restart your terminal or source your profile
6. Clone this repo:
```
go get github.com/mjneil/go-chunked-streaming-server
```
7. Go the the source code dir `
```
cd $HOME/MYDIR/go/src/github.com/mjneil/go-chunked-streaming-server
```
8. Compile `main.go` doing:
```
make
```

## Testing
You can execute `./bin/./go-chunked-streaming-server -h` to see all the possible command arguments.
```
Usage of ./bin/./go-chunked-streaming-server:
  -c string
        Certificate file path (only for https)
  -i int
        Port used for HTTP ingress/ egress (default 9094)
  -k string
        Key file path (only for https)
  -o string
        JSON file path with the CORS headers definition
  -p string
        Path used to store (default "./content")
```

## Example simple HTTP
- Start the server
```
./bin/./go-chunked-streaming-server
```
- Upload a file
```
curl http://localhost:9094 --upload-file file.test.txt
```
- Consume the file and saved it to disc
```
curl http://localhost:9094/file.test.txt -o file.test.downloaded.txt
```

## Example chunked HTTP
We could build a low latency HLS pipeline in conjunction with [go-ts-segmenter](https://github.com/jordicenzano/go-ts-segmenter) by doing:
1. Start webserver
```
./bin/./go-chunked-streaming-server
```
2. Generate LHLS data
```
ffmpeg -f lavfi -re -i smptebars=size=320x200:rate=30 -f lavfi -i sine=frequency=1000:sample_rate=48000 -pix_fmt yuv420p -c:v libx264 -b:v 180k -g 60 -keyint_min 60 -profile:v baseline -preset veryfast -c:a aac -b:a 96k -f mpegts - | manifest-generator -l 3 -d 2
```
3. Play that data with any HLS player (recommended any player that implements low latency mode)
```
ffplay http://localhost:9094/results/chunklist.m3u8
```
Or use Safari with this URL `http://localhost:9094/results/chunklist.m3u8`
