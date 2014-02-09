// Copyright 2014 Frustra. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package bbcode

import (
	"bytes"
	"errors"
	"regexp"
	"strings"
	"unicode"
)

type lexerState int

const (
	INIT_STATE lexerState = iota
	TAG_START_STATE
	TAG_ARGS_STATE
	ARG_VALUE_STATE
)

// Abuse of the generated types to keep parser state in the lexer
type lexer struct {
	str        []byte
	state      lexerState
	tagsOpened int
	buffer     bytes.Buffer
	err        error
}

var (
	tags       = []string{"url", "img", "b", "i", "u", "s", "quote", "code"}
	tagRegexps []*regexp.Regexp
	idRegexp   = regexp.MustCompile(`^[A-Za-z0-9_]+`)
	textRegexp = regexp.MustCompile(`^(.+?)[ \]]`)
)

func init() {
	for _, tag := range tags {
		r := regexp.MustCompile(`(?i)^\[/?[ \t]*` + tag + `[\]= \t]`)
		tagRegexps = append(tagRegexps, r)
	}
}

func newLexer(str string) *lexer {
	return &lexer{
		str: []byte(str),
	}
}

func (l *lexer) Lex(lval *yySymType) int {
	if len(l.str) <= 0 {
		if l.tagsOpened > 0 {
			l.tagsOpened--
			return MISSING_CLOSING
		} else {
			return 0
		}
	}
	var c byte = l.str[0]

	switch l.state {
	case TAG_START_STATE:
		str := strings.ToLower(string(l.str))
		for _, tag := range tags {
			if strings.HasPrefix(str, tag) {
				lval.str = tag
				l.str = l.str[len(tag):]
				l.state = TAG_ARGS_STATE
				return ID
			}
		}
	case TAG_ARGS_STATE:
		for unicode.IsSpace(rune(c)) {
			l.str = l.str[1:]
			c = l.str[0]
		}
		switch {
		case c == ']':
			l.str = l.str[1:]
			l.state = INIT_STATE
			return int(c)
		case c == '=':
			l.str = l.str[1:]
			l.state = ARG_VALUE_STATE
			return int(c)
		default:
			match := idRegexp.Find(l.str)
			if match != nil {
				lval.str = string(match)
				l.str = l.str[len(match):]
				return ID
			}
		}
	case ARG_VALUE_STATE:
		for unicode.IsSpace(rune(c)) {
			l.str = l.str[1:]
			c = l.str[0]
		}
		switch {
		case c == '"' || c == '\'':
			return 0 //l.LexQuotedString(c, lval)
		}
		matches := textRegexp.FindSubmatch(l.str)
		if matches != nil && len(matches) > 1 && len(matches[1]) > 0 {
			lval.str = string(matches[1])
			l.str = l.str[len(matches[1]):]
			l.state = TAG_ARGS_STATE
			return TEXT
		}
	case INIT_STATE:
		if c == '\n' {
			l.str = l.str[1:]
			return NEWLINE
		}
		if c == '[' {
			for _, r := range tagRegexps {
				if r.Match(l.str) {
					l.str = l.str[1:]
					l.state = TAG_START_STATE
					if l.str[0] == '/' {
						l.str = l.str[1:]
						if l.tagsOpened <= 0 {
							return MISSING_OPENING
						} else {
							l.tagsOpened--
							return CLOSING_TAG_OPENING
						}
					} else {
						l.tagsOpened++
						return int(c)
					}
				}
			}
		}
		offset := 1
		for offset < len(l.str) {
			curr := l.str[offset]
			if curr == '[' || curr == '\n' {
				break
			}
			offset++
		}
		lval.str = string(l.str[0:offset])
		l.str = l.str[offset:]
		return TEXT
	}
	return TEXT
}

func (l *lexer) Error(s string) {
	l.err = errors.New(s)
}
