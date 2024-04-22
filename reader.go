package nutclient

import (
	"bufio"
	"errors"
	"io"
	"strings"
)

var (
	errMissingEndQuote = errors.New("missing \"")

	errBeginListMissing = errors.New("BEGIN LIST expected")
	errVarExpected      = errors.New("VAR expected")
	errVarNameMissing   = errors.New("variable name expected")
	errVarValueMissing  = errors.New("variable value expected")
	errUnexpectedEof    = errors.New("unexpected EOF")
)

func isSpace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\r' || b == '\n'
}

func split(data []byte, atEOF bool) (advance int, token []byte, err error) {

	// Skip whitespace
	for ; advance < len(data) && isSpace(data[advance]); advance++ {
	}

	// If there is nothing beyond the whitespace, return no token
	if advance == len(data) {
		if atEOF {
			err = bufio.ErrFinalToken
		}
		return
	}

	// If the next character is an open quote, read until end quote or EOF
	if data[advance] == '"' {
		advance++
		foundQuote := false
		for ; advance < len(data); advance++ {
			if data[advance] == '"' {
				foundQuote = true
				break
			}
			token = append(token, data[advance])
		}
		if !foundQuote {
			if atEOF {
				err = errMissingEndQuote
			} else {
				advance = 0
				token = nil
			}
			return
		}
		advance++
	}

	// Read until whitespace
	for ; advance < len(data) && !isSpace(data[advance]); advance++ {
		token = append(token, data[advance])
	}

	return
}

type responseReader struct {
	scanner *bufio.Scanner
}

func (r *responseReader) next() bool {
	if !r.scanner.Scan() {
		return false
	}
	return len(r.scanner.Text()) != 0
}

func (r *responseReader) isKeyword(v string) bool {
	return strings.ToLower(r.scanner.Text()) == v
}

func (r *responseReader) expectKeyword(v string) bool {
	return r.next() && r.isKeyword(v)
}

type listReader struct {
	responseReader
	variables map[string]string
}

func (l *listReader) parse() error {
	if !l.expectKeyword("begin") || !l.expectKeyword("list") || !l.next() {
		return errBeginListMissing
	}
	for l.next() {
		if l.isKeyword("end") {
			if l.expectKeyword("list") &&
				l.expectKeyword("var") &&
				l.next() {
				return nil
			}
			return errUnexpectedEof
		}
		if !l.isKeyword("var") {
			return errVarExpected
		}
		if !l.next() {
			return errUnexpectedEof
		}
		if !l.next() {
			return errVarNameMissing
		}
		varName := l.scanner.Text()
		if !l.next() {
			return errVarValueMissing
		}
		l.variables[varName] = l.scanner.Text()
	}
	return errUnexpectedEof
}

func newListReader(r io.Reader) (*listReader, error) {
	l := &listReader{
		responseReader: responseReader{
			scanner: bufio.NewScanner(r),
		},
		variables: map[string]string{},
	}
	l.scanner.Split(split)
	if err := l.parse(); err != nil {
		return nil, err
	}
	return l, nil
}
