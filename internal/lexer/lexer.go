package lexer

import "fmt"

type Lexer struct {
	input   string
	pos     int  // curr
	readPos int  // next
	ch      byte // curr char
	line    int
	col     int
}

func New(input string) *Lexer {
	l := &Lexer{input: input, line: 1, col: 0}
	l.advance()
	return l
}

func (l *Lexer) NextToken() Token {
	l.skipWhitespace()
	l.skipComments()

	tok := Token{Line: l.line, Col: l.col}

	switch {
	case l.ch == 0:
		tok.Type = EOF

	// Punctuation
	case l.ch == '(':
		tok = l.makeToken(LPAREN, "(")
	case l.ch == ')':
		tok = l.makeToken(RPAREN, ")")
	case l.ch == '{':
		tok = l.makeToken(LBRACE, "{")
	case l.ch == '}':
		tok = l.makeToken(RBRACE, "}")
	case l.ch == ',':
		tok = l.makeToken(COMMA, ",")
	case l.ch == ':':
		tok = l.makeToken(COLON, ":")
	case l.ch == ';':
		tok = l.makeToken(SEMICOLON, ";")
	case l.ch == '?':
		tok = l.makeToken(QUESTION, "?")
	case l.ch == '%':
		tok = l.makeToken(PERCENT, "%")
	case l.ch == '_':
		tok = l.makeToken(UNDERSCORE, "_")
	case l.ch == '|':
		if l.peek() == '|' {
			l.advance()
			tok = l.makeToken(OR, "||")
		} else {
			tok = l.makeToken(PIPE, "|")
		}
	case l.ch == '&':
		if l.peek() == '&' {
			l.advance()
			tok = l.makeToken(AND, "&&")
		} else {
			tok = l.makeToken(ILLEGAL, string(l.ch))
		}
	case l.ch == '!':
		if l.peek() == '=' {
			l.advance()
			tok = l.makeToken(NEQ, "!=")
		} else {
			tok = l.makeToken(NOT, "!")
		}
	case l.ch == '=':
		if l.peek() == '=' {
			l.advance()
			tok = l.makeToken(EQ, "==")
		} else {
			tok = l.makeToken(ASSIGN, "=")
		}
	case l.ch == '<':
		if l.peek() == '=' {
			l.advance()
			tok = l.makeToken(LTE, "<=")
		} else {
			tok = l.makeToken(LT, "<")
		}
	case l.ch == '>':
		if l.peek() == '=' {
			l.advance()
			tok = l.makeToken(GTE, ">=")
		} else {
			tok = l.makeToken(GT, ">")
		}
	case l.ch == '+':
		tok = l.makeToken(PLUS, "+")
	case l.ch == '-':
		if l.peek() == '>' {
			l.advance()
			tok = l.makeToken(ARROW, "->")
		} else {
			tok = l.makeToken(MINUS, "-")
		}
	case l.ch == '/':
		tok = l.makeToken(SLASH, "/")
	case l.ch == '*':
		tok = l.makeToken(STAR, "*")

	// Dot or destructuring
	case l.ch == '.':
		if l.peek() == '.' && l.peekN(2) == '.' {
			l.advance()
			l.advance()
			tok = l.makeToken(ELLIPSIS, "...")
		} else if l.peek() == '{' {
			tok = l.makeToken(DOT, ".")
		} else {
			tok = l.makeToken(DOT, ".")
		}

	// [ or []
	case l.ch == '[':
		if l.peek() == ']' {
			l.advance()
			tok = l.makeToken(LBRACKET, "[]")
		} else {
			tok = l.makeToken(LBRACKET, "[")
		}
	case l.ch == ']':
		tok = l.makeToken(RBRACKET, "]")

	// @ annotations
	case l.ch == '@':
		tok = l.readAnnotation()

	// String
	case l.ch == '\'':
		tok = l.readString()

	// Number
	case isDigit(l.ch):
		tok = l.readNumber()

	// Identifier or keyword
	case isLetter(l.ch):
		tok = l.readIdentOrKeyword()

	default:
		tok = l.makeToken(ILLEGAL, string(l.ch))
	}

	l.advance()
	return tok
}

// --- Readers ---

func (l *Lexer) readIdentOrKeyword() Token {
	start := l.pos
	startCol := l.col

	// fn* & fn
	if l.ch == 'f' && l.peek() == 'n' {
		l.advance()
		if l.peek() == '*' {
			l.advance()
			return Token{Type: FN_STAR, Literal: "fn*", Line: l.line, Col: startCol}
		}
		if l.peek() == '<' {
			l.advance()
			return Token{Type: FN_ANGLE, Literal: "fn<", Line: l.line, Col: startCol}
		}
	}

	for isLetter(l.ch) || isDigit(l.ch) {
		if l.peek() == 0 || (!isLetter(l.peek()) && !isDigit(l.peek())) {
			break
		}
		l.advance()
	}

	literal := l.input[start : l.pos+1]
	typ := lookupKeyword(literal)
	return Token{Type: typ, Literal: literal, Line: l.line, Col: startCol}
}

func (l *Lexer) readString() Token {
	startCol := l.col
	l.advance() // skip opening '
	start := l.pos

	for l.ch != '\'' && l.ch != 0 {
		if l.ch == '\n' {
			l.line++
			l.col = 0
		}
		l.advance()
	}

	literal := l.input[start:l.pos]
	return Token{Type: STRING, Literal: literal, Line: l.line, Col: startCol}
}

func (l *Lexer) readNumber() Token {
	start := l.pos
	startCol := l.col
	isFloat := false

	for isDigit(l.peek()) {
		l.advance()
	}

	if l.peek() == '.' && isDigit(l.peekN(2)) {
		isFloat = true
		l.advance() // .
		for isDigit(l.peek()) {
			l.advance()
		}
	}

	literal := l.input[start : l.pos+1]
	typ := INT
	if isFloat {
		typ = FLOAT
	}
	return Token{Type: typ, Literal: literal, Line: l.line, Col: startCol}
}

func (l *Lexer) readAnnotation() Token {
	startCol := l.col
	l.advance() // skip @
	start := l.pos

	for isLetter(l.ch) || isDigit(l.peek()) {
		if !isLetter(l.peek()) {
			break
		}
		l.advance()
	}

	literal := "@" + l.input[start:l.pos+1]
	switch literal {
	case "@serializable":
		return Token{Type: AT_SERIALIZABLE, Literal: literal, Line: l.line, Col: startCol}
	case "@builder":
		return Token{Type: AT_BUILDER, Literal: literal, Line: l.line, Col: startCol}
	default:
		return Token{Type: ILLEGAL, Literal: literal, Line: l.line, Col: startCol}
	}
}

// --- Helpers ---

func (l *Lexer) advance() {
	if l.readPos >= len(l.input) {
		l.ch = 0
	} else {
		l.ch = l.input[l.readPos]
	}
	l.pos = l.readPos
	l.readPos++
	l.col++
}

func (l *Lexer) peek() byte {
	if l.readPos >= len(l.input) {
		return 0
	}
	return l.input[l.readPos]
}

func (l *Lexer) peekN(n int) byte {
	pos := l.pos + n
	if pos >= len(l.input) {
		return 0
	}
	return l.input[pos]
}

func (l *Lexer) makeToken(typ TokenType, literal string) Token {
	return Token{Type: typ, Literal: literal, Line: l.line, Col: l.col}
}

func (l *Lexer) skipWhitespace() {
	for l.ch == ' ' || l.ch == '\t' || l.ch == '\r' || l.ch == '\n' {
		if l.ch == '\n' {
			l.line++
			l.col = 0
		}
		l.advance()
	}
}

func (l *Lexer) skipComments() {
	if l.ch == '/' && l.peek() == '/' {
		for l.ch != '\n' && l.ch != 0 {
			l.advance()
		}
		l.skipWhitespace()
	}
}

func (l *Lexer) Tokenize() ([]Token, error) {
	var tokens []Token
	for {
		tok := l.NextToken()
		if tok.Type == ILLEGAL {
			return nil, fmt.Errorf("illegal token %q at line %d col %d", tok.Literal, tok.Line, tok.Col)
		}
		tokens = append(tokens, tok)
		if tok.Type == EOF {
			break
		}
	}
	return tokens, nil
}

func lookupKeyword(ident string) TokenType {
	keywords := map[string]TokenType{
		"fn":        FN,
		"async":     ASYNC,
		"mut":       MUT,
		"object":    OBJECT,
		"prototype": PROTOTYPE,
		"interface": INTERFACE,
		"enum":      ENUM,
		"match":     MATCH,
		"import":    IMPORT,
		"export":    EXPORT,
		"from":      FROM,
		"as":        AS,
		"in":        IN,
		"super":     SUPER,
		"this":      THIS,
		"yield":     YIELD,
		"if":        IF,
		"else":      ELSE,
		"for":       FOR,
		"await":     AWAIT,
		"return":    RETURN,
		"try":       TRY,
		"Some":      SOME,
		"None":      NONE,
		"Ok":        OK,
		"Err":       ERR,
		"number":    TYPE_NUMBER,
		"string":    TYPE_STRING,
		"bool":      TYPE_BOOL,
		"void":      TYPE_VOID,
		"never":     TYPE_NEVER,
		"unknown":   TYPE_UNKNOWN,
		"Option":    TYPE_OPTION,
		"Result":    TYPE_RESULT,
		"Promise":   TYPE_PROMISE,
		"unpack":    UNPACK,
	}
	if typ, ok := keywords[ident]; ok {
		return typ
	}
	return IDENT
}

// --- Character helpers ---

func isLetter(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || ch == '_'
}

func isDigit(ch byte) bool {
	return ch >= '0' && ch <= '9'
}
