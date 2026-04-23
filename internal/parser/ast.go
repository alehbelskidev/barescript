package parser

import "fmt"

type Node interface {
	TokenLiteral() string
	String() string
}

type Statement interface {
	Node
	statementNode()
}

type Expression interface {
	Node
	expressionNode()
}

// ─── Program ───────────────────────────────────────────────────────────────

type Program struct {
	Statements []Statement
}

func (p *Program) TokenLiteral() string {
	if len(p.Statements) > 0 {
		return p.Statements[0].TokenLiteral()
	}
	return ""
}

func (p *Program) String() string {
	return fmt.Sprintf("Program(%d statements)", len(p.Statements))
}

// ─── Literals ──────────────────────────────────────────────────────────────

type IntLiteral struct {
	Value int64
	Raw   string
}

func (n *IntLiteral) expressionNode()      {}
func (n *IntLiteral) TokenLiteral() string { return n.Raw }
func (n *IntLiteral) String() string       { return n.Raw }

type FloatLiteral struct {
	Value float64
	Raw   string
}

func (n *FloatLiteral) expressionNode()      {}
func (n *FloatLiteral) TokenLiteral() string { return n.Raw }
func (n *FloatLiteral) String() string       { return n.Raw }

type StringLiteral struct {
	Value string
}

func (n *StringLiteral) expressionNode()      {}
func (n *StringLiteral) TokenLiteral() string { return n.Value }
func (n *StringLiteral) String() string       { return fmt.Sprintf("'%s'", n.Value) }

type InterpolatedString struct {
	Parts []Expression
}

func (n *InterpolatedString) expressionNode()      {}
func (n *InterpolatedString) TokenLiteral() string { return "interpolated" }
func (n *InterpolatedString) String() string       { return "interpolated_string" }

type TextPart struct {
	Value string
}

func (n *TextPart) expressionNode()      {}
func (n *TextPart) TokenLiteral() string { return n.Value }
func (n *TextPart) String() string       { return n.Value }

type BoolLiteral struct {
	Value bool
}

func (n *BoolLiteral) expressionNode()      {}
func (n *BoolLiteral) TokenLiteral() string { return fmt.Sprintf("%v", n.Value) }
func (n *BoolLiteral) String() string       { return fmt.Sprintf("%v", n.Value) }

// ─── Identifiers & Types ───────────────────────────────────────────────────

type Identifier struct {
	Name string
}

func (n *Identifier) expressionNode()      {}
func (n *Identifier) statementNode()       {}
func (n *Identifier) TokenLiteral() string { return n.Name }
func (n *Identifier) String() string       { return n.Name }

type TypeExpr struct {
	Name     string
	Nullable bool
	Array    bool
	Params   []TypeExpr
}

func (n *TypeExpr) String() string {
	s := n.Name
	if len(n.Params) > 0 {
		s += "<"
		for i, p := range n.Params {
			if i > 0 {
				s += ", "
			}
			s += p.String()
		}
		s += ">"
	}
	if n.Array {
		s = "[]" + s
	}
	if n.Nullable {
		s += "?"
	}
	return s
}

// ─── Variables ─────────────────────────────────────────────────────────────

// greeting = 'Hello'        — immutable, inferred
// mut counter number        — mutable, typed, no value
// mut flag = true           — mutable, inferred
type VarDecl struct {
	Name    string
	Mutable bool
	Type    *TypeExpr
	Value   Expression
}

func (n *VarDecl) statementNode()       {}
func (n *VarDecl) TokenLiteral() string { return n.Name }
func (n *VarDecl) String() string {
	mut := ""
	if n.Mutable {
		mut = "mut "
	}
	return fmt.Sprintf("%s%s", mut, n.Name)
}

// ─── Functions ─────────────────────────────────────────────────────────────

type Param struct {
	Name     string
	Type     TypeExpr
	Variadic bool
}

// fn sum(a number, b number) number { a + b }
type FnDecl struct {
	Name       string
	Async      bool
	Generator  bool   // fn*
	TypeParam  string // fn<T> - TODO: handling just one param for now, add support for multiple
	Params     []Param
	ReturnType *TypeExpr // nil = void
	Body       []Statement
	Exported   bool
}

func (n *FnDecl) statementNode()       {}
func (n *FnDecl) TokenLiteral() string { return "fn" }
func (n *FnDecl) String() string       { return fmt.Sprintf("fn %s(...)", n.Name) }

// Lambda: () { ... } or { item in ... }
type LambdaExpr struct {
	Params   []Param
	Body     []Statement
	Trailing bool // Swift-like trailing closure
}

func (n *LambdaExpr) expressionNode()      {}
func (n *LambdaExpr) TokenLiteral() string { return "lambda" }
func (n *LambdaExpr) String() string       { return "lambda(...)" }

// ─── Calls ─────────────────────────────────────────────────────────────────

// sum(a, b) OR Human(25, 180, 'Alex') OR nums.map { item in ... }
type CallExpr struct {
	Callee   Expression
	Args     []Expression
	Trailing *LambdaExpr
}

func (n *CallExpr) expressionNode()      {}
func (n *CallExpr) TokenLiteral() string { return "call" }
func (n *CallExpr) String() string       { return fmt.Sprintf("call(%s)", n.Callee.String()) }

// ─── Objects ───────────────────────────────────────────────────────────────

// object Human { ... }
type ObjectDecl struct {
	Name        string
	Parent      string // object Programmer: Human
	TypeParam   string // object Response<T>
	Fields      []Field
	Methods     []FnDecl // static
	Init        *FnDecl  // constructor fn init
	Exported    bool
	Annotations []string // @serializable etc
}

func (n *ObjectDecl) statementNode()       {}
func (n *ObjectDecl) TokenLiteral() string { return "object" }
func (n *ObjectDecl) String() string       { return fmt.Sprintf("object %s", n.Name) }

type Field struct {
	Name string
	Type TypeExpr
}

// this{age: age, name: name} — constructor ONLY!
type ThisInit struct {
	Fields map[string]Expression
}

func (n *ThisInit) statementNode()       {}
func (n *ThisInit) expressionNode()      {}
func (n *ThisInit) TokenLiteral() string { return "this" }
func (n *ThisInit) String() string       { return "this{...}" }

// super(age, height, name)
type SuperCall struct {
	Args []Expression
}

func (n *SuperCall) statementNode()       {}
func (n *SuperCall) expressionNode()      {}
func (n *SuperCall) TokenLiteral() string { return "super" }
func (n *SuperCall) String() string       { return "super(...)" }

// ─── Prototype ─────────────────────────────────────────────────────────────

// prototype Human: HumanLike { ... }
type PrototypeDecl struct {
	Name       string
	Interfaces []string
	Methods    []FnDecl
	Exported   bool
}

func (n *PrototypeDecl) statementNode()       {}
func (n *PrototypeDecl) TokenLiteral() string { return "prototype" }
func (n *PrototypeDecl) String() string       { return fmt.Sprintf("prototype %s", n.Name) }

// ─── Interface ─────────────────────────────────────────────────────────────

// interface HumanLike { sayHi() }
type InterfaceDecl struct {
	Name     string
	Methods  []MethodSig
	Exported bool
}

type MethodSig struct {
	Name       string
	Params     []Param
	ReturnType *TypeExpr
}

func (n *InterfaceDecl) statementNode()       {}
func (n *InterfaceDecl) TokenLiteral() string { return "interface" }
func (n *InterfaceDecl) String() string       { return fmt.Sprintf("interface %s", n.Name) }

// ─── Enum ──────────────────────────────────────────────────────────────────

// enum RequestState { Idle  Loading  Success(data User)  Failure(message string) }
type EnumDecl struct {
	Name     string
	Variants []EnumVariant
	Exported bool
}

type EnumVariant struct {
	Name    string
	Payload *Param
}

func (n *EnumDecl) statementNode()       {}
func (n *EnumDecl) TokenLiteral() string { return "enum" }
func (n *EnumDecl) String() string       { return fmt.Sprintf("enum %s", n.Name) }

// ─── Match ─────────────────────────────────────────────────────────────────

// match state { ... }
type MatchExpr struct {
	Subject Expression
	Arms    []MatchArm
}

type MatchArm struct {
	Pattern Expression // EnumPattern, Identifier, WildcardPattern
	Body    Expression // single-liner -> expr | block
}

// Status.Failure(msg) inside pattern matching
type EnumPattern struct {
	Variant string // "Status.Failure"
	Binding string // "msg" payload var name
}

func (n *EnumPattern) expressionNode()      {}
func (n *EnumPattern) TokenLiteral() string { return n.Variant }
func (n *EnumPattern) String() string       { return n.Variant }

// _ inside pattern matching
type WildcardPattern struct{}

func (n *WildcardPattern) expressionNode()      {}
func (n *WildcardPattern) TokenLiteral() string { return "_" }
func (n *WildcardPattern) String() string       { return "_" }

func (n *MatchExpr) expressionNode()      {}
func (n *MatchExpr) TokenLiteral() string { return "match" }
func (n *MatchExpr) String() string       { return "match(...)" }

// ─── Control Flow ──────────────────────────────────────────────────────────

// if counter > 0 { ... }
// if user = findUser(1) { ... }  — if-let
type IfStmt struct {
	Binding   *VarDecl   // if-let binding, nil for regular if
	Condition Expression // nil IF if-let
	Then      []Statement
	Else      []Statement
}

func (n *IfStmt) statementNode()       {}
func (n *IfStmt) TokenLiteral() string { return "if" }
func (n *IfStmt) String() string       { return "if(...)" }

// for item in nums { ... }
type ForStmt struct {
	Binding string
	Iter    Expression
	Body    []Statement
}

func (n *ForStmt) statementNode()       {}
func (n *ForStmt) TokenLiteral() string { return "for" }
func (n *ForStmt) String() string       { return "for(...)" }

// ─── Expressions ───────────────────────────────────────────────────────────

// a + b, a > 0
type BinaryExpr struct {
	Left  Expression
	Op    string
	Right Expression
}

func (n *BinaryExpr) expressionNode()      {}
func (n *BinaryExpr) TokenLiteral() string { return n.Op }
func (n *BinaryExpr) String() string {
	return fmt.Sprintf("(%s %s %s)", n.Left, n.Op, n.Right)
}

// !flag
type UnaryExpr struct {
	Op      string
	Operand Expression
}

func (n *UnaryExpr) expressionNode()      {}
func (n *UnaryExpr) TokenLiteral() string { return n.Op }
func (n *UnaryExpr) String() string       { return fmt.Sprintf("(%s%s)", n.Op, n.Operand) }

// person.name
type MemberExpr struct {
	Object   Expression
	Property string
}

func (n *MemberExpr) expressionNode()      {}
func (n *MemberExpr) TokenLiteral() string { return "." }
func (n *MemberExpr) String() string {
	return fmt.Sprintf("%s.%s", n.Object, n.Property)
}

// .{name, age} = person
type Destructure struct {
	Fields []string
	Value  Expression
}

func (n *Destructure) statementNode()       {}
func (n *Destructure) TokenLiteral() string { return ".{}" }
func (n *Destructure) String() string       { return ".{...}" }

// yield value
type YieldExpr struct {
	Value Expression
}

func (n *YieldExpr) expressionNode()      {}
func (n *YieldExpr) statementNode()       {}
func (n *YieldExpr) TokenLiteral() string { return "yield" }
func (n *YieldExpr) String() string       { return fmt.Sprintf("yield %s", n.Value) }

// try JSON.parse(raw)
type TryExpr struct {
	Expr Expression
}

func (n *TryExpr) expressionNode()      {}
func (n *TryExpr) TokenLiteral() string { return "try" }
func (n *TryExpr) String() string       { return fmt.Sprintf("try %s", n.Expr) }

// await fetchUser(1)
type AwaitExpr struct {
	Expr Expression
}

func (n *AwaitExpr) expressionNode()      {}
func (n *AwaitExpr) TokenLiteral() string { return "await" }
func (n *AwaitExpr) String() string       { return fmt.Sprintf("await %s", n.Expr) }

// [...a, ...b]
type ArrayLiteral struct {
	Elements []Expression
}

func (n *ArrayLiteral) expressionNode()      {}
func (n *ArrayLiteral) TokenLiteral() string { return "[]" }
func (n *ArrayLiteral) String() string       { return "[...]" }

// ...args spread
type SpreadExpr struct {
	Value Expression
}

func (n *SpreadExpr) expressionNode()      {}
func (n *SpreadExpr) TokenLiteral() string { return "..." }
func (n *SpreadExpr) String() string       { return fmt.Sprintf("...%s", n.Value) }

// ─── Imports ───────────────────────────────────────────────────────────────

// import ( * as z from zod  {v4 as uuid} from uuid )
type ImportDecl struct {
	Specifiers []ImportSpecifier
}

type ImportSpecifier struct {
	Name  string // local name
	Alias string // as alias
	Star  bool   // * as z
	Path  string // from path
}

func (n *ImportDecl) statementNode()       {}
func (n *ImportDecl) TokenLiteral() string { return "import" }
func (n *ImportDecl) String() string       { return "import(...)" }

// ─── Expression Statement ──────────────────────────────────────────────────

// last statement is fn return
type ExprStmt struct {
	Expr             Expression
	IsImplicitReturn bool
}

func (n *ExprStmt) statementNode()       {}
func (n *ExprStmt) TokenLiteral() string { return n.Expr.TokenLiteral() }
func (n *ExprStmt) String() string       { return n.Expr.String() }
