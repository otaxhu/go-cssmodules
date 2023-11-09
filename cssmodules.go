package cssmodules

import (
	"bytes"
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

// It's the base32 encoding but without = signs on the output
var base32StdEncodingNoPad = base32.StdEncoding.WithPadding(-1)

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
		if b == '/' {
			if starByte, err := bb.ReadByte(); err != nil {
				return nil, nil, ErrInvalidInputCSSModules
			} else if starByte != '*' {
				resultingCSS.WriteByte(starByte)
				continue
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
		} else if b == '@' {
			indexSpace := bytes.IndexByte(bb.Bytes(), ' ')
			if indexSpace <= 0 {
				return nil, nil, ErrInvalidInputCSSModules
			}
			specialDeclarationName := bb.Next(indexSpace)
			indexOpenBracket := bytes.IndexByte(bb.Bytes(), '{')
			if indexOpenBracket <= 0 {
				return nil, nil, ErrInvalidInputCSSModules
			}
			if bytes.Equal(specialDeclarationName, []byte("media")) {
				tmpBuffer := getBuffer()
				defer releaseBuffer(tmpBuffer)
				tmpBuffer.WriteString("@media")
				conditions := bb.Next(indexOpenBracket + 1)
				tmpBuffer.Write(conditions)
				bracesCount := 1
				for {
					byteAfterMediaQuery, err := bb.ReadByte()
					if err == io.EOF {
						if bracesCount != 0 {
							return nil, nil, ErrInvalidInputCSSModules
						}
						break
					} else if err != nil {
						return nil, nil, ErrInvalidInputCSSModules
					}
					if byteAfterMediaQuery == '.' {
						indexStartStyles := bytes.IndexByte(bb.Bytes(), '{')
						if indexStartStyles <= 0 {
							return nil, nil, ErrInvalidInputCSSModules
						}
						bracesCount++
						indexEndStyles := bytes.IndexByte(bb.Bytes(), '}')
						if indexEndStyles <= indexStartStyles {
							return nil, nil, ErrInvalidInputCSSModules
						}
						bracesCount--
						classRule := bb.Next(indexEndStyles + 1)
						classRule, scopedClasses, err := scopeCSSClass(bytes.NewReader(classRule), salt)
						if err != nil {
							return nil, nil, err
						}
						tmpBuffer.Write(classRule)
						kvClasses[scopedClasses.originalClassName] = scopedClasses.scopedClassName
					} else {
						tmpBuffer.WriteByte(byteAfterMediaQuery)
						if byteAfterMediaQuery == '{' {
							bracesCount++
						} else if byteAfterMediaQuery == '}' {
							bracesCount--
						}
						if bracesCount == 0 {
							break
						}
					}
				}
				if _, err := tmpBuffer.WriteTo(resultingCSS); err != nil {
					return nil, nil, ErrUnexpectedError
				}
			} else {

			}
		} else if b == '.' {
			indexStartStyles := bytes.IndexByte(bb.Bytes(), '{')
			if indexStartStyles <= 0 {
				return nil, nil, ErrInvalidInputCSSModules
			}
			indexEndStyles := bytes.IndexByte(bb.Bytes(), '}')
			if indexEndStyles <= indexStartStyles {
				return nil, nil, ErrInvalidInputCSSModules
			}
			cssClassRule := bb.Next(indexEndStyles + 1)
			cpCssClassRule := make([]byte, len(cssClassRule), len(cssClassRule))
			copy(cpCssClassRule, cssClassRule)
			cpCpCssClassRule, scopedClass, err := scopeCSSClass(bytes.NewReader(cpCssClassRule), salt)
			if err != nil {
				return nil, nil, err
			}
			resultingCSS.Write(cpCpCssClassRule)
			kvClasses[scopedClass.originalClassName] = scopedClass.scopedClassName
		} else if b == ':' {
			indexStartStyles := bytes.IndexByte(bb.Bytes(), '{')
			if indexStartStyles < 0 {
				resultingCSS.WriteByte(b)
				continue
			} else if indexStartStyles == 0 {
				return nil, nil, ErrInvalidInputCSSModules
			}
			keyword := bb.Next(indexStartStyles)
			keyword = rightSpacesRegexp.ReplaceAll(keyword, []byte{})
			cpKeyword := make([]byte, len(keyword), len(keyword))
			copy(cpKeyword, keyword)
			if !bytes.Equal(keyword, []byte("global")) {
				resultingCSS.WriteByte(b)
				resultingCSS.Write(cpKeyword)
				continue
			}
			if _, err := bb.ReadByte(); err != nil {
				return nil, nil, ErrInvalidInputCSSModules
			}
			tmpBuffer := getBuffer()
			defer releaseBuffer(tmpBuffer)
			braceCount := 1
			for {
				byteAfterStartStyles, err := bb.ReadByte()
				if err != nil {
					return nil, nil, ErrInvalidInputCSSModules
				}
				tmpBuffer.WriteByte(byteAfterStartStyles)
				if byteAfterStartStyles == '{' {
					braceCount++
				} else if byteAfterStartStyles == '}' {
					braceCount--
					if braceCount == 0 {
						tmpBuffer.Truncate(tmpBuffer.Len() - 1)
						break
					}
				}
			}
			cpTmpBuffer := make([]byte, tmpBuffer.Len(), tmpBuffer.Len())
			if _, err := tmpBuffer.Read(cpTmpBuffer); err != nil {
				return nil, nil, ErrInvalidInputCSSModules
			}
			releaseBuffer(tmpBuffer)
			cpTmpBuffer = bytes.TrimSpace(cpTmpBuffer)
			resultingCSS.Write(cpTmpBuffer)
		} else {
			resultingCSS.WriteByte(b)
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
	c := rightSpacesRegexp.ReplaceAll(bb.Bytes(), []byte{})

	bef, aft, hasPseudo := bytes.Cut(c, []byte{':'})

	if !hasPseudo {
		befAltered := rightSpacesRegexp.ReplaceAll(bef, []byte{})
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
	befAltered = rightSpacesRegexp.ReplaceAll(befAltered, []byte{})
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

// scopeCSSClass receives a css class and process it and then returns the processed css blob and
// the class name
//
// The css param must be only one class.
//
// (IMPORTANT): the class must not start with a dot . otherwise it will return a error
func scopeCSSClass(css io.Reader, salt string) ([]byte, *struct {
	originalClassName string
	scopedClassName   string
}, error) {
	if css == nil {
		return nil, nil, ErrInvalidInputCSSModules
	}

	bb := getBuffer()
	defer releaseBuffer(bb)

	if _, err := bb.ReadFrom(css); err != nil {
		return nil, nil, ErrInvalidInputCSSModules
	}

	resultingCSS := getBuffer()
	defer releaseBuffer(resultingCSS)

	indexStartStyles := bytes.IndexByte(bb.Bytes(), '{')
	if indexStartStyles <= 0 {
		return nil, nil, ErrInvalidInputCSSModules
	}

	selector := bb.Next(indexStartStyles)
	selector, pseudo, err := cutSelectorAndPseudo(bytes.NewReader(selector))
	if err != nil {
		return nil, nil, err
	}
	restClass := make([]byte, bb.Len(), bb.Len())
	if _, err := bb.Read(restClass); err != nil {
		return nil, nil, ErrUnexpectedError
	}
	tmpBuffer := getBuffer()
	defer releaseBuffer(tmpBuffer)
	tmpBuffer.Write(selector)
	tmpBuffer.WriteString(salt)

	encodedClassName := base32StdEncodingNoPad.EncodeToString(tmpBuffer.Bytes())

	tmpBuffer.Reset()

	// This is the construction of the scoped class name
	tmpBuffer.WriteByte('_')
	tmpBuffer.Write(selector)
	tmpBuffer.WriteByte('_')
	tmpBuffer.WriteString(encodedClassName)

	cpTmpBuffer := make([]byte, tmpBuffer.Len(), tmpBuffer.Len())
	copy(cpTmpBuffer, tmpBuffer.Bytes())

	resultingCSS.WriteByte('.')
	if _, err := resultingCSS.ReadFrom(tmpBuffer); err != nil {
		return nil, nil, ErrUnexpectedError
	}
	resultingCSS.Write(pseudo)
	resultingCSS.WriteByte(' ')
	resultingCSS.Write(restClass)

	cpResultingCSS := make([]byte, resultingCSS.Len(), resultingCSS.Len())
	if _, err := resultingCSS.Read(cpResultingCSS); err != nil {
		return nil, nil, ErrUnexpectedError
	}
	classNameScoped := &struct {
		originalClassName string
		scopedClassName   string
	}{
		originalClassName: string(selector),
		scopedClassName:   string(cpTmpBuffer),
	}

	return cpResultingCSS, classNameScoped, nil
}
