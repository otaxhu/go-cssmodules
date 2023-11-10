package benchmarks

import (
	"strings"
	"testing"

	"github.com/otaxhu/go-cssmodules"
)

func BenchmarkProcessCSSModules(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _, err := cssmodules.ProcessCSSModules(strings.NewReader(`@media screen {
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

}`))
		if err != nil {
			b.Errorf("unexpected error: %q", err.Error())
		}
	}
}
