// Package anonymizer masks sensitive values in text before it is stored or sent
// to a model. It replaces this repository's former dependency on
// github.com/vxcontrol/cloud/anonymizer, which was not distributable under this
// project's licence.
package anonymizer

import (
	"fmt"
	"io"
	"regexp"
	"strings"
)

// Replacer masks sensitive values in text.
type Replacer interface {
	ReplaceString(text string) string
	ReplaceBytes(data []byte) []byte
	WrapReader(r io.Reader) io.Reader
}

// NewReplacer pairs every expression with the name its matches are masked under.
//
// A pattern whose expression carries a group named replace masks only that span;
// without it the whole match is masked. Names and regexes must be the same
// length and align positionally.
func NewReplacer(regexes []*regexp.Regexp, names []string) (Replacer, error) {
	if len(regexes) != len(names) {
		return nil, fmt.Errorf("anonymizer: %d regexes against %d names", len(regexes), len(names))
	}

	entries := make([]entry, 0, len(regexes))
	for i, re := range regexes {
		if re == nil {
			return nil, fmt.Errorf("anonymizer: regex %d (%s) is nil", i, names[i])
		}
		entries = append(entries, entry{re: re, name: names[i], group: re.SubexpIndex("replace")})
	}
	return &replacer{entries: entries}, nil
}

type entry struct {
	re    *regexp.Regexp
	name  string
	group int
}

type replacer struct {
	entries []entry
}

// mask is the placeholder the web UI recognises and renders as an anonymised
// span; its shape must not change without the frontend.
func mask(name string) string {
	return "§*" + name + "*§"
}

func (r *replacer) ReplaceString(text string) string {
	for _, e := range r.entries {
		if !e.re.MatchString(text) {
			continue
		}
		if e.group >= 0 {
			text = replaceGroup(e.re, e.group, text, mask(e.name))
		} else {
			text = e.re.ReplaceAllLiteralString(text, mask(e.name))
		}
	}
	return text
}

func (r *replacer) ReplaceBytes(data []byte) []byte {
	return []byte(r.ReplaceString(string(data)))
}

func (r *replacer) WrapReader(src io.Reader) io.Reader {
	return &replacedReader{src: src, replace: r.ReplaceString}
}

// replaceGroup swaps every match of one capture group for mark, keeping the
// text around it — the caller's pattern names the exact span to hide and says
// with what context it should stay readable.
func replaceGroup(re *regexp.Regexp, group int, text, mark string) string {
	matches := re.FindAllStringSubmatchIndex(text, -1)

	var b strings.Builder
	last, wrote := 0, false
	for _, m := range matches {
		start, end := m[2*group], m[2*group+1]
		if start < 0 {
			continue
		}
		b.WriteString(text[last:start])
		b.WriteString(mark)
		last, wrote = end, true
	}
	if !wrote {
		return text
	}
	b.WriteString(text[last:])
	return b.String()
}

// replacedReader reads its source on the first Read, then serves the masked
// text; an error the source produced is handed to the caller unchanged.
type replacedReader struct {
	src     io.Reader
	replace func(string) string

	buf  *strings.Reader
	done bool
}

func (rr *replacedReader) Read(p []byte) (int, error) {
	if !rr.done {
		data, err := io.ReadAll(rr.src)
		if err != nil {
			return 0, err
		}
		rr.buf = strings.NewReader(rr.replace(string(data)))
		rr.done = true
	}
	return rr.buf.Read(p)
}
