package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"time"
)

const (
	segmentLen = 16
	chunkLen   = 4
	max        = 5
)

func main() {
	out := os.Stdout

	for i := 0; i < max; i++ {
		currentSegment := newSegment(i)
		for offset := 0; offset < len(currentSegment); offset += chunkLen {
			b := currentSegment[offset : offset+chunkLen]
			buf := bytes.NewBuffer(b)
			io.Copy(out, buf)
			time.Sleep(1 * time.Second)
		}
	}
}

func newSegment(index int) []byte {
	str := fmt.Sprintf("$%015d", index)
	buf := make([]byte, 16)
	copy(buf, str)
	return buf
}
