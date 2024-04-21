package nutclient

import (
	"bufio"
	"errors"
)

var (
	errMissingEndQuote = errors.New("missing \"")
)

func isSpace(b byte) bool {
	return b == ' ' || b == '\t'
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
