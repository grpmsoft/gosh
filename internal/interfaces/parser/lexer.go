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
	TokenRedirectIn               // <
	TokenRedirectOut              // >
	TokenRedirectAppend           // >>
	TokenRedirectErr              // 2>
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
	Value string
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

		// Check special characters
		switch {
		case l.match("|"):
			l.addToken(TokenPipe, "|")
		case l.match(">>"):
			l.addToken(TokenRedirectAppend, ">>")
		case l.match("2>"):
			l.addToken(TokenRedirectErr, "2>")
		case l.match(">"):
			l.addToken(TokenRedirectOut, ">")
		case l.match("<"):
			l.addToken(TokenRedirectIn, "<")
		case l.match("&&"):
			l.addToken(TokenAnd, "&&")
		case l.match("||"):
			l.addToken(TokenOr, "||")
		case l.match("&"):
			l.addToken(TokenBackground, "&")
		case l.match(";"):
			l.addToken(TokenSemicolon, ";")
		default:
			// Read word
			word := l.readWord()
			if word != "" {
				l.addToken(TokenWord, word)
			}
		}
	}

	l.addToken(TokenEOF, "")
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

// addToken adds a token to the list
func (l *Lexer) addToken(tokenType TokenType, value string) {
	l.tokens = append(l.tokens, Token{
		Type:  tokenType,
		Value: value,
		Pos:   l.pos,
	})
}

// Tokens returns the list of tokens
func (l *Lexer) Tokens() []Token {
	return l.tokens
}
