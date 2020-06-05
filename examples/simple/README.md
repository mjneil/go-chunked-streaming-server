# Simple Example

This simple example demonstrates chunked transfer of a single request.
It uses 2 clients, a POST and GET client. The POST client opens a POST request to the chunked-streaming-server using STDIN as the request body. When a newline is read from STDIN, the POST client writes those bytes to the request. The GET client opens a GET request to the chunked-streaming-server for the same file as the POST request, logging lines to STDOUT as they are recieved.


In one terminal session, start the chunked-streaming-server
```
~/go-chunked-streaming-server $ go run main.go
```

Open a second terminal session and start the POST client.
```
~/go-chunked-streaming-server $ go run examples/simple/post_client/main.go
```

Open a third terminal session and start the GET client.
```
~/go-chunked-streaming-server $ go run examples/simple/get_client/main.go
```

In the second terminal session running the POST client, start typing whatever text you want. Each newline should be printed on the third terminal session runnining the GET client.
