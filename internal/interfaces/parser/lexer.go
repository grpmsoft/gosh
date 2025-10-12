package parser

import (
	"strings"
	"unicode"
)

// TokenType represents the type of token
type TokenType int

const (
	TokenWord           TokenType = iota
	TokenPipe                     // |
	TokenRedirectIn               // N< (file descriptor input)
	TokenRedirectOut              // N> (file descriptor output)
	TokenRedirectAppend           // N>> (file descriptor append)
	TokenRedirectDup              // N>&M (file descriptor duplication)
	TokenBackground               // &
	TokenSemicolon                // ;
	TokenAnd                      // &&
	TokenOr                       // ||
	TokenEOF
	TokenError
)

// Token represents a lexical token
type Token struct {
	Type  TokenType
	Value string // For redirections: filename or "&N" for FD duplication
	FD    int    // File descriptor for redirections (0 for stdin, 1 for stdout, 2 for stderr, etc.)
	Pos   int
}

// Lexer performs lexical analysis of command line
type Lexer struct {
	input  string
	pos    int
	tokens []Token
}

// NewLexer creates a new lexer
func NewLexer(input string) *Lexer {
	return &Lexer{
		input:  input,
		pos:    0,
		tokens: make([]Token, 0),
	}
}

// Tokenize splits input string into tokens
func (l *Lexer) Tokenize() ([]Token, error) {
	for l.pos < len(l.input) {
		// Skip whitespace
		if unicode.IsSpace(rune(l.input[l.pos])) {
			l.pos++
			continue
		}

		// Check for file descriptor redirections (N>, N<, N>>, N>&M)
		if l.tryParseRedirection() {
			continue
		}

		// Check special characters
		switch {
		case l.match("|"):
			l.addToken(TokenPipe, "", -1)
		case l.match("&&"):
			l.addToken(TokenAnd, "", -1)
		case l.match("||"):
			l.addToken(TokenOr, "", -1)
		case l.match("&"):
			l.addToken(TokenBackground, "", -1)
		case l.match(";"):
			l.addToken(TokenSemicolon, "", -1)
		default:
			// Read word
			word := l.readWord()
			if word != "" {
				l.addToken(TokenWord, word, -1)
			}
		}
	}

	l.addToken(TokenEOF, "", -1)
	return l.tokens, nil
}

// match checks and consumes a substring
func (l *Lexer) match(s string) bool {
	if l.pos+len(s) > len(l.input) {
		return false
	}

	if l.input[l.pos:l.pos+len(s)] == s {
		l.pos += len(s)
		return true
	}

	return false
}

// readWord reads a word (until whitespace or special character)
func (l *Lexer) readWord() string {
	start := l.pos
	inQuotes := false
	inSingleQuotes := false
	escaped := false

	for l.pos < len(l.input) {
		ch := l.input[l.pos]

		// Handle escape sequences
		if escaped {
			escaped = false
			l.pos++
			continue
		}

		if ch == '\\' {
			escaped = true
			l.pos++
			continue
		}

		// Handle quotes
		if ch == '"' && !inSingleQuotes {
			inQuotes = !inQuotes
			l.pos++
			continue
		}

		if ch == '\'' && !inQuotes {
			inSingleQuotes = !inSingleQuotes
			l.pos++
			continue
		}

		// If inside quotes, continue
		if inQuotes || inSingleQuotes {
			l.pos++
			continue
		}

		// Check for special characters
		if unicode.IsSpace(rune(ch)) || ch == '|' || ch == '>' || ch == '<' || ch == '&' || ch == ';' {
			break
		}

		l.pos++
	}

	word := l.input[start:l.pos]
	// Remove quotes from word
	word = strings.ReplaceAll(word, "\"", "")
	word = strings.ReplaceAll(word, "'", "")
	return word
}

// tryParseRedirection tries to parse file descriptor redirections
// Supports: N<, N>, N>>, N>&M where N and M are digits
// Defaults: < = 0<, > = 1>, >> = 1>>
func (l *Lexer) tryParseRedirection() bool {
	start := l.pos
	fd := -1

	// Check if we have a digit followed by redirection
	if l.pos < len(l.input) && l.input[l.pos] >= '0' && l.input[l.pos] <= '9' {
		// Parse FD number
		fd = 0
		for l.pos < len(l.input) && l.input[l.pos] >= '0' && l.input[l.pos] <= '9' {
			fd = fd*10 + int(l.input[l.pos]-'0')
			l.pos++
		}
	}

	// Now check for redirection operators
	switch {
	case l.match(">>"):
		if fd == -1 {
			fd = 1 // default stdout
		}
		l.addToken(TokenRedirectAppend, "", fd)
		return true

	case l.match(">&"):
		// N>&M duplication
		if fd == -1 {
			fd = 1 // default stdout
		}
		l.addToken(TokenRedirectDup, "", fd)
		return true

	case l.match(">"):
		if fd == -1 {
			fd = 1 // default stdout
		}
		l.addToken(TokenRedirectOut, "", fd)
		return true

	case l.match("<"):
		if fd == -1 {
			fd = 0 // default stdin
		}
		l.addToken(TokenRedirectIn, "", fd)
		return true
	}

	// No redirection found, restore position
	l.pos = start
	return false
}

// addToken adds a token to the list
func (l *Lexer) addToken(tokenType TokenType, value string, fd int) {
	l.tokens = append(l.tokens, Token{
		Type:  tokenType,
		Value: value,
		FD:    fd,
		Pos:   l.pos,
	})
}

// Tokens returns the list of tokens
func (l *Lexer) Tokens() []Token {
	return l.tokens
}
