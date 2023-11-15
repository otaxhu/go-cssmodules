package cssmodules

import (
	"bytes"
	"encoding/base32"
	"io"
	"sync"
)

type writer interface {
	io.Writer
	io.ByteWriter
	io.StringWriter
}

// It's the base32 encoding but without = signs on the output
var base32StdEncodingNoPad = base32.StdEncoding.WithPadding(-1)

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
