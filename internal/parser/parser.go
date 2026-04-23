package parser

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/alehbelskidev/barescript/internal/lexer"
)

type Parser struct {
	tokens          []lexer.Token
	pos             int
	errors          []string
	noTrailingClose bool
}

func New(tokens []lexer.Token) *Parser {
	return &Parser{tokens: tokens, pos: 0}
}

func (p *Parser) Errors() []string {
	return p.errors
}

func (p *Parser) Parse() *Program {
	program := &Program{}
	for !p.isEOF() {
		stmt := p.parseStatement()
		if stmt != nil {
			program.Statements = append(program.Statements, stmt)
		}
	}
	return program
}

// ─── Helpers ───────────────────────────────────────────────────────────────

func (p *Parser) cur() lexer.Token {
	if p.pos >= len(p.tokens) {
		return lexer.Token{Type: lexer.EOF}
	}
	return p.tokens[p.pos]
}

func (p *Parser) peek() lexer.Token {
	if p.pos+1 >= len(p.tokens) {
		return lexer.Token{Type: lexer.EOF}
	}
	return p.tokens[p.pos+1]
}

func (p *Parser) advance() lexer.Token {
	tok := p.cur()
	p.pos++
	return tok
}

func (p *Parser) expect(typ lexer.TokenType) (lexer.Token, bool) {
	if p.cur().Type != typ {
		p.errorf("expected %v, got %q at line %d col %d",
			typ, p.cur().Literal, p.cur().Line, p.cur().Col)
		return p.cur(), false
	}
	return p.advance(), true
}

func (p *Parser) isEOF() bool {
	return p.cur().Type == lexer.EOF
}

func (p *Parser) errorf(format string, args ...any) {
	p.errors = append(p.errors, fmt.Sprintf(format, args...))
}

func (p *Parser) skipSemicolons() {
	for p.cur().Type == lexer.SEMICOLON {
		p.advance()
	}
}

// ─── Statements ────────────────────────────────────────────────────────────

func (p *Parser) parseStatement() Statement {
	p.skipSemicolons()

	switch p.cur().Type {
	case lexer.EXPORT:
		return p.parseExport()
	case lexer.FN, lexer.ASYNC, lexer.FN_STAR, lexer.FN_ANGLE:
		return p.parseFnDecl(false)
	case lexer.MUT:
		return p.parseVarDecl()
	case lexer.OBJECT:
		return p.parseObjectDecl(false)
	case lexer.PROTOTYPE:
		return p.parsePrototypeDecl(false)
	case lexer.INTERFACE:
		return p.parseInterfaceDecl(false)
	case lexer.ENUM:
		return p.parseEnumDecl(false)
	case lexer.IMPORT:
		return p.parseImportDecl()
	case lexer.IF:
		return p.parseIfStmt()
	case lexer.FOR:
		return p.parseForStmt()
	case lexer.RETURN:
		p.advance()
		val := p.parseExpression()
		return &ExprStmt{Expr: val, IsImplicitReturn: true}
	case lexer.MATCH:
		expr := p.parseMatchExpr()
		return &ExprStmt{Expr: expr}
	case lexer.AT_SERIALIZABLE, lexer.AT_BUILDER:
		return p.parseAnnotated()
	case lexer.UNPACK:
		return p.parseDestructure()
	default:
		return p.parseExprOrVarDecl()
	}
}

func (p *Parser) parseExport() Statement {
	p.advance()
	switch p.cur().Type {
	case lexer.FN, lexer.ASYNC, lexer.FN_STAR, lexer.FN_ANGLE:
		decl := p.parseFnDecl(false).(*FnDecl)
		decl.Exported = true
		return decl
	case lexer.OBJECT:
		decl := p.parseObjectDecl(false).(*ObjectDecl)
		decl.Exported = true
		return decl
	case lexer.PROTOTYPE:
		decl := p.parsePrototypeDecl(false).(*PrototypeDecl)
		decl.Exported = true
		return decl
	case lexer.INTERFACE:
		decl := p.parseInterfaceDecl(false).(*InterfaceDecl)
		decl.Exported = true
		return decl
	case lexer.ENUM:
		decl := p.parseEnumDecl(false).(*EnumDecl)
		decl.Exported = true
		return decl
	default:
		p.errorf("unexpected token after export: %q", p.cur().Literal)
		return nil
	}
}

func (p *Parser) parseAnnotated() Statement {
	var annotations []string
	for p.cur().Type == lexer.AT_SERIALIZABLE || p.cur().Type == lexer.AT_BUILDER {
		annotations = append(annotations, p.advance().Literal)
	}
	if p.cur().Type == lexer.OBJECT {
		decl := p.parseObjectDecl(false).(*ObjectDecl)
		decl.Annotations = annotations
		return decl
	}
	p.errorf("annotations only supported on object declarations")
	return nil
}

// ─── Var Decl ──────────────────────────────────────────────────────────────

func (p *Parser) parseVarDecl() Statement {
	mutable := false
	if p.cur().Type == lexer.MUT {
		mutable = true
		p.advance()
	}

	name, ok := p.expect(lexer.IDENT)
	if !ok {
		return nil
	}

	var typeExpr *TypeExpr
	var value Expression

	if p.cur().Type != lexer.ASSIGN && p.isTypeToken() {
		t := p.parseTypeExpr()
		typeExpr = &t
	}

	if p.cur().Type == lexer.ASSIGN {
		p.advance()
		value = p.parseExpression()
	}

	return &VarDecl{
		Name:    name.Literal,
		Mutable: mutable,
		Type:    typeExpr,
		Value:   value,
	}
}

func (p *Parser) parseExprOrVarDecl() Statement {
	if p.cur().Type == lexer.IDENT && p.peek().Type == lexer.ASSIGN {
		name := p.advance().Literal
		p.advance()
		value := p.parseExpression()
		return &VarDecl{Name: name, Mutable: false, Value: value}
	}

	expr := p.parseExpression()
	if expr == nil {
		p.advance()
		return nil
	}
	return &ExprStmt{Expr: expr}
}

func (p *Parser) parseDestructure() Statement {
	p.advance() // unpack
	p.expect(lexer.LBRACE)
	var fields []string
	for p.cur().Type != lexer.RBRACE && !p.isEOF() {
		name, ok := p.expect(lexer.IDENT)
		if !ok {
			break
		}
		fields = append(fields, name.Literal)
		if p.cur().Type == lexer.COMMA {
			p.advance()
		}
	}
	p.expect(lexer.RBRACE)
	p.expect(lexer.ASSIGN)
	value := p.parseExpression()
	return &Destructure{Fields: fields, Value: value}
}

// ─── Functions ─────────────────────────────────────────────────────────────

func (p *Parser) parseFnDecl(_ bool) Statement {
	isAsync := false
	isGenerator := false
	typeParam := ""

	if p.cur().Type == lexer.ASYNC {
		isAsync = true
		p.advance()
	}

	switch p.cur().Type {
	case lexer.FN_STAR:
		isGenerator = true
		p.advance()
	case lexer.FN_ANGLE:
		p.advance()
		t, ok := p.expect(lexer.IDENT)
		if ok {
			typeParam = t.Literal
		}
		p.expect(lexer.GT)
	case lexer.FN:
		p.advance()
	default:
		p.errorf("expected fn, got %q", p.cur().Literal)
		return nil
	}

	name, ok := p.expect(lexer.IDENT)
	if !ok {
		return nil
	}

	params := p.parseParams()
	returnType := p.parseOptionalType()
	body := p.parseBlock()

	return &FnDecl{
		Name:       name.Literal,
		Async:      isAsync,
		Generator:  isGenerator,
		TypeParam:  typeParam,
		Params:     params,
		ReturnType: returnType,
		Body:       body,
	}
}

func (p *Parser) parseParams() []Param {
	p.expect(lexer.LPAREN)
	var params []Param
	for p.cur().Type != lexer.RPAREN && !p.isEOF() {
		variadic := false
		if p.cur().Type == lexer.ELLIPSIS {
			variadic = true
			p.advance()
		}
		name, ok := p.expect(lexer.IDENT)
		if !ok {
			break
		}
		var typ TypeExpr
		if p.isTypeToken() {
			typ = p.parseTypeExpr()
		}
		params = append(params, Param{
			Name:     name.Literal,
			Type:     typ,
			Variadic: variadic,
		})
		if p.cur().Type == lexer.COMMA {
			p.advance()
		} else {
			break
		}
	}
	p.expect(lexer.RPAREN)
	return params
}

func (p *Parser) parseOptionalType() *TypeExpr {
	if p.isTypeToken() {
		t := p.parseTypeExpr()
		return &t
	}
	return nil
}

func (p *Parser) isTypeToken() bool {
	switch p.cur().Type {
	case lexer.TYPE_NUMBER, lexer.TYPE_STRING, lexer.TYPE_BOOL,
		lexer.TYPE_VOID, lexer.TYPE_NEVER, lexer.TYPE_UNKNOWN,
		lexer.TYPE_OPTION, lexer.TYPE_RESULT, lexer.TYPE_PROMISE,
		lexer.LBRACKET:
		return true
	case lexer.IDENT:
		return len(p.cur().Literal) > 0 &&
			p.cur().Literal[0] >= 'A' && p.cur().Literal[0] <= 'Z'
	}
	return false
}

func (p *Parser) parseTypeExpr() TypeExpr {
	isArray := false
	if p.cur().Type == lexer.LBRACKET {
		isArray = true
		p.advance()
	}

	var name string
	switch p.cur().Type {
	case lexer.TYPE_NUMBER:
		name = "number"
	case lexer.TYPE_STRING:
		name = "string"
	case lexer.TYPE_BOOL:
		name = "bool"
	case lexer.TYPE_VOID:
		name = "void"
	case lexer.TYPE_NEVER:
		name = "never"
	case lexer.TYPE_UNKNOWN:
		name = "unknown"
	case lexer.TYPE_OPTION:
		name = "Option"
	case lexer.TYPE_RESULT:
		name = "Result"
	case lexer.TYPE_PROMISE:
		name = "Promise"
	case lexer.IDENT:
		name = p.cur().Literal
	default:
		p.errorf("expected type, got %q", p.cur().Literal)
		return TypeExpr{}
	}
	p.advance()

	var params []TypeExpr
	if p.cur().Type == lexer.LT {
		p.advance()
		for p.cur().Type != lexer.GT && !p.isEOF() {
			params = append(params, p.parseTypeExpr())
			if p.cur().Type == lexer.COMMA {
				p.advance()
			}
		}
		p.expect(lexer.GT)
	}

	nullable := false
	if p.cur().Type == lexer.QUESTION {
		nullable = true
		p.advance()
	}

	return TypeExpr{
		Name:     name,
		Nullable: nullable,
		Array:    isArray,
		Params:   params,
	}
}

func (p *Parser) parseBlock() []Statement {
	p.expect(lexer.LBRACE)
	var stmts []Statement
	for p.cur().Type != lexer.RBRACE && !p.isEOF() {
		stmt := p.parseStatement()
		if stmt != nil {
			stmts = append(stmts, stmt)
		}
	}
	p.expect(lexer.RBRACE)

	if len(stmts) > 0 {
		if es, ok := stmts[len(stmts)-1].(*ExprStmt); ok {
			es.IsImplicitReturn = true
		}
	}

	return stmts
}

// ─── Object ────────────────────────────────────────────────────────────────

func (p *Parser) parseObjectDecl(exported bool) Statement {
	p.expect(lexer.OBJECT)
	name, ok := p.expect(lexer.IDENT)
	if !ok {
		return nil
	}

	typeParam := ""
	if p.cur().Type == lexer.LT {
		p.advance()
		t, _ := p.expect(lexer.IDENT)
		typeParam = t.Literal
		p.expect(lexer.GT)
	}

	parent := ""
	if p.cur().Type == lexer.COLON {
		p.advance()
		par, ok := p.expect(lexer.IDENT)
		if ok {
			parent = par.Literal
		}
	}

	p.expect(lexer.LBRACE)

	var fields []Field
	var methods []FnDecl
	var initFn *FnDecl

	for p.cur().Type != lexer.RBRACE && !p.isEOF() {
		p.skipSemicolons()
		if p.cur().Type == lexer.FN || p.cur().Type == lexer.ASYNC {
			fn := p.parseFnDecl(true).(*FnDecl)
			if fn.Name == "init" {
				initFn = fn
			} else {
				methods = append(methods, *fn)
			}
		} else if p.cur().Type == lexer.IDENT {
			fieldName := p.advance().Literal
			fieldType := p.parseTypeExpr()
			fields = append(fields, Field{Name: fieldName, Type: fieldType})
		} else {
			p.advance()
		}
	}
	p.expect(lexer.RBRACE)

	return &ObjectDecl{
		Name:      name.Literal,
		Parent:    parent,
		TypeParam: typeParam,
		Fields:    fields,
		Methods:   methods,
		Init:      initFn,
		Exported:  exported,
	}
}

// ─── Prototype ─────────────────────────────────────────────────────────────

func (p *Parser) parsePrototypeDecl(exported bool) Statement {
	p.expect(lexer.PROTOTYPE)
	name, ok := p.expect(lexer.IDENT)
	if !ok {
		return nil
	}

	var interfaces []string
	if p.cur().Type == lexer.COLON {
		p.advance()
		for {
			iface, ok := p.expect(lexer.IDENT)
			if !ok {
				break
			}
			interfaces = append(interfaces, iface.Literal)
			if p.cur().Type != lexer.COMMA {
				break
			}
			p.advance()
		}
	}

	p.expect(lexer.LBRACE)
	var methods []FnDecl
	for p.cur().Type != lexer.RBRACE && !p.isEOF() {
		p.skipSemicolons()
		if p.cur().Type == lexer.FN || p.cur().Type == lexer.ASYNC {
			fn := p.parseFnDecl(false).(*FnDecl)
			methods = append(methods, *fn)
		} else {
			p.advance()
		}
	}
	p.expect(lexer.RBRACE)

	return &PrototypeDecl{
		Name:       name.Literal,
		Interfaces: interfaces,
		Methods:    methods,
		Exported:   exported,
	}
}

// ─── Interface ─────────────────────────────────────────────────────────────

func (p *Parser) parseInterfaceDecl(exported bool) Statement {
	p.expect(lexer.INTERFACE)
	name, ok := p.expect(lexer.IDENT)
	if !ok {
		return nil
	}

	p.expect(lexer.LBRACE)
	var methods []MethodSig
	for p.cur().Type != lexer.RBRACE && !p.isEOF() {
		p.skipSemicolons()
		if p.cur().Type == lexer.IDENT {
			methodName := p.advance().Literal
			params := p.parseParams()
			returnType := p.parseOptionalType()
			methods = append(methods, MethodSig{
				Name:       methodName,
				Params:     params,
				ReturnType: returnType,
			})
		} else {
			p.advance()
		}
	}
	p.expect(lexer.RBRACE)

	return &InterfaceDecl{
		Name:     name.Literal,
		Methods:  methods,
		Exported: exported,
	}
}

// ─── Enum ──────────────────────────────────────────────────────────────────

func (p *Parser) parseEnumDecl(exported bool) Statement {
	p.expect(lexer.ENUM)
	name, ok := p.expect(lexer.IDENT)
	if !ok {
		return nil
	}

	p.expect(lexer.LBRACE)
	var variants []EnumVariant
	for p.cur().Type != lexer.RBRACE && !p.isEOF() {
		p.skipSemicolons()
		if p.cur().Type != lexer.IDENT {
			p.advance()
			continue
		}
		variantName := p.advance().Literal
		var payload *Param
		if p.cur().Type == lexer.LPAREN {
			p.advance()
			pname, _ := p.expect(lexer.IDENT)
			ptype := p.parseTypeExpr()
			payload = &Param{Name: pname.Literal, Type: ptype}
			p.expect(lexer.RPAREN)
		}
		variants = append(variants, EnumVariant{Name: variantName, Payload: payload})
	}
	p.expect(lexer.RBRACE)

	return &EnumDecl{
		Name:     name.Literal,
		Variants: variants,
		Exported: exported,
	}
}

// ─── Match ─────────────────────────────────────────────────────────────────

func (p *Parser) parseMatchExpr() Expression {
	p.expect(lexer.MATCH)

	// trailing closure should be "turned off" while reading subject
	// `match try foo(x) {` — `{` will be recognized as trailing closure otherwise :(
	p.noTrailingClose = true
	subject := p.parseExpression()
	p.noTrailingClose = false

	p.expect(lexer.LBRACE)

	var arms []MatchArm
	for p.cur().Type != lexer.RBRACE && !p.isEOF() {
		p.skipSemicolons()
		if p.cur().Type == lexer.RBRACE {
			break
		}
		pattern := p.parseMatchPattern()
		p.expect(lexer.ARROW)

		var body Expression
		if p.cur().Type == lexer.LBRACE {
			stmts := p.parseBlock()
			body = &LambdaExpr{Body: stmts}
		} else {
			p.noTrailingClose = true
			body = p.parseExpression()
			p.noTrailingClose = false
		}

		arms = append(arms, MatchArm{Pattern: pattern, Body: body})
	}
	p.expect(lexer.RBRACE)

	return &MatchExpr{Subject: subject, Arms: arms}
}

func (p *Parser) parseMatchPattern() Expression {
	if p.cur().Type == lexer.UNDERSCORE {
		p.advance()
		return &WildcardPattern{}
	}

	if p.cur().Type == lexer.NONE {
		p.advance()
		return &EnumPattern{Variant: "None"}
	}

	name := p.advance().Literal
	if p.cur().Type == lexer.DOT {
		p.advance()
		variant, _ := p.expect(lexer.IDENT)
		name = name + "." + variant.Literal
	}

	binding := ""
	if p.cur().Type == lexer.LPAREN {
		p.advance()
		b, _ := p.expect(lexer.IDENT)
		binding = b.Literal
		p.expect(lexer.RPAREN)
	}

	return &EnumPattern{Variant: name, Binding: binding}
}

// ─── If / For ──────────────────────────────────────────────────────────────

func (p *Parser) parseIfStmt() Statement {
	p.advance() // if

	if p.cur().Type == lexer.IDENT && p.peek().Type == lexer.ASSIGN {
		name := p.advance().Literal
		p.advance() // =
		p.noTrailingClose = true
		value := p.parseExpression()
		p.noTrailingClose = false
		binding := &VarDecl{Name: name, Value: value}
		then := p.parseBlock()
		var els []Statement
		if p.cur().Type == lexer.ELSE {
			p.advance()
			els = p.parseBlock()
		}
		return &IfStmt{Binding: binding, Then: then, Else: els}
	}

	p.noTrailingClose = true
	cond := p.parseExpression()
	p.noTrailingClose = false
	then := p.parseBlock()
	var els []Statement
	if p.cur().Type == lexer.ELSE {
		p.advance()
		els = p.parseBlock()
	}
	return &IfStmt{Condition: cond, Then: then, Else: els}
}

func (p *Parser) parseForStmt() Statement {
	p.advance() // for
	binding, ok := p.expect(lexer.IDENT)
	if !ok {
		return nil
	}
	p.expect(lexer.IN)
	p.noTrailingClose = true
	iter := p.parseExpression()
	p.noTrailingClose = false
	body := p.parseBlock()
	return &ForStmt{Binding: binding.Literal, Iter: iter, Body: body}
}

// ─── Import ────────────────────────────────────────────────────────────────

func (p *Parser) parseImportDecl() Statement {
	p.expect(lexer.IMPORT)
	p.expect(lexer.LPAREN)

	var specifiers []ImportSpecifier

	for p.cur().Type != lexer.RPAREN && !p.isEOF() {
		p.skipSemicolons()
		if p.cur().Type == lexer.RPAREN {
			break
		}
		spec := ImportSpecifier{}

		if p.cur().Type == lexer.STAR {
			p.advance()
			p.expect(lexer.AS)
			alias, _ := p.expect(lexer.IDENT)
			spec.Star = true
			spec.Alias = alias.Literal
		} else if p.cur().Type == lexer.LBRACE {
			p.advance()
			name, _ := p.expect(lexer.IDENT)
			spec.Name = name.Literal
			if p.cur().Type == lexer.AS {
				p.advance()
				alias, _ := p.expect(lexer.IDENT)
				spec.Alias = alias.Literal
			}
			p.expect(lexer.RBRACE)
		} else if p.cur().Type == lexer.DOT {
			p.advance()
			p.expect(lexer.LBRACE)
			name, _ := p.expect(lexer.IDENT)
			spec.Name = name.Literal
			p.expect(lexer.RBRACE)
		} else {
			name, _ := p.expect(lexer.IDENT)
			spec.Name = name.Literal
		}

		p.expect(lexer.FROM)
		spec.Path = p.parseImportPath()
		specifiers = append(specifiers, spec)
	}

	p.expect(lexer.RPAREN)
	return &ImportDecl{Specifiers: specifiers}
}

func (p *Parser) parseImportPath() string {
	var parts []string
	for !p.isEOF() {
		tok := p.cur()
		if tok.Type == lexer.IDENT || tok.Type == lexer.COLON ||
			tok.Type == lexer.SLASH || tok.Type == lexer.DOT {
			parts = append(parts, tok.Literal)
			p.advance()
		} else {
			break
		}
	}
	return strings.Join(parts, "")
}

// ─── Expressions ───────────────────────────────────────────────────────────

func (p *Parser) parseExpression() Expression {
	return p.parseComparison()
}

func (p *Parser) parseComparison() Expression {
	left := p.parseAdditive()
	for p.cur().Type == lexer.EQ || p.cur().Type == lexer.NEQ ||
		p.cur().Type == lexer.LT || p.cur().Type == lexer.GT ||
		p.cur().Type == lexer.LTE || p.cur().Type == lexer.GTE {
		op := p.advance().Literal
		right := p.parseAdditive()
		left = &BinaryExpr{Left: left, Op: op, Right: right}
	}
	return left
}

func (p *Parser) parseAdditive() Expression {
	left := p.parseMultiplicative()
	for p.cur().Type == lexer.PLUS || p.cur().Type == lexer.MINUS {
		op := p.advance().Literal
		right := p.parseMultiplicative()
		left = &BinaryExpr{Left: left, Op: op, Right: right}
	}
	return left
}

func (p *Parser) parseMultiplicative() Expression {
	left := p.parseUnary()
	for p.cur().Type == lexer.STAR || p.cur().Type == lexer.SLASH ||
		p.cur().Type == lexer.PERCENT {
		op := p.advance().Literal
		right := p.parseUnary()
		left = &BinaryExpr{Left: left, Op: op, Right: right}
	}
	return left
}

func (p *Parser) parseUnary() Expression {
	if p.cur().Type == lexer.NOT || p.cur().Type == lexer.MINUS {
		op := p.advance().Literal
		operand := p.parseUnary()
		return &UnaryExpr{Op: op, Operand: operand}
	}
	return p.parsePostfix()
}

func (p *Parser) parsePostfix() Expression {
	expr := p.parsePrimary()
	for {
		switch p.cur().Type {
		case lexer.DOT:
			p.advance()
			prop, ok := p.expect(lexer.IDENT)
			if !ok {
				return expr
			}
			expr = &MemberExpr{Object: expr, Property: prop.Literal}
		case lexer.LPAREN:
			expr = p.parseCallArgs(expr)
		case lexer.LBRACE:
			if !p.noTrailingClose && p.isTrailingClosure() {
				lambda := p.parseTrailingClosure()
				expr = &CallExpr{Callee: expr, Trailing: lambda}
			} else {
				return expr
			}
		default:
			return expr
		}
	}
}

func (p *Parser) isTrailingClosure() bool {
	if p.pos == 0 {
		return false
	}
	prev := p.tokens[p.pos-1]
	return prev.Type == lexer.RPAREN || prev.Type == lexer.IDENT
}

func (p *Parser) parseCallArgs(callee Expression) Expression {
	p.expect(lexer.LPAREN)
	var args []Expression
	for p.cur().Type != lexer.RPAREN && !p.isEOF() {
		if p.cur().Type == lexer.ELLIPSIS {
			p.advance()
			arg := p.parseExpression()
			args = append(args, &SpreadExpr{Value: arg})
		} else {
			args = append(args, p.parseExpression())
		}
		if p.cur().Type == lexer.COMMA {
			p.advance()
		}
	}
	p.expect(lexer.RPAREN)

	var trailing *LambdaExpr
	if !p.noTrailingClose && p.cur().Type == lexer.LBRACE && p.isTrailingClosure() {
		trailing = p.parseTrailingClosure()
	}

	return &CallExpr{Callee: callee, Args: args, Trailing: trailing}
}

func (p *Parser) parseTrailingClosure() *LambdaExpr {
	p.expect(lexer.LBRACE)
	var params []Param

	if p.cur().Type == lexer.IDENT {
		saved := p.pos
		var names []string
		for p.cur().Type == lexer.IDENT {
			names = append(names, p.advance().Literal)
			if p.cur().Type == lexer.COMMA {
				p.advance()
			}
		}
		if p.cur().Type == lexer.IN {
			p.advance()
			for _, name := range names {
				params = append(params, Param{Name: name})
			}
		} else {
			p.pos = saved
		}
	}

	var stmts []Statement
	for p.cur().Type != lexer.RBRACE && !p.isEOF() {
		stmt := p.parseStatement()
		if stmt != nil {
			stmts = append(stmts, stmt)
		}
	}
	p.expect(lexer.RBRACE)

	if len(stmts) > 0 {
		if es, ok := stmts[len(stmts)-1].(*ExprStmt); ok {
			es.IsImplicitReturn = true
		}
	}

	return &LambdaExpr{Params: params, Body: stmts, Trailing: true}
}

func (p *Parser) parsePrimary() Expression {
	switch p.cur().Type {
	case lexer.INT:
		val, _ := strconv.ParseInt(p.cur().Literal, 10, 64)
		tok := p.advance()
		return &IntLiteral{Value: val, Raw: tok.Literal}

	case lexer.FLOAT:
		val, _ := strconv.ParseFloat(p.cur().Literal, 64)
		tok := p.advance()
		return &FloatLiteral{Value: val, Raw: tok.Literal}

	case lexer.STRING:
		val := p.advance().Literal
		return p.parseStringInterpolation(val)

	case lexer.IDENT:
		return &Identifier{Name: p.advance().Literal}

	case lexer.TYPE_PROMISE, lexer.TYPE_OPTION, lexer.TYPE_RESULT,
		lexer.SOME, lexer.NONE, lexer.OK, lexer.ERR:
		return &Identifier{Name: p.advance().Literal}

	case lexer.LPAREN:
		if p.isLambda() {
			return p.parseLambda()
		}
		p.advance()
		expr := p.parseExpression()
		p.expect(lexer.RPAREN)
		return expr

	case lexer.LBRACKET:
		return p.parseArrayLiteral()

	case lexer.MATCH:
		return p.parseMatchExpr()

	case lexer.TRY:
		p.advance()
		expr := p.parseExpression()
		return &TryExpr{Expr: expr}

	case lexer.AWAIT:
		p.advance()
		expr := p.parseExpression()
		return &AwaitExpr{Expr: expr}

	case lexer.YIELD:
		p.advance()
		expr := p.parseExpression()
		return &YieldExpr{Value: expr}

	case lexer.THIS:
		p.advance()
		if p.cur().Type == lexer.LBRACE {
			return p.parseThisInit()
		}
		return &Identifier{Name: "this"}

	case lexer.SUPER:
		p.advance()
		args := []Expression{}
		if p.cur().Type == lexer.LPAREN {
			call := p.parseCallArgs(&Identifier{Name: "super"}).(*CallExpr)
			args = call.Args
		}
		return &SuperCall{Args: args}

	case lexer.ELLIPSIS:
		p.advance()
		expr := p.parseExpression()
		return &SpreadExpr{Value: expr}

	default:
		return nil
	}
}

func (p *Parser) parseThisInit() Expression {
	p.expect(lexer.LBRACE)
	fields := map[string]Expression{}
	for p.cur().Type != lexer.RBRACE && !p.isEOF() {
		key, ok := p.expect(lexer.IDENT)
		if !ok {
			break
		}
		p.expect(lexer.COLON)
		val := p.parseExpression()
		fields[key.Literal] = val
		if p.cur().Type == lexer.COMMA {
			p.advance()
		}
	}
	p.expect(lexer.RBRACE)
	return &ThisInit{Fields: fields}
}

func (p *Parser) isLambda() bool {
	saved := p.pos
	p.advance() // (
	depth := 1
	for depth > 0 && !p.isEOF() {
		if p.cur().Type == lexer.LPAREN {
			depth++
		}
		if p.cur().Type == lexer.RPAREN {
			depth--
		}
		p.advance()
	}
	result := p.cur().Type == lexer.LBRACE
	p.pos = saved
	return result
}

func (p *Parser) parseLambda() Expression {
	params := p.parseParams()
	body := p.parseBlock()
	return &LambdaExpr{Params: params, Body: body}
}

func (p *Parser) parseArrayLiteral() Expression {
	p.expect(lexer.LBRACKET)
	var elements []Expression
	for p.cur().Type != lexer.RBRACKET && !p.isEOF() {
		if p.cur().Type == lexer.ELLIPSIS {
			p.advance()
			elements = append(elements, &SpreadExpr{Value: p.parseExpression()})
		} else {
			elements = append(elements, p.parseExpression())
		}
		if p.cur().Type == lexer.COMMA {
			p.advance()
		}
	}
	p.expect(lexer.RBRACKET)
	return &ArrayLiteral{Elements: elements}
}

func (p *Parser) parseStringInterpolation(raw string) Expression {
	if !strings.Contains(raw, "#{") {
		return &StringLiteral{Value: raw}
	}

	var parts []Expression
	for len(raw) > 0 {
		idx := strings.Index(raw, "#{")
		if idx == -1 {
			parts = append(parts, &TextPart{Value: raw})
			break
		}
		if idx > 0 {
			parts = append(parts, &TextPart{Value: raw[:idx]})
		}
		raw = raw[idx+2:]
		end := strings.Index(raw, "}")
		if end == -1 {
			p.errorf("unclosed interpolation in string")
			break
		}
		expr := raw[:end]
		parts = append(parts, &Identifier{Name: strings.TrimSpace(expr)})
		raw = raw[end+1:]
	}

	return &InterpolatedString{Parts: parts}
}
