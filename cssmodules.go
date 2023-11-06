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
	ErrInvalidInputCSSModules = errors.New("go-cssmodules: the input css cannot be converted to css modules")

	ErrUnexpectedError = errors.New("go-cssmodules: appears to happen an unexpected error")
)

var (
	validCSSModulesSelectorRegexp = regexp.MustCompile(`^[a-zA-Z][0-9\w-]*$`)

	rightSpacesRegexp = regexp.MustCompile(`[\s]+$`)

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

// Process the CSS and returns the css processed, the key-value pair of the
// classes and scoped classes, and an error if there is one
func ProcessCSSModules(css io.Reader) ([]byte, map[string]string, error) {
	if css == nil {
		return nil, nil, ErrInvalidInputCSSModules
	}
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
		return nil, nil, ErrInvalidInputCSSModules
	}

	salt := uuid.NewString()

	// The check sum is based on the selector and a salt value.
	checkSumBuffer := getBuffer()
	defer releaseBuffer(checkSumBuffer)

	for {
		b, err := bb.ReadByte()
		if err == io.EOF {
			if resultingCSS.Len() == 0 {
				return nil, nil, ErrInvalidInputCSSModules
			}
			cpResultingCSS := make([]byte, resultingCSS.Len(), resultingCSS.Len())
			if _, err := resultingCSS.Read(cpResultingCSS); err != nil {
				return nil, nil, ErrUnexpectedError
			}
			return cpResultingCSS, kvClasses, nil
		} else if err != nil {
			return nil, nil, ErrInvalidInputCSSModules
		}
		// It's a comment
		if b == '/' {
			if starByte, err := bb.ReadByte(); err != nil {
				return nil, nil, ErrInvalidInputCSSModules
			} else if starByte != '*' {
				return nil, nil, ErrInvalidInputCSSModules
			}
			resultingCSS.WriteString("/*")
			for {
				byteContentComment, err := bb.ReadByte()
				if err == io.EOF {
					break
				} else if err != nil {
					return nil, nil, ErrInvalidInputCSSModules
				}
				resultingCSS.WriteByte(byteContentComment)
				if byteContentComment == '*' {
					byteAfterStar, err := bb.ReadByte()
					if err == io.EOF {
						break
					} else if err != nil {
						return nil, nil, ErrInvalidInputCSSModules
					}
					resultingCSS.WriteByte(byteAfterStar)
					if byteAfterStar == '/' {
						break
					}
				}
			}
		}
		// It's a special declaration, the classes inside of it will be scoped too
		if b == '@' {

		}
		// It's a class
		if b == '.' {
			indexStartStyles := bytes.IndexByte(bb.Bytes(), '{')
			if indexStartStyles <= 0 {
				return nil, nil, ErrInvalidInputCSSModules
			}
			indexEndStyles := bytes.IndexByte(bb.Bytes(), '}')
			if indexEndStyles <= indexStartStyles {
				return nil, nil, ErrInvalidInputCSSModules
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
			cpCheckSumBuffer := make([]byte, checkSumBuffer.Len(), checkSumBuffer.Len())
			if _, err := checkSumBuffer.Read(cpCheckSumBuffer); err != nil {
				return nil, nil, ErrUnexpectedError
			}
			checkSumBuffer.Reset()
			selectorCheckSum := sha256.Sum256(cpCheckSumBuffer)

			s := base32.StdEncoding.EncodeToString(selectorCheckSum[:])
			encodedCheckSum := s[:len(s)/2]
			resultingCSS.WriteByte(b)
			resultingCSS.WriteString(encodedCheckSum)
			resultingCSS.Write(combinatorAndPseudo)
			resultingCSS.Write(content)
			kvClasses[string(selector)] = encodedCheckSum
			continue
		}
		// It's a global block
		if b == ':' {
			indexStartStyles := bytes.IndexByte(bb.Bytes(), '{')
			if indexStartStyles <= 0 {
				return nil, nil, ErrInvalidInputCSSModules
			}
			keyword := bb.Next(indexStartStyles)
			keyword = rightSpacesRegexp.ReplaceAll(keyword, []byte{})
			if !bytes.Equal(keyword, []byte("global")) {
				return nil, nil, ErrInvalidInputCSSModules
			}
			if _, err := bb.ReadByte(); err != nil {
				return nil, nil, ErrInvalidInputCSSModules
			}
			braceCount := 1
			for {
				byteAfterStartStyles, err := bb.ReadByte()
				if err != nil {
					return nil, nil, ErrInvalidInputCSSModules
				}
				resultingCSS.WriteByte(byteAfterStartStyles)
				if byteAfterStartStyles == '{' {
					braceCount++
				} else if byteAfterStartStyles == '}' {
					braceCount--
					if braceCount == 0 {
						resultingCSS.Truncate(resultingCSS.Len() - 1)
						break
					}
				}
			}
		}
	}
}

// cutSelectorAndPseudo retrieves the class selector and the pseudo selector,
// and the combinator if it has one.
//
// This function also validates if the selector is a valid css selector.
//
// The returned slices are copies of the buffers so you don't have to worry about
// unexpected behaviour with the buffer pool releasing the buffers
func cutSelectorAndPseudo(selectorAndPseudo io.Reader) ([]byte, []byte, error) {
	bb := getBuffer()
	defer releaseBuffer(bb)
	if _, err := bb.ReadFrom(selectorAndPseudo); err != nil {
		return nil, nil, err
	}
	c := bytes.TrimSpace(bb.Bytes())

	bef, aft, hasPseudo := bytes.Cut(c, []byte{':'})

	if !hasPseudo {
		befAltered := bytes.TrimSpace(bef)
		if !validCSSModulesSelectorRegexp.Match(befAltered) {
			return nil, nil, ErrInvalidInputCSSModules
		}
		cpBefAltered := make([]byte, len(befAltered), len(befAltered))
		copy(cpBefAltered, befAltered)
		return cpBefAltered, nil, nil
	}

	combinator := combinatorSelectorRegexp.FindAll(bef, 2)

	// This buffer holds both combinator and colons
	combinatorBuffer := getBuffer()
	defer releaseBuffer(combinatorBuffer)

	if combinator == nil {
		befAltered := bytes.TrimSpace(bef)
		if !validCSSModulesSelectorRegexp.Match(befAltered) {
			return nil, nil, ErrInvalidInputCSSModules
		}
		if len(bef) != len(befAltered) {
			combinatorBuffer.WriteByte(' ')
		}
	} else if len(combinator) == 1 {
		combinatorBuffer.Write(combinator[0])
	} else {
		return nil, nil, ErrInvalidInputCSSModules
	}
	colons := pseudoColonRegexp.FindAll(aft, -1)
	if colons == nil {
		combinatorBuffer.WriteByte(':')
	} else if len(colons) == 1 {
		combinatorBuffer.WriteString("::")
	} else {
		return nil, nil, ErrInvalidInputCSSModules
	}
	aftAltered := pseudoColonRegexp.ReplaceAll(aft, []byte{})
	aftAltered = bytes.TrimSpace(aftAltered)
	if !validCSSModulesSelectorRegexp.Match(aftAltered) {
		return nil, nil, ErrInvalidInputCSSModules
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
		return nil, nil, ErrInvalidInputCSSModules
	}
	cpBefAltered := make([]byte, len(befAltered), len(befAltered))
	copy(cpBefAltered, befAltered)

	cpCombinatorBuffer := make([]byte, combinatorBuffer.Len(), combinatorBuffer.Len())
	if _, err := combinatorBuffer.Read(cpCombinatorBuffer); err != nil {
		return nil, nil, ErrUnexpectedError
	}

	return cpBefAltered, cpCombinatorBuffer, nil
}
