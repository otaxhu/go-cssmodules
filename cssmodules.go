package cssmodules

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"io"
	"sync"

	"github.com/tdewolff/parse/v2"
	css_parser "github.com/tdewolff/parse/v2/css"
)

type CSSModulesParser struct {
	r              io.Reader
	alreadyWritten bool
}

func NewCSSModulesParser(css io.Reader) *CSSModulesParser {
	return &CSSModulesParser{r: css}
}

func (p *CSSModulesParser) ParseTo(w io.Writer) (map[string]string, error) {
	if p.alreadyWritten {
		return nil, ErrAlreadyWritten
	}
	if x, ok := w.(writer); ok {
		return processCSSModules(p.r, x)
	}
	buf := getBuffer()
	defer releaseBuffer(buf)
	classes, err := processCSSModules(p.r, buf)
	if err != nil {
		return nil, err
	}
	if _, err := buf.WriteTo(w); err != nil {
		return nil, err
	}
	p.alreadyWritten = true
	return classes, nil
}

// Parses the CSS and returns the CSS processed, the key-value pair of the
// classes and scoped classes, and an error if there is one
func ProcessCSSModules(css io.Reader) ([]byte, map[string]string, error) {

	bb := getBuffer()
	defer releaseBuffer(bb)

	scopedClasses, err := processCSSModules(css, bb)
	if err != nil {
		return nil, nil, err
	}
	cpBb := make([]byte, bb.Len())
	if _, err := bb.Read(cpBb); err != nil {
		return nil, nil, err
	}
	return cpBb, scopedClasses, nil
}

func processCSSModules(r io.Reader, w writer) (map[string]string, error) {
	zz := css_parser.NewLexer(parse.NewInput(r))
	scopedClasses := map[string]string{}

	mutex := &sync.Mutex{}

	salt := make([]byte, 4)
	if _, err := rand.Read(salt); err != nil {
		return nil, err
	}
	dataTempBuffer := getBuffer()
	defer releaseBuffer(dataTempBuffer)
mainLoop:
	for {
		zt, data := zz.Next()
		if zt == css_parser.ErrorToken {
			if err := zz.Err(); err == io.EOF {
				return scopedClasses, nil
			} else if err != nil {
				return nil, err
			}
		}

		dataTempBuffer.Write(data)

		if zt == css_parser.ColonToken {
			zt, data := zz.Next()
			if zt == css_parser.ErrorToken {
				continue mainLoop
			}
			if zt != css_parser.IdentToken {
				dataTempBuffer.WriteTo(w)
				w.Write(data)
				continue mainLoop
			}
			if string(data) != "global" {
				dataTempBuffer.WriteTo(w)
				w.Write(data)
				continue mainLoop
			}
			braceCount := 0
			for {
				zt, data := zz.Next()
				if zt == css_parser.ErrorToken {
					continue mainLoop
				}
				if zt == css_parser.LeftBraceToken {
					if braceCount != 0 {
						w.Write(data)
					}
					braceCount++
				} else if zt == css_parser.RightBraceToken {
					if braceCount != 1 {
						w.Write(data)
					}
					braceCount--
					if braceCount <= 0 {
						break
					}
				} else {
					w.Write(data)
				}
			}
		} else if zt == css_parser.AtKeywordToken {
			if _, err := dataTempBuffer.WriteTo(w); err != nil {
				return nil, err
			}
			zt, data := zz.Next()
			if zt == css_parser.ErrorToken {
				continue mainLoop
			}
			w.Write(data)
			if zt != css_parser.IdentToken {
				continue mainLoop
			}
			if string(data) != "media" {
				continue mainLoop
			}
			for {
				zt, data := zz.Next()
				if zt == css_parser.ErrorToken {
					continue mainLoop
				}

				w.Write(data)

				braceCount := 0
				if zt == css_parser.DelimToken && string(data) == "." {
					zt, data := zz.Next()
					if zt == css_parser.ErrorToken {
						continue mainLoop
					}
					if zt != css_parser.IdentToken {
						w.Write(data)
						continue
					}
					scopeCSSClass(data, salt, w, scopedClasses, mutex)
				} else if zt == css_parser.LeftBraceToken {
					braceCount++
				} else if zt == css_parser.RightBraceToken {
					braceCount--
					if braceCount <= 0 {
						break
					}
				}
			}
		} else if zt == css_parser.DelimToken && string(data) == "." {
			if _, err := dataTempBuffer.WriteTo(w); err != nil {
				return nil, err
			}
			zt, data := zz.Next()
			if zt == css_parser.ErrorToken {
				continue mainLoop
			}
			if zt != css_parser.IdentToken {
				w.Write(data)
				continue mainLoop
			}
			scopeCSSClass(data, salt, w, scopedClasses, mutex)
		} else {
			if _, err := dataTempBuffer.WriteTo(w); err != nil {
				return nil, err
			}
		}
		dataTempBuffer.Reset()
	}
}

func scopeCSSClass(data []byte, salt []byte, w writer, scopedClasses map[string]string, mutex *sync.Mutex) {
	defer adlerHashFunction.Reset()
	defer mutex.Unlock()
	mutex.Lock()
	adlerHashFunction.Write(data)
	adlerHashFunction.Write(salt)

	checksum := adlerHashFunction.Sum32()

	bufChecksumUint32 := make([]byte, 4)
	binary.NativeEndian.PutUint32(bufChecksumUint32, checksum)

	encodedChecksum := base64.RawURLEncoding.EncodeToString(bufChecksumUint32)

	tempBuffer := getBuffer()
	defer releaseBuffer(tempBuffer)

	tempBuffer.WriteByte('_')
	tempBuffer.Write(data)
	tempBuffer.WriteByte('_')
	tempBuffer.WriteString(encodedChecksum)

	bufScopedClassName := make([]byte, tempBuffer.Len())
	copy(bufScopedClassName, tempBuffer.Bytes())

	tempBuffer.WriteTo(w)

	scopedClasses[string(data)] = string(bufScopedClassName)
}
