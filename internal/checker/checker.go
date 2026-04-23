package checker

import (
	"fmt"

	"github.com/alehbelskidev/barescript/internal/diagnostic"
	"github.com/alehbelskidev/barescript/internal/parser"
)

type Checker struct {
	scope       *Scope
	diagnostics []diagnostic.Diagnostic
	objects     map[string]*parser.ObjectDecl
	enums       map[string]*parser.EnumDecl
	interfaces  map[string]*parser.InterfaceDecl
	currentFn   *parser.FnDecl
}

func New() *Checker {
	c := &Checker{
		scope:      NewScope(nil),
		objects:    map[string]*parser.ObjectDecl{},
		enums:      map[string]*parser.EnumDecl{},
		interfaces: map[string]*parser.InterfaceDecl{},
	}
	c.loadPrelude()
	return c
}

func (c *Checker) loadPrelude() {
	globals := []string{
		"console", "JSON", "Math", "Object", "Array",
		"setTimeout", "setInterval", "clearTimeout", "clearInterval",
		"Promise", "Error", "Date", "RegExp", "Map", "Set",
		"parseInt", "parseFloat", "isNaN", "isFinite",
		"encodeURIComponent", "decodeURIComponent",
		"fetch", "window", "document", "navigator",
		"process", "Buffer", // node
		"Some", "None", "Ok", "Err", // builtins
	}
	for _, name := range globals {
		c.scope.Define(name, TypeUnknown)
	}
}

func (c *Checker) Check(program *parser.Program) []diagnostic.Diagnostic {
	c.collectDeclarations(program)

	for _, stmt := range program.Statements {
		c.checkStatement(stmt)
	}

	return c.diagnostics
}

// ─── Pass 1: collect ────────────────────────────────────────────────────────

func (c *Checker) collectDeclarations(program *parser.Program) {
	for _, stmt := range program.Statements {
		switch s := stmt.(type) {
		case *parser.ObjectDecl:
			c.objects[s.Name] = s
			c.scope.Define(s.Name, &Type{Kind: KindObject, Name: s.Name})

		case *parser.EnumDecl:
			c.enums[s.Name] = s
			c.scope.Define(s.Name, &Type{Kind: KindEnum, Name: s.Name})

		case *parser.InterfaceDecl:
			c.interfaces[s.Name] = s

		case *parser.FnDecl:
			typ := c.fnDeclType(s)
			c.scope.Define(s.Name, typ)
		}
	}
}

// ─── Statements ─────────────────────────────────────────────────────────────

func (c *Checker) checkStatement(stmt parser.Statement) {
	switch s := stmt.(type) {
	case *parser.VarDecl:
		c.checkVarDecl(s)
	case *parser.FnDecl:
		c.checkFnDecl(s)
	case *parser.ObjectDecl:
		c.checkObjectDecl(s)
	case *parser.PrototypeDecl:
		c.checkPrototypeDecl(s)
	case *parser.EnumDecl:
	case *parser.InterfaceDecl:
	case *parser.IfStmt:
		c.checkIfStmt(s)
	case *parser.ForStmt:
		c.checkForStmt(s)
	case *parser.ExprStmt:
		c.checkExpr(s.Expr)
	case *parser.Destructure:
		c.checkDestructure(s)
	case *parser.ImportDecl:
		c.checkImport(s)
	}
}

// ─── Var Decl ───────────────────────────────────────────────────────────────

func (c *Checker) checkVarDecl(s *parser.VarDecl) {
	var typ *Type

	if s.Value != nil {
		inferred := c.checkExpr(s.Value)
		if s.Type != nil {
			declared := c.resolveTypeExpr(s.Type)
			if !c.assignable(inferred, declared) {
				c.errorf("cannot assign %s to %s", inferred, declared)
			}
			typ = declared
		} else {
			typ = inferred
		}
	} else {
		if s.Type == nil {
			c.errorf("variable %q must have a type or value", s.Name)
			return
		}
		if !s.Mutable {
			c.errorf("immutable variable %q must be initialized", s.Name)
			return
		}
		typ = c.resolveTypeExpr(s.Type)
	}

	c.scope.Define(s.Name, typ)

	if !s.Mutable {
		c.scope.MarkImmutable(s.Name)
	}
}

// ─── Function ───────────────────────────────────────────────────────────────

func (c *Checker) checkFnDecl(s *parser.FnDecl) {
	prev := c.currentFn
	c.currentFn = s

	c.pushScope()
	for _, p := range s.Params {
		c.scope.Define(p.Name, c.resolveTypeExpr(&p.Type))
	}
	c.checkBlock(s.Body)
	c.popScope()

	c.currentFn = prev
}

func (c *Checker) checkBlock(stmts []parser.Statement) {
	for _, stmt := range stmts {
		c.checkStatement(stmt)
	}
}

// ─── Object ─────────────────────────────────────────────────────────────────

func (c *Checker) checkObjectDecl(s *parser.ObjectDecl) {
	if s.Parent != "" {
		if _, ok := c.objects[s.Parent]; !ok {
			c.errorf("object %q extends unknown object %q", s.Name, s.Parent)
		}
	}

	if c.hasAnnotation(s, "@serializable") {
		for _, field := range s.Fields {
			if c.isObjectType(field.Type.Name) {
				dep, ok := c.objects[field.Type.Name]
				if !ok {
					continue
				}
				if !c.hasAnnotation(dep, "@serializable") {
					c.errorf(
						"field %q in @serializable object %q uses %q which is not @serializable",
						field.Name, s.Name, field.Type.Name,
					)
				}
			}
		}
		if c.hasSerializableCycle(s.Name, map[string]bool{}) {
			c.errorf("@serializable object %q has a circular dependency", s.Name)
		}
	}

	if s.Init != nil {
		c.checkFnDecl(s.Init)
	}

	for i := range s.Methods {
		c.checkFnDecl(&s.Methods[i])
	}
}

func (c *Checker) hasSerializableCycle(name string, visited map[string]bool) bool {
	if visited[name] {
		return true
	}
	visited[name] = true
	obj, ok := c.objects[name]
	if !ok {
		return false
	}
	for _, field := range obj.Fields {
		if c.isObjectType(field.Type.Name) {
			if c.hasSerializableCycle(field.Type.Name, visited) {
				return true
			}
		}
	}
	return false
}

// ─── Prototype ──────────────────────────────────────────────────────────────

func (c *Checker) checkPrototypeDecl(s *parser.PrototypeDecl) {
	if _, ok := c.objects[s.Name]; !ok {
		c.errorf("prototype declared for unknown object %q", s.Name)
	}

	for _, ifaceName := range s.Interfaces {
		iface, ok := c.interfaces[ifaceName]
		if !ok {
			c.errorf("prototype %q implements unknown interface %q", s.Name, ifaceName)
			continue
		}
		for _, required := range iface.Methods {
			if !c.prototypeHasMethod(s, required.Name) {
				c.errorf(
					"prototype %q must implement %q from interface %q",
					s.Name, required.Name, ifaceName,
				)
			}
		}
	}

	for i := range s.Methods {
		c.checkFnDecl(&s.Methods[i])
	}
}

func (c *Checker) prototypeHasMethod(s *parser.PrototypeDecl, name string) bool {
	for _, m := range s.Methods {
		if m.Name == name {
			return true
		}
	}
	return false
}

// ─── If / For ───────────────────────────────────────────────────────────────

func (c *Checker) checkIfStmt(s *parser.IfStmt) {
	c.pushScope()

	if s.Binding != nil {
		valType := c.checkExpr(s.Binding.Value)
		if !valType.Nullable {
			c.warnf("if-let binding %q is not nullable — condition is always true", s.Binding.Name)
		}
		if valType.Kind != KindUnknown && !valType.Nullable {
			c.warnf("if-let binding %q is not nullable", s.Binding.Name)
		}
		unwrapped := &Type{Kind: valType.Kind, Name: valType.Name, Nullable: false}
		c.scope.Define(s.Binding.Name, unwrapped)
	} else {
		c.checkExpr(s.Condition)
	}

	c.checkBlock(s.Then)
	c.popScope()

	if len(s.Else) > 0 {
		c.pushScope()
		c.checkBlock(s.Else)
		c.popScope()
	}
}

func (c *Checker) checkForStmt(s *parser.ForStmt) {
	iterType := c.checkExpr(s.Iter)

	c.pushScope()
	elemType := c.arrayElemType(iterType)
	c.scope.Define(s.Binding, elemType)
	c.checkBlock(s.Body)
	c.popScope()
}

// ─── Destructure ────────────────────────────────────────────────────────────

func (c *Checker) checkDestructure(s *parser.Destructure) {
	objType := c.checkExpr(s.Value)
	if objType.Kind == KindUnknown {
		return
	}

	obj, ok := c.objects[objType.Name]
	if !ok {
		c.errorf("cannot destructure non-object type %q", objType.Name)
		return
	}
	for _, field := range s.Fields {
		if !c.objectHasField(obj, field) {
			c.errorf("object %q has no field %q", objType.Name, field)
		} else {
			c.scope.Define(field, c.fieldType(obj, field))
		}
	}
}

// ─── Import ─────────────────────────────────────────────────────────────────

func (c *Checker) checkImport(s *parser.ImportDecl) {
	// TODO: resolve through importer/dts.go
	// unknown meanwhile
	for _, spec := range s.Specifiers {
		name := spec.Name
		if spec.Alias != "" {
			name = spec.Alias
		}
		if spec.Star {
			name = spec.Alias
		}
		c.scope.Define(name, &Type{Kind: KindUnknown, Name: name})
	}
}

// ─── Expressions ────────────────────────────────────────────────────────────

func (c *Checker) checkExpr(expr parser.Expression) *Type {
	if expr == nil {
		return TypeVoid
	}
	switch e := expr.(type) {
	case *parser.IntLiteral:
		return TypeNumber
	case *parser.FloatLiteral:
		return TypeNumber
	case *parser.StringLiteral:
		return TypeString
	case *parser.InterpolatedString:
		return TypeString
	case *parser.BoolLiteral:
		return TypeBool
	case *parser.Identifier:
		return c.checkIdentifier(e)
	case *parser.BinaryExpr:
		return c.checkBinaryExpr(e)
	case *parser.UnaryExpr:
		return c.checkUnaryExpr(e)
	case *parser.MemberExpr:
		return c.checkMemberExpr(e)
	case *parser.CallExpr:
		return c.checkCallExpr(e)
	case *parser.MatchExpr:
		return c.checkMatchExpr(e)
	case *parser.TryExpr:
		return c.checkTryExpr(e)
	case *parser.AwaitExpr:
		inner := c.checkExpr(e.Expr)
		return c.unwrapPromise(inner)
	case *parser.YieldExpr:
		if c.currentFn == nil || !c.currentFn.Generator {
			c.errorf("yield outside generator function")
		}
		return c.checkExpr(e.Value)
	case *parser.ArrayLiteral:
		return c.checkArrayLiteral(e)
	case *parser.LambdaExpr:
		return TypeFn
	case *parser.SpreadExpr:
		return c.checkExpr(e.Value)
	case *parser.ThisInit:
		return TypeVoid
	case *parser.SuperCall:
		return TypeVoid
	}
	return TypeUnknown
}

func (c *Checker) checkIdentifier(e *parser.Identifier) *Type {
	typ, ok := c.scope.Lookup(e.Name)
	if !ok {
		c.errorf("undefined: %q", e.Name)
		return TypeUnknown
	}
	return typ
}

func (c *Checker) checkBinaryExpr(e *parser.BinaryExpr) *Type {
	left := c.checkExpr(e.Left)
	right := c.checkExpr(e.Right)

	switch e.Op {
	case "+", "-", "*", "/", "%":
		if left.Kind == KindNumber && right.Kind == KindNumber {
			return TypeNumber
		}
		if e.Op == "+" && left.Kind == KindString {
			return TypeString
		}
		c.errorf("operator %q not applicable to %s and %s", e.Op, left, right)
		return TypeUnknown
	case "==", "!=", "<", ">", "<=", ">=":
		return TypeBool
	case "&&", "||":
		return TypeBool
	}
	return TypeUnknown
}

func (c *Checker) checkUnaryExpr(e *parser.UnaryExpr) *Type {
	operand := c.checkExpr(e.Operand)
	switch e.Op {
	case "!":
		return TypeBool
	case "-":
		if operand.Kind != KindNumber {
			c.errorf("unary - requires number, got %s", operand)
		}
		return TypeNumber
	}
	return TypeUnknown
}

func (c *Checker) checkMemberExpr(e *parser.MemberExpr) *Type {
	objType := c.checkExpr(e.Object)

	if objType.Nullable {
		c.errorf(
			"cannot access %q on nullable type %s — unwrap with if-let first",
			e.Property, objType,
		)
		return TypeUnknown
	}

	obj, ok := c.objects[objType.Name]
	if !ok {
		return TypeUnknown
	}

	for _, field := range obj.Fields {
		if field.Name == e.Property {
			return c.resolveTypeExpr(&field.Type)
		}
	}

	if e.Property == "fromJSON" {
		return &Type{Kind: KindResult, Name: "Result"}
	}

	c.errorf("object %q has no field %q", objType.Name, e.Property)
	return TypeUnknown
}

func (c *Checker) checkCallExpr(e *parser.CallExpr) *Type {
	fnType := c.checkExpr(e.Callee)
	_ = fnType
	// TODO: fn signature check
	// unknown meanwhile
	return TypeUnknown
}

func (c *Checker) checkMatchExpr(e *parser.MatchExpr) *Type {
	subjectType := c.checkExpr(e.Subject)

	if subjectType.Kind == KindEnum {
		c.checkMatchExhaustive(e, subjectType)
	}

	if len(e.Arms) > 0 {
		return c.checkExpr(e.Arms[0].Body)
	}
	return TypeVoid
}

func (c *Checker) checkMatchExhaustive(e *parser.MatchExpr, enumType *Type) {
	enumDecl, ok := c.enums[enumType.Name]
	if !ok {
		return
	}

	hasWildcard := false
	covered := map[string]bool{}

	for _, arm := range e.Arms {
		switch p := arm.Pattern.(type) {
		case *parser.WildcardPattern:
			hasWildcard = true
		case *parser.EnumPattern:
			variant := p.Variant
			if idx := len(enumType.Name) + 1; len(variant) > idx {
				variant = variant[idx:]
			}
			covered[variant] = true
		}
	}

	if !hasWildcard {
		for _, variant := range enumDecl.Variants {
			if !covered[variant.Name] {
				c.errorf(
					"match on %q is not exhaustive — missing variant %q",
					enumType.Name, variant.Name,
				)
			}
		}
	}
}

func (c *Checker) checkTryExpr(e *parser.TryExpr) *Type {
	inner := c.checkExpr(e.Expr)
	return &Type{
		Kind: KindResult,
		Name: "Result",
		Params: []*Type{
			inner,
			{Kind: KindObject, Name: "Error"},
		},
	}
}

func (c *Checker) checkArrayLiteral(e *parser.ArrayLiteral) *Type {
	if len(e.Elements) == 0 {
		return &Type{Kind: KindArray, Name: "array", Params: []*Type{TypeUnknown}}
	}
	elemType := c.checkExpr(e.Elements[0])
	return &Type{Kind: KindArray, Name: "array", Params: []*Type{elemType}}
}

// ─── Helpers ────────────────────────────────────────────────────────────────

func (c *Checker) resolveTypeExpr(t *parser.TypeExpr) *Type {
	if t == nil {
		return TypeVoid
	}
	kind := kindFromName(t.Name)
	var params []*Type
	for _, p := range t.Params {
		pp := p
		params = append(params, c.resolveTypeExpr(&pp))
	}
	return &Type{
		Kind:     kind,
		Name:     t.Name,
		Nullable: t.Nullable,
		Array:    t.Array,
		Params:   params,
	}
}

func (c *Checker) fnDeclType(s *parser.FnDecl) *Type {
	return &Type{Kind: KindFn, Name: s.Name}
}

func (c *Checker) hasAnnotation(s *parser.ObjectDecl, name string) bool {
	for _, a := range s.Annotations {
		if a == name {
			return true
		}
	}
	return false
}

func (c *Checker) isObjectType(name string) bool {
	_, ok := c.objects[name]
	return ok
}

func (c *Checker) assignable(from, to *Type) bool {
	if from == nil || to == nil {
		return true
	}
	if to.Kind == KindUnknown {
		return true
	}
	return from.Kind == to.Kind && from.Name == to.Name
}

func (c *Checker) arrayElemType(t *Type) *Type {
	if t.Array && len(t.Params) > 0 {
		return t.Params[0]
	}
	return TypeUnknown
}

func (c *Checker) unwrapPromise(t *Type) *Type {
	if t.Name == "Promise" && len(t.Params) > 0 {
		return t.Params[0]
	}
	return t
}

func (c *Checker) objectHasField(obj *parser.ObjectDecl, name string) bool {
	for _, f := range obj.Fields {
		if f.Name == name {
			return true
		}
	}
	return false
}

func (c *Checker) fieldType(obj *parser.ObjectDecl, name string) *Type {
	for _, f := range obj.Fields {
		if f.Name == name {
			return c.resolveTypeExpr(&f.Type)
		}
	}
	return TypeUnknown
}

func (c *Checker) pushScope() {
	c.scope = NewScope(c.scope)
}

func (c *Checker) popScope() {
	c.scope = c.scope.parent
}

func (c *Checker) errorf(format string, args ...any) {
	c.diagnostics = append(c.diagnostics, diagnostic.Diagnostic{
		Severity: diagnostic.Error,
		Message:  fmt.Sprintf(format, args...),
	})
}

func (c *Checker) warnf(format string, args ...any) {
	c.diagnostics = append(c.diagnostics, diagnostic.Diagnostic{
		Severity: diagnostic.Warning,
		Message:  fmt.Sprintf(format, args...),
	})
}
