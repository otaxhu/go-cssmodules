package cssmodules

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

func newExpectedCSSModules(canMatch bool, value []byte) struct {
	canMatch bool
	value    []byte
} {
	return struct {
		canMatch bool
		value    []byte
	}{
		canMatch: canMatch,
		value:    value,
	}
}

func TestProcessCSSModules(t *testing.T) {
	testCases := []struct {
		name               string
		payload            io.Reader
		expectedCSSModules struct {
			canMatch bool
			value    []byte
		}
		expectedScopedClasses []string
		expectedErr           string
	}{
		{
			name:                  "ValidCSSModules",
			expectedCSSModules:    newExpectedCSSModules(false, nil),
			expectedScopedClasses: []string{"test-class"},
			expectedErr:           "",

			payload: strings.NewReader(`.test-class {
    color: red;
    font-size: large;
}`),
		},
		{
			name: "ValidCSSModules_GlobalKeyword",
			expectedCSSModules: newExpectedCSSModules(true,
				[]byte(`.test-class { color: red; font-size: large; }`),
			),
			expectedScopedClasses: nil,
			expectedErr:           "",

			payload: strings.NewReader(`:global {.test-class { color: red; font-size: large; }}`),
		},
		{
			name:                  "ValidCSSModules_MediaQueryScoping",
			expectedCSSModules:    newExpectedCSSModules(false, nil),
			expectedScopedClasses: []string{"test-class"},
			expectedErr:           "",

			payload: strings.NewReader(`@media screen and (min-width: 768px) and (max-width: 1024px) {
	.test-class {
		color: green;
		font-size: large;
	}
}`),
		},
		{
			name: "ValidCSSModules_Comments",
			expectedCSSModules: newExpectedCSSModules(true,
				[]byte(`/* Test Comments *//* Not Closing Comment`),
			),
			expectedScopedClasses: nil,
			expectedErr:           "",

			payload: strings.NewReader(`/* Test Comments */ /* Not Closing Comment`),
		},
		{
			name:                  "InvalidCSSModules_ID#SymbolWillNotBeRead",
			expectedCSSModules:    newExpectedCSSModules(true, nil),
			expectedScopedClasses: nil,
			expectedErr:           ErrInvalidInputCSSModules.Error(),

			payload: strings.NewReader(`#test-class {}`),
		},
		{
			name:                  "InvalidCSSModules_GlobalBlockMalformed",
			expectedCSSModules:    newExpectedCSSModules(true, nil),
			expectedScopedClasses: nil,
			expectedErr:           ErrInvalidInputCSSModules.Error(),

			payload: strings.NewReader(`:global {.test-class { color: red; font-size: large; }`),
		},
		{
			name:                  "InvalidCSSModules_NilPayload",
			expectedCSSModules:    newExpectedCSSModules(true, nil),
			expectedScopedClasses: nil,
			expectedErr:           ErrInvalidInputCSSModules.Error(),

			payload: nil,
		},
	}
	for i := range testCases {
		tc := testCases[i]
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			css, scopedClasses, err := ProcessCSSModules(tc.payload)
			if err != nil {
				if err.Error() != tc.expectedErr {
					t.Errorf("unexpected error value: expected %q got %q", tc.expectedErr, err.Error())
					return
				}
			} else {
				if tc.expectedErr != "" {
					t.Errorf("unexpected error value: expected %q got %q", tc.expectedErr, "")
					return
				}
			}
			if tc.expectedCSSModules.canMatch {
				if !bytes.Equal(tc.expectedCSSModules.value, css) {
					t.Errorf("unexpected css slice of bytes value: expected %q got %q", tc.expectedCSSModules.value, css)
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
