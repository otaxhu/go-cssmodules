package cssmodules

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

type matchableCSS struct {
	canMatch bool
	value    []byte
}

func newMatchableCSS(canMatch bool, value []byte) matchableCSS {
	return matchableCSS{
		canMatch: canMatch,
		value:    value,
	}
}

func TestProcessCSSModules(t *testing.T) {
	testCases := []struct {
		name                  string
		payload               io.Reader
		expectedCSSModules    matchableCSS
		expectedScopedClasses []string
		expectedError         string
	}{
		{
			name:                  "ValidCSSModules",
			expectedCSSModules:    newMatchableCSS(false, nil),
			expectedScopedClasses: []string{"test-class"},
			expectedError:         "",

			payload: strings.NewReader(`.test-class {
    color: red;
    font-size: large;
}`),
		},
		{
			name: "ValidCSSModules_GlobalKeyword",
			expectedCSSModules: newMatchableCSS(true,
				[]byte(`.test-class { color: red; font-size: large; }`),
			),
			expectedScopedClasses: nil,
			expectedError:         "",

			payload: strings.NewReader(`:global {.test-class { color: red; font-size: large; }}`),
		},
		{
			name:                  "ValidCSSModules_MediaQueryScoping",
			expectedCSSModules:    newMatchableCSS(false, nil),
			expectedScopedClasses: []string{"test-class"},
			expectedError:         "",

			payload: strings.NewReader(`@media screen and (min-width: 768px) and (max-width: 1024px) {
	.test-class {
		color: green;
		font-size: large;
	}
}`),
		},
		{
			name: "ValidCSSModules_Comments",
			expectedCSSModules: newMatchableCSS(true,
				[]byte(`/* Test Comments */ /* Not Closing Comment`),
			),
			expectedScopedClasses: nil,
			expectedError:         "",

			payload: strings.NewReader(`/* Test Comments */ /* Not Closing Comment`),
		},
		{
			name:                  "ValidCSSModules_ID#SymbolWillNotBeScoped_AnywaysWillBeWritten",
			expectedCSSModules:    newMatchableCSS(true, []byte(`#test-class {color: red; font-size: medium}`)),
			expectedScopedClasses: nil,
			expectedError:         "",

			payload: strings.NewReader(`#test-class {color: red; font-size: medium}`),
		},
		{
			name: "ValidCSSModules_Another@declarationsSupport",
			expectedCSSModules: newMatchableCSS(true, []byte(`@import url("path/to/styles.css");
@keyframes myAnimation {
	from {
		background-color: red;
	}
	to {
		background-color: blue;
	}
}
@keyframes anotherAnimation {
	0% {
		background-color: green;
	}
	10% {
		background-color: red;
	}
	90% {
		background-color: black;
	}
	100% {
		background-color: purple;
	}
}`)),
			expectedScopedClasses: nil,
			expectedError:         "",

			payload: strings.NewReader(`@import url("path/to/styles.css");
@keyframes myAnimation {
	from {
		background-color: red;
	}
	to {
		background-color: blue;
	}
}
@keyframes anotherAnimation {
	0% {
		background-color: green;
	}
	10% {
		background-color: red;
	}
	90% {
		background-color: black;
	}
	100% {
		background-color: purple;
	}
}`),
		},
		{
			name:                  "InvalidCSSModules_GlobalBlockMalformed",
			expectedCSSModules:    newMatchableCSS(true, nil),
			expectedScopedClasses: nil,
			expectedError:         ErrInvalidInputCSSModules.Error(),

			payload: strings.NewReader(`:global {.test-class { color: red; font-size: large; }`),
		},
		{
			name:                  "InvalidCSSModules_NilPayload",
			expectedCSSModules:    newMatchableCSS(true, nil),
			expectedScopedClasses: nil,
			expectedError:         ErrInvalidInputCSSModules.Error(),

			payload: nil,
		},
		{
			name:                  "InvalidCSSModules_ClassNameStartsWithSpace",
			expectedCSSModules:    newMatchableCSS(true, nil),
			expectedScopedClasses: nil,
			expectedError:         ErrInvalidInputCSSModules.Error(),

			payload: strings.NewReader(`. test-class { color:red; font-size: large;}`),
		},
		{
			name:                  "InvalidCSSModules_ClassNameStartsWithSpace_HasPseudoAndCombinator",
			expectedCSSModules:    newMatchableCSS(true, nil),
			expectedScopedClasses: nil,
			expectedError:         ErrInvalidInputCSSModules.Error(),

			payload: strings.NewReader(`. test-class :hover { color:green; font-size: medium; }`),
		},
	}
	for i := range testCases {
		tc := testCases[i]
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			css, scopedClasses, err := ProcessCSSModules(tc.payload)
			if err != nil {
				if err.Error() != tc.expectedError {
					t.Errorf("unexpected error value: expected %q got %q", tc.expectedError, err.Error())
					return
				}
			} else {
				if tc.expectedError != "" {
					t.Errorf("unexpected error value: expected %q got <nil>", tc.expectedError)
					return
				}
			}
			if tc.expectedCSSModules.canMatch {
				if !bytes.Equal(tc.expectedCSSModules.value, css) {
					t.Errorf("unexpected css slice of bytes value: expected\n%q\ngot\n%q", tc.expectedCSSModules.value, css)
					return
				}
			}
			for i := range tc.expectedScopedClasses {
				esc := tc.expectedScopedClasses[i]
				if _, ok := scopedClasses[esc]; !ok {
					t.Errorf("unexpected scopedClasses value absence: expected to have %q inside of it, got %q map", esc, scopedClasses)
					return
				}
			}
		})
	}
}

func TestCutSelectorAndPseudo(t *testing.T) {
	testCases := []struct {
		name                        string
		payload                     io.Reader
		expectedSelector            []byte
		expectedCombinatorAndPseudo []byte
		expectedError               string
	}{
		{
			name:                        "ValidPayload",
			expectedSelector:            []byte(`test-class`),
			expectedCombinatorAndPseudo: nil,
			expectedError:               "",

			payload: strings.NewReader(`test-class`),
		},
		{
			name:                        "ValidPayload_HasPseudoElement",
			expectedSelector:            []byte(`test-class`),
			expectedCombinatorAndPseudo: []byte(`:hover`),
			expectedError:               "",

			payload: strings.NewReader(`test-class:hover`),
		},
		{
			name:                        "ValidPayload_HasPseudoElement_HasSpaceCombinator",
			expectedSelector:            []byte(`test-class`),
			expectedCombinatorAndPseudo: []byte(` :hover`),
			expectedError:               "",

			payload: strings.NewReader(`test-class :hover`),
		},
		{
			name:                        "ValidPayload_HasPseudoElement_Has>Combinator",
			expectedSelector:            []byte(`test-class`),
			expectedCombinatorAndPseudo: []byte(`>:hover`),
			expectedError:               "",

			payload: strings.NewReader(`test-class > :hover`),
		},
		{
			name:                        "ValidPayload_HasNumericCharacter",
			expectedSelector:            []byte(`test-class-1`),
			expectedCombinatorAndPseudo: nil,
			expectedError:               "",

			payload: strings.NewReader(`test-class-1`),
		},
		{
			name:                        "InvalidPayload_NonAlphanumericSymbolsAreNotAllowed",
			expectedSelector:            nil,
			expectedCombinatorAndPseudo: nil,
			expectedError:               ErrInvalidInputCSSModules.Error(),

			payload: strings.NewReader(`|invalid!class`),
		},
	}
	for i := range testCases {
		tc := testCases[i]
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			selector, combinatorAndPseudo, err := cutSelectorAndPseudo(tc.payload)
			if err != nil {
				if err.Error() != tc.expectedError {
					t.Errorf("unexpected error value: expected %q got %q", tc.expectedError, err.Error())
					return
				}
			} else {
				if tc.expectedError != "" {
					t.Errorf("unexpected error value: expected %q got <nil>", tc.expectedError)
					return
				}
			}
			if !bytes.Equal(selector, tc.expectedSelector) {
				t.Errorf("unexpected selector value: expected %q got %q", tc.expectedSelector, selector)
				return
			}
			if !bytes.Equal(combinatorAndPseudo, tc.expectedCombinatorAndPseudo) {
				t.Errorf("unexpected combinatorAndPseudo value: expected %q got %q", tc.expectedCombinatorAndPseudo, combinatorAndPseudo)
				return
			}
		})
	}
}

func TestScopeCSSClass(t *testing.T) {
	testCases := []struct {
		name              string
		salt              string
		payload           io.Reader
		expectedCSS       matchableCSS
		expectedError     string
		expectedClassName []byte
	}{
		{
			name:              "ValidCSS",
			salt:              "",
			expectedError:     "",
			expectedCSS:       newMatchableCSS(false, nil),
			expectedClassName: []byte("test-class"),

			payload: strings.NewReader(`test-class {color: red; font-size: medium;}`),
		},
		{
			name:              "ValidCSS_WithPseudo",
			salt:              "",
			expectedError:     "",
			expectedCSS:       newMatchableCSS(false, nil),
			expectedClassName: []byte("test-class"),

			payload: strings.NewReader(`test-class:hover {color: red; font-size: medium;}`),
		},
		{
			name:              "ValidCSS_WithPseudo_WithCombinator",
			salt:              "",
			expectedError:     "",
			expectedCSS:       newMatchableCSS(false, nil),
			expectedClassName: []byte("test-class"),

			payload: strings.NewReader(`test-class :hover {color: red; font-size: medium;}`),
		},
		{
			name:              "InvalidCSS",
			salt:              "",
			expectedError:     ErrInvalidInputCSSModules.Error(),
			expectedCSS:       newMatchableCSS(true, nil),
			expectedClassName: nil,

			payload: strings.NewReader(`.test-class {}`),
		},
	}
	for i := range testCases {
		tc := testCases[i]
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			cssProcessed, scopedClass, err := scopeCSSClass(tc.payload, tc.salt)
			if err != nil {
				if err.Error() != tc.expectedError {
					t.Errorf("unexpected error value: expected %q got %q", tc.expectedError, err.Error())
					return
				}
			} else {
				if tc.expectedError != "" {
					t.Errorf("unexpected error value: expected %q got <nil>", tc.expectedError)
					return
				}
			}
			if tc.expectedCSS.canMatch {
				if !bytes.Equal(cssProcessed, tc.expectedCSS.value) {
					t.Errorf("unexpected cssProcessed value: expected %q got %q", tc.expectedCSS.value, cssProcessed)
					return
				}
			}
			if scopedClass != nil {
				if !bytes.Equal([]byte(scopedClass.originalClassName), tc.expectedClassName) {
					t.Errorf("unexpected class name value: expected %q got %q", tc.expectedClassName, scopedClass.originalClassName)
					return
				}
			} else {
				if !bytes.Equal(nil, tc.expectedClassName) {
					t.Errorf("unexpected class name value: expected %q got <nil>", tc.expectedClassName)
				}
			}
		})
	}
}
