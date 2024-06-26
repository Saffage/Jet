package scanner

import (
	"errors"

	"github.com/saffage/jet/config"
	"github.com/saffage/jet/token"
)

type Flags int

const (
	SkipWhitespace Flags = 1 << iota
	SkipIllegal
	SkipComments

	NoFlags      Flags = 0
	DefaultFlags Flags = NoFlags
)

func Scan(buffer []byte, fileid config.FileID, flags Flags) ([]token.Token, error) {
	s := New(buffer, fileid, flags)
	return s.AllTokens(), errors.Join(s.errors...)
}

func MustScan(buffer []byte, fileid config.FileID, flags Flags) []token.Token {
	tokens, err := Scan(buffer, fileid, flags)
	if err != nil {
		panic(err)
	}
	return tokens
}
