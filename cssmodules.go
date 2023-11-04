package cssmodules

import (
	"bytes"
	"crypto/sha256"
	"encoding/base32"
	"errors"
	"io"
	"regexp"
	"sync"

	"github.com/google/uuid"
)

var (
	ErrInvalidCSSModules = errors.New("go-cssmodules: the input css cannot be converted to css modules")

	ErrUnexpectedError = errors.New("go-cssmodules: appears to happen an unexpected error")
)

var (
	validCSSModulesSelectorRegexp = regexp.MustCompile(`^[a-zA-Z][\w-]*$`)

	// Matches all of the combinator except for the space which is a special case
	combinatorSelectorRegexp = regexp.MustCompile(`[+>~]`)

	pseudoColonRegexp = regexp.MustCompile(`[:]`)
)

var bp = sync.Pool{
	New: func() any {
		return &bytes.Buffer{}
	},
}

func getBuffer() *bytes.Buffer {
	return bp.Get().(*bytes.Buffer)
}

func releaseBuffer(b *bytes.Buffer) {
	if b != nil {
		b.Reset()
		bp.Put(b)
	}
}

// Process the CSS and returns the css processed, the key-value pair of the
// classes and scoped classes, and an error if there is one
func ProcessCSSModules(css io.Reader) ([]byte, map[string]string, error) {
	// Buffer that will contain the css processed and will be returned if there is
	// no error
	resultingCSS := getBuffer()
	defer releaseBuffer(resultingCSS)

	// The key-value pair that contains the classes scoped (the css modules classes)
	// and the actual classes
	kvClasses := map[string]string{}

	bb := getBuffer()
	defer releaseBuffer(bb)
	if _, err := bb.ReadFrom(css); err != nil {
		return nil, nil, err
	}

	salt := uuid.NewString()

	for {
		// The check sum is based on the selector and a salt value.
		checkSumBuffer := getBuffer()
		defer releaseBuffer(checkSumBuffer)

		b, err := bb.ReadByte()
		if err == io.EOF {
			if resultingCSS.Len() == 0 {
				return nil, nil, ErrInvalidCSSModules
			}
			return resultingCSS.Bytes(), kvClasses, nil
		} else if err != nil {
			return nil, nil, err
		}
		if b == '.' {
			firstByteAfterDot, err := bb.ReadByte()
			if err != nil {
				return nil, nil, err
			}
			if !validCSSModulesSelectorRegexp.Match([]byte{firstByteAfterDot}) {
				return nil, nil, ErrInvalidCSSModules
			}
			bb.UnreadByte()
			indexStartStyles := bytes.IndexByte(bb.Bytes(), '{')
			if indexStartStyles <= 0 {
				return nil, nil, ErrInvalidCSSModules
			}
			indexEndStyles := bytes.IndexByte(bb.Bytes(), '}')
			if indexEndStyles <= indexStartStyles {
				return nil, nil, ErrInvalidCSSModules
			}
			selector := bb.Next(indexStartStyles)
			selector, combinatorAndPseudo, err := cutSelectorAndPseudo(bytes.NewReader(selector))
			if err != nil {
				return nil, nil, err
			}
			indexEndStyles = bytes.IndexByte(bb.Bytes(), '}')
			// The rules of the selector
			content := bb.Next(indexEndStyles + 1)

			checkSumBuffer.Write(selector)
			checkSumBuffer.WriteString(salt)
			selectorCheckSum := sha256.Sum256(checkSumBuffer.Bytes())
			releaseBuffer(checkSumBuffer)

			s := base32.StdEncoding.EncodeToString(selectorCheckSum[:])
			encodedCheckSum := s[:len(s)/2]
			resultingCSS.WriteByte(b)
			resultingCSS.WriteString(encodedCheckSum)
			resultingCSS.Write(combinatorAndPseudo)
			resultingCSS.Write(content)
			kvClasses[string(selector)] = encodedCheckSum
			continue
		}
		// if b == ':' {
		// 	indexStartStyles := bytes.IndexByte(bb.Bytes(), '{')

		// }
	}
}

// cutSelectorAndPseudo retrieves the class selector and the pseudo selector,
// and the combinator if it has one.
// Also reports whether the class has a pseudo selector through the bool return value,
// the underlying payload class is guaranteed to be unmodified thanks to the io.Reader
// interface
func cutSelectorAndPseudo(class io.Reader) ([]byte, []byte, error) {
	bb := getBuffer()
	defer releaseBuffer(bb)
	if _, err := bb.ReadFrom(class); err != nil {
		return nil, nil, err
	}
	c := bytes.TrimSpace(bb.Bytes())

	bef, aft, hasPseudo := bytes.Cut(c, []byte{':'})

	if !hasPseudo {
		befAltered := bytes.TrimSpace(bef)
		if !validCSSModulesSelectorRegexp.Match(befAltered) {
			return nil, nil, ErrInvalidCSSModules
		}
		return befAltered, nil, nil
	}

	combinator := combinatorSelectorRegexp.FindAll(bef, 2)

	// This buffer holds both combinator and colons
	combinatorBuffer := getBuffer()

	if combinator == nil {
		befAltered := bytes.TrimSpace(bef)
		if !validCSSModulesSelectorRegexp.Match(befAltered) {
			return nil, nil, ErrInvalidCSSModules
		}
		if len(bef) != len(befAltered) {
			combinatorBuffer.WriteByte(' ')
		}
	} else if len(combinator) == 1 {
		combinatorBuffer.Write(combinator[0])
	} else {
		return nil, nil, ErrInvalidCSSModules
	}
	colons := pseudoColonRegexp.FindAll(aft, -1)
	if colons == nil {
		combinatorBuffer.WriteByte(':')
	} else if len(colons) == 1 {
		combinatorBuffer.WriteString("::")
	} else {
		return nil, nil, ErrInvalidCSSModules
	}
	aftAltered := pseudoColonRegexp.ReplaceAll(aft, []byte{})
	aftAltered = bytes.TrimSpace(aftAltered)
	if !validCSSModulesSelectorRegexp.Match(aftAltered) {
		return nil, nil, ErrInvalidCSSModules
	}
	combinatorBuffer.Write(aftAltered)

	//////////////////
	//
	// From now on the combinatorBuffer has been succesfully constructed with the
	// colons and its combinator
	//
	//////////////////
	befAltered := combinatorSelectorRegexp.ReplaceAllLiteral(bef, []byte{})
	befAltered = bytes.TrimSpace(befAltered)
	if !validCSSModulesSelectorRegexp.Match(befAltered) {
		return nil, nil, ErrInvalidCSSModules
	}

	return befAltered, combinatorBuffer.Bytes(), nil
}
