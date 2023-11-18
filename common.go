package cssmodules

import (
	"bytes"
	"hash/adler32"
	"io"
	"sync"
)

type writer interface {
	io.Writer
	io.ByteWriter
	io.StringWriter
}

var adlerHashFunction = adler32.New()

// Buffer pool
var bp = sync.Pool{
	New: func() any {
		return &bytes.Buffer{}
	},
}

func getBuffer() *bytes.Buffer {
	b := bp.Get().(*bytes.Buffer)
	b.Reset()
	return b
}

func releaseBuffer(b *bytes.Buffer) {
	if b != nil {
		b.Reset()
		bp.Put(b)
	}
}
