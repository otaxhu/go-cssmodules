package benchmarks

import (
	"bytes"
	"testing"

	"github.com/otaxhu/go-cssmodules"
)

func Benchmark_ProcessCSSModules(b *testing.B) {
	s := `@media screen {
	.mi-clase {
		color: red;
		font-size: large;
	}

	.otra-clase:hover {
		color: green;
		font-size: medium
	}
}

.otra-clase {
	color: red;
	font-size: medium;
}

#mi-id {

}`
	buffer := &bytes.Buffer{}
	for i := 0; i < b.N; i++ {
		buffer.WriteString(s)
		if _, _, err := cssmodules.ProcessCSSModules(buffer); err != nil {
			b.Error(err)
		}
		buffer.Reset()
	}
}

func Benchmark_CSSModulesParser_ParseTo(b *testing.B) {
	s := `@media screen {
	.mi-clase {
		color: red;
		font-size: large;
	}

	.otra-clase:hover {
		color: green;
		font-size: medium
	}
}

.otra-clase {
	color: red;
	font-size: medium;
}

#mi-id {

}`
	payloadBuf := &bytes.Buffer{}
	buf := &bytes.Buffer{}
	for i := 0; i < b.N; i++ {
		payloadBuf.WriteString(s)
		if _, err := cssmodules.NewCSSModulesParser(payloadBuf).ParseTo(buf); err != nil {
			b.Error(err)
		}
		payloadBuf.Reset()
		buf.Reset()
	}
}
