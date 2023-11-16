package cssmodules

import (
	"bytes"
	"io"

	html_parser "golang.org/x/net/html"
)

type HTMLCSSModulesParser struct {
	r              io.Reader
	sc             map[string]string
	alreadyWritten bool
}

func NewHTMLCSSModulesParser(html io.Reader, scopedClasses map[string]string) *HTMLCSSModulesParser {
	return &HTMLCSSModulesParser{
		r:  html,
		sc: scopedClasses,
	}
}

func (p *HTMLCSSModulesParser) ParseTo(w io.Writer) error {
	if p.alreadyWritten {
		return ErrAlreadyWritten
	}
	if x, ok := w.(writer); ok {
		return parseHTMLWithCSSModules(p.r, x, p.sc)
	}
	buf := getBuffer()
	defer releaseBuffer(buf)
	if err := parseHTMLWithCSSModules(p.r, buf, p.sc); err != nil {
		return err
	}
	if _, err := buf.WriteTo(w); err != nil {
		return err
	}
	p.alreadyWritten = true
	return nil
}

func ProcessHTMLWithCSSModules(html io.Reader, scopedClasses map[string]string) ([]byte, error) {
	buf := getBuffer()
	defer releaseBuffer(buf)
	if err := parseHTMLWithCSSModules(html, buf, scopedClasses); err != nil {
		return nil, err
	}
	cpBuf := make([]byte, buf.Len())
	if _, err := buf.Read(cpBuf); err != nil {
		return nil, err
	}
	return cpBuf, nil
}

func parseHTMLWithCSSModules(r io.Reader, w writer, scopedClasses map[string]string) error {

	zz := html_parser.NewTokenizer(r)

mainLoop:
	for {
		zt := zz.Next()
		if err := zz.Err(); err == io.EOF {
			return nil
		} else if err != nil {
			return err
		}

		// Exclude tokens that doesn't have attributes and writes to w
		if zt != html_parser.StartTagToken && zt != html_parser.SelfClosingTagToken {
			w.Write(zz.Raw())
			continue
		}

		tagName, hasAttr := zz.TagName()

		// If the tag doesn't have attributes at all, then there is no need to read it.
		// It's directly written to w
		if !hasAttr {
			w.Write(zz.Raw())
			continue
		}

		w.WriteByte('<')
		w.Write(tagName)

		var (
			hasClassAttr bool
			classVal     []byte
		)

		var (
			cssModulesVal []byte
		)

		for {
			tagAttrKey, tagAttrVal, hasMoreAttr := zz.TagAttr()
			if string(tagAttrKey) == "css-module" {
				cssModulesVal = tagAttrVal
			} else if string(tagAttrKey) == "class" {
				hasClassAttr = true
				classVal = tagAttrVal
			} else {
				w.WriteByte(' ')
				w.Write(tagAttrKey)
				w.WriteString(`="`)
				w.WriteString(html_parser.EscapeString(string(tagAttrVal)))
				w.WriteByte('"')
			}
			if !hasMoreAttr {
				if cssModulesVal == nil {
					if !hasClassAttr {
						w.WriteByte('>')
						continue mainLoop
					}
					w.WriteString(` class="`)
					w.WriteString(html_parser.EscapeString(string(bytes.TrimSpace(classVal))))
					w.WriteString(`">`)
					continue mainLoop
				}
				w.WriteString(` class="`)
				if hasClassAttr {
					w.WriteString(html_parser.EscapeString(string(bytes.TrimSpace(classVal))))
					w.WriteByte(' ')
				}
				break
			}
		}

		classes := bytes.Split(cssModulesVal, []byte{' '})
		for i, c := range classes {
			// If equals empty then ignore the consumer's HTML syntax error and continue
			if bytes.Equal(c, nil) {
				continue
			}
			class, exists := scopedClasses[string(c)]
			if !exists {
				return ErrClassNotFound
			}
			if i != 0 {
				w.WriteByte(' ')
			}
			w.WriteString(class)
		}
		w.WriteString(`">`)
	}
}
