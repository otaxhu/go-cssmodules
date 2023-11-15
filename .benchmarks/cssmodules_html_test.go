package benchmarks

import (
	"bytes"
	"testing"

	"github.com/otaxhu/go-cssmodules"
)

func Benchmark_ProcessHTMLWithCSSModules(b *testing.B) {
	// Example with go templates
	s := `{{ template "test-template" .variable }}
<nav>
	<ul>
		<li>
			<a href="/home" css-module="class-1"><img src="/logo.png">Test Logo</a>
		</li>
		<li>
			<a href="/test" css-module="class-2">Link to Test</a>
		</li>
	</ul>
</nav>`
	classes := map[string]string{"class-1": "RAN_1", "class-2": "RAN_2"}
	payloadBuf := &bytes.Buffer{}
	for i := 0; i < b.N; i++ {
		payloadBuf.WriteString(s)
		_, err := cssmodules.ProcessHTMLWithCSSModules(payloadBuf, classes)
		if err != nil {
			b.Error(err)
		}
		payloadBuf.Reset()
	}
}

func Benchmark_HTMLCSSModulesParser_ParseTo(b *testing.B) {
	// Example with go templates
	s := `{{ template "test-template" .variable }}
<nav>
	<ul>
		<li>
			<a href="/home" css-module="class-1"><img src="/logo.png">Test Logo</a>
		</li>
		<li>
			<a href="/test" css-module="class-2">Link to Test</a>
		</li>
	</ul>
</nav>`
	classes := map[string]string{"class-1": "RAN_1", "class-2": "RAN_2"}
	buf := &bytes.Buffer{}
	payloadBuf := &bytes.Buffer{}
	for i := 0; i < b.N; i++ {
		payloadBuf.WriteString(s)
		if err := cssmodules.NewHTMLCSSModulesParser(payloadBuf, classes).ParseTo(buf); err != nil {
			b.Error(err)
		}
		payloadBuf.Reset()
		buf.Reset()
	}
}
