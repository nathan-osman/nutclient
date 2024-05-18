package nutclient

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"
)

var (
	errMissingEndQuote = errors.New("missing \"")
	errMissingValue    = errors.New("missing value")
	errUnexpectedEOF   = errors.New("unexpected EOF")
)

func isSpace(b byte) bool {
	return b == ' ' || b == '\t'
}

func split(data []byte, atEOF bool) (int, []byte, error) {

	var start = 0

	// Skip whitespace
	for ; start < len(data) && isSpace(data[start]); start++ {
	}

	var (
		end      = start
		isQuote  = false
		foundEnd = false
	)

	// Read token
	for ; end < len(data); end++ {
		if end == start && data[end] == '"' {
			start += 1
			isQuote = true
			continue
		}
		if isQuote && data[end] == '"' || !isQuote && isSpace(data[end]) {
			foundEnd = true
			break
		}
	}

	// If the token was not terminated, check to see if there is more
	// - if a regular token and EOF, return what's there (or stop if empty)
	// - if a regular token and not EOF, request more
	// - if a quote token and EOF, that's an error
	// - if a quote token and not EOF, request more
	if !foundEnd {
		if atEOF {
			if isQuote {
				return 0, nil, errMissingEndQuote
			}
			if start == end {
				return 0, nil, bufio.ErrFinalToken
			}
		} else {
			return start, nil, nil
		}
	}

	// Grab the slice to return and adjust the advance if there's a quote
	token := data[start:end]
	if isQuote {
		end += 1
	}

	return end, token, nil
}

func parseLine(v string) ([]string, error) {
	var (
		s     = bufio.NewScanner(strings.NewReader(v))
		lines []string
	)
	s.Split(split)
	for s.Scan() {
		if len(s.Text()) == 0 {
			break
		}
		lines = append(lines, s.Text())
	}
	if s.Err() != nil {
		return nil, s.Err()
	}
	if len(lines) == 0 {
		return nil, errMissingValue
	}
	return lines, nil
}

func trimPrefix(v, prefixes []string) ([]string, error) {
	var (
		numPrefixes = len(prefixes)
		errMissingX = fmt.Errorf("%s expected", strings.Join(prefixes, " "))
	)
	if len(v) < numPrefixes {
		return nil, errMissingX
	}
	for i := 0; i < numPrefixes; i++ {
		if !strings.EqualFold(v[i], prefixes[i]) {
			return nil, errMissingX
		}
	}
	return v[numPrefixes:], nil
}

type nutConn struct {
	rw      io.ReadWriter
	scanner *bufio.Scanner
}

func newNutConn(rw io.ReadWriter) *nutConn {
	s := bufio.NewScanner(rw)
	s.Split(bufio.ScanLines)
	return &nutConn{
		rw:      rw,
		scanner: s,
	}
}

func (n *nutConn) send(cmd, v string) ([]string, []string, error) {
	prefixes, err := parseLine(v)
	if err != nil {
		return nil, nil, err
	}
	var writeCmd string
	if v == "" {
		writeCmd = cmd
	} else {
		writeCmd = strings.Join([]string{cmd, v}, " ")
	}
	writeCmd += "\n"
	if _, err := n.rw.Write([]byte(writeCmd)); err != nil {
		return nil, nil, err
	}
	if !n.scanner.Scan() {
		return nil, nil, n.scanner.Err()
	}
	l, err := parseLine(n.scanner.Text())
	if err != nil {
		return nil, nil, err
	}
	if len(l) >= 2 && strings.ToLower(l[0]) == "err" {
		return nil, nil, fmt.Errorf("server returned %s", l[1])
	}
	return prefixes, l, nil
}

func (n *nutConn) runGet(v string) (string, error) {
	prefixes, l, err := n.send("GET", v)
	if err != nil {
		return "", err
	}
	t, err := trimPrefix(l, prefixes)
	if err != nil {
		return "", err
	}
	if len(t) == 0 {
		return "", errMissingValue
	}
	return t[0], nil
}

func (n *nutConn) runList(v string) ([][]string, error) {
	prefixes, l, err := n.send("LIST", v)
	if err != nil {
		return nil, err
	}
	if _, err := trimPrefix(l, append([]string{"begin", "list"}, prefixes...)); err != nil {
		return nil, err
	}
	values := [][]string{}
	for {
		if !n.scanner.Scan() {
			return nil, errUnexpectedEOF
		}
		l, err = parseLine(n.scanner.Text())
		if err != nil {
			return nil, err
		}
		if strings.ToLower(l[0]) == "end" {
			break
		}
		t, err := trimPrefix(l, prefixes)
		if err != nil {
			return nil, err
		}
		values = append(values, t)
	}
	if _, err := trimPrefix(l, append([]string{"end", "list"}, prefixes...)); err != nil {
		return nil, err
	}
	return values, nil
}

func (n *nutConn) runCmd(v string) error {
	_, l, err := n.send(v, "")
	if err != nil {
		return err
	}
	if len(l) < 1 || strings.ToLower(l[0]) != "ok" {
		return errUnexpectedEOF
	}
	return nil
}
