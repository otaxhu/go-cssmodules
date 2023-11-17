package cssmodules

import (
	"strings"
	"testing"
)

var testCasesHTMLCSSModules = []struct {
	name              string
	payload           string
	cssModulesClasses map[string]string
	expectedHTML      string
	expectedError     string
}{
	{
		name:              "ValidHTMLCSSModules",
		cssModulesClasses: map[string]string{"test-1": "RAN_1", "test-2": "RAN_2"},
		expectedError:     "",

		expectedHTML: `<!DOCTYPE html>
<head>
	<title>Test Title</title>
	<link rel="stylesheet" href="/path/to/styles.css">
</head>
<body>
	<div class="RAN_1">
		<h2 class="RAN_2">Header 2 with css module = "test-2"</h2>
		<p class="RAN_1 RAN_2">Paragraph with css module = "test-1" and "test-2"</p>
		<a class="my-normal-class">Anchor with classes without css module</a>
		<span class="my-normal-class RAN_1 RAN_2">Span with css module = "test-1" and "test-2". Also with a class</span>
		<span class="my-normal-class RAN_1 RAN_2">Same as above but with different order</span>
	</div>
</body>`,

		payload: `<!DOCTYPE html>
<head>
	<title>Test Title</title>
	<link rel="stylesheet" href="/path/to/styles.css">
</head>
<body>
	<div css-module="test-1">
		<h2 css-module="test-2">Header 2 with css module = "test-2"</h2>
		<p css-module="test-1 test-2">Paragraph with css module = "test-1" and "test-2"</p>
		<a class="my-normal-class">Anchor with classes without css module</a>
		<span css-module="test-1 test-2" class="my-normal-class">Span with css module = "test-1" and "test-2". Also with a class</span>
		<span class="my-normal-class" css-module="test-1 test-2">Same as above but with different order</span>
	</div>
</body>`,
	},
	{
		name:              "ValidHTMLCSSModules_GoTemplatesSupport",
		cssModulesClasses: map[string]string{"test-1": "RAN_1", "test-2": "RAN_2"},
		expectedError:     "",

		expectedHTML: `{{template "layouts/layout-head"}}
{{template "components/Navbar" .testVariable}}
<div class="RAN_1">
	<p class="RAN_1 RAN_2">Some Paragraph Content</p>
</div>
{{template "layouts/layout-foot"}}`,

		payload: `{{template "layouts/layout-head"}}
{{template "components/Navbar" .testVariable}}
<div css-module="test-1">
	<p css-module="test-1 test-2">Some Paragraph Content</p>
</div>
{{template "layouts/layout-foot"}}`,
	},
}

func TestProcessHTMLWithCSSModules(t *testing.T) {
	for i := range testCasesHTMLCSSModules {
		tc := testCasesHTMLCSSModules[i]
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			resultingHTML, err := ProcessHTMLWithCSSModules(strings.NewReader(tc.payload), tc.cssModulesClasses)
			if err != nil {
				if err.Error() != tc.expectedError {
					t.Errorf("unexpected error value: expected %s got %s", tc.expectedError, err.Error())
				}
			} else {
				if tc.expectedError != "" {
					t.Errorf("unexpected error value: expected %s got <nil>", tc.expectedError)
				}
			}
			if string(resultingHTML) != tc.expectedHTML {
				t.Errorf("unexpected html value: expected %s got %s", tc.expectedHTML, resultingHTML)
			}
		})
	}
}

func TestHTMLCSSModulesParser_ParseTo(t *testing.T) {
	for i := range testCasesHTMLCSSModules {
		tc := testCasesHTMLCSSModules[i]
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			buf := getBuffer()
			if err := NewHTMLCSSModulesParser(strings.NewReader(tc.payload), tc.cssModulesClasses).ParseTo(buf); err != nil {
				if tc.expectedError != err.Error() {
					t.Errorf("unexpected error value: expected %q got %q", tc.expectedError, err.Error())
					return
				}
			} else {
				if tc.expectedError != "" {
					t.Errorf("unexpected error value: expected %q got <nil>", tc.expectedError)
					return
				}
			}
			if buf.String() != tc.expectedHTML {
				t.Errorf("unexpected html value: expected %s got %s", tc.expectedHTML, buf.String())
				return
			}
		})
	}
}
