package codegen

import (
	"fmt"
	"strings"

	"github.com/alehbelskidev/barescript/internal/parser"
)

type Codegen struct {
	target Target
	buf    strings.Builder
	indent int
	errors []string
}

func New(target Target) *Codegen {
	return &Codegen{target: target}
}

func (g *Codegen) Errors() []string {
	return g.errors
}

func (g *Codegen) Generate(program *parser.Program) string {
	for _, stmt := range program.Statements {
		g.genStatement(stmt)
	}
	return g.buf.String()
}

// ─── Helpers ────────────────────────────────────────────────────────────────

func (g *Codegen) write(s string) {
	g.buf.WriteString(s)
}

func (g *Codegen) writeln(s string) {
	g.buf.WriteString(g.pad() + s + "\n")
}

func (g *Codegen) pad() string {
	return strings.Repeat("  ", g.indent)
}

func (g *Codegen) pushIndent() { g.indent++ }
func (g *Codegen) popIndent()  { g.indent-- }

func (g *Codegen) errorf(format string, args ...any) {
	g.errors = append(g.errors, fmt.Sprintf(format, args...))
}

// ─── Statements ─────────────────────────────────────────────────────────────

func (g *Codegen) genStatement(stmt parser.Statement) {
	switch s := stmt.(type) {
	case *parser.VarDecl:
		g.genVarDecl(s)
	case *parser.FnDecl:
		g.genFnDecl(s)
	case *parser.ObjectDecl:
		g.genObjectDecl(s)
	case *parser.PrototypeDecl:
		g.genPrototypeDecl(s)
	case *parser.InterfaceDecl:
	case *parser.EnumDecl:
		g.genEnumDecl(s)
	case *parser.ImportDecl:
		g.genImportDecl(s)
	case *parser.IfStmt:
		g.genIfStmt(s)
	case *parser.ForStmt:
		g.genForStmt(s)
	case *parser.Destructure:
		g.genDestructure(s)
	case *parser.ExprStmt:
		if s.IsImplicitReturn {
			g.writeln("return " + g.genExpr(s.Expr) + ";")
		} else {
			g.writeln(g.genExpr(s.Expr) + ";")
		}
	case *parser.ThisInit:
		g.genThisInit(s)
	case *parser.SuperCall:
		g.writeln("super(" + g.genArgs(s.Args) + ");")
	}
}

// ─── Var Decl ───────────────────────────────────────────────────────────────

func (g *Codegen) genVarDecl(s *parser.VarDecl) {
	keyword := "const"
	if s.Mutable {
		keyword = "let"
	}
	if s.Value != nil {
		g.writeln(fmt.Sprintf("%s %s = %s;", keyword, s.Name, g.genExpr(s.Value)))
	} else {
		g.writeln(fmt.Sprintf("%s %s;", keyword, s.Name))
	}
}

// ─── Functions ──────────────────────────────────────────────────────────────

func (g *Codegen) genFnDecl(s *parser.FnDecl) {
	prefix := ""
	if s.Exported {
		prefix = "export "
	}

	asyncKw := ""
	if s.Async {
		asyncKw = "async "
	}

	generatorStar := ""
	if s.Generator {
		generatorStar = "*"
	}

	params := g.genParams(s.Params)

	g.writeln(fmt.Sprintf("%s%sfunction%s %s(%s) {",
		prefix, asyncKw, generatorStar, s.Name, params))
	g.pushIndent()
	g.genBlock(s.Body)
	g.popIndent()
	g.writeln("}")
}

func (g *Codegen) genParams(params []parser.Param) string {
	var parts []string
	for _, p := range params {
		if p.Variadic {
			parts = append(parts, "..."+p.Name)
		} else {
			parts = append(parts, p.Name)
		}
	}
	return strings.Join(parts, ", ")
}

func (g *Codegen) genBlock(stmts []parser.Statement) {
	for _, stmt := range stmts {
		g.genStatement(stmt)
	}
}

// ─── Object ──────────────────────────────────────────────────────────────────

func (g *Codegen) genObjectDecl(s *parser.ObjectDecl) {
	prefix := ""
	if s.Exported {
		prefix = "export "
	}

	params := ""
	if s.Init != nil {
		params = g.genParams(s.Init.Params)
	}

	extendsStr := ""
	if s.Parent != "" {
		extendsStr = fmt.Sprintf(" extends %s", s.Parent)
	}

	hasSerializable := false
	for _, a := range s.Annotations {
		if a == "@serializable" {
			hasSerializable = true
		}
	}

	g.writeln(fmt.Sprintf("%sclass %s%s {", prefix, s.Name, extendsStr))
	g.pushIndent()

	g.writeln(fmt.Sprintf("constructor(%s) {", params))
	g.pushIndent()
	if s.Parent != "" {
		g.writeln("super();")
	}
	if s.Init != nil {
		g.genBlock(s.Init.Body)
	}
	g.popIndent()
	g.writeln("}")

	for _, m := range s.Methods {
		g.write(g.pad())
		g.write("static ")
		g.genMethodBody(&m)
	}

	if hasSerializable {
		g.genFromJSON(s)
	}

	g.popIndent()
	g.writeln("}")
}

func (g *Codegen) genMethodBody(s *parser.FnDecl) {
	asyncKw := ""
	if s.Async {
		asyncKw = "async "
	}
	generatorStar := ""
	if s.Generator {
		generatorStar = "*"
	}
	params := g.genParams(s.Params)
	g.writeln(fmt.Sprintf("%s%s%s(%s) {", asyncKw, generatorStar, s.Name, params))
	g.pushIndent()
	g.genBlock(s.Body)
	g.popIndent()
	g.writeln("}")
}

func (g *Codegen) genFromJSON(s *parser.ObjectDecl) {
	g.writeln("static fromJSON(raw) {")
	g.pushIndent()
	g.writeln("try {")
	g.pushIndent()
	g.writeln("const data = typeof raw === 'string' ? JSON.parse(raw) : raw;")
	g.writeln(fmt.Sprintf("const obj = new %s();", s.Name))
	for _, field := range s.Fields {
		g.writeln(fmt.Sprintf("obj.%s = data.%s;", field.Name, field.Name))
	}
	g.writeln("return { ok: true, value: obj };")
	g.popIndent()
	g.writeln("} catch (e) {")
	g.pushIndent()
	g.writeln("return { ok: false, error: e };")
	g.popIndent()
	g.writeln("}")
	g.popIndent()
	g.writeln("}")
}

// ─── Prototype ──────────────────────────────────────────────────────────────

func (g *Codegen) genPrototypeDecl(s *parser.PrototypeDecl) {
	for _, m := range s.Methods {
		asyncKw := ""
		if m.Async {
			asyncKw = "async "
		}
		params := g.genParams(m.Params)
		g.writeln(fmt.Sprintf("%s.prototype.%s = %sfunction(%s) {",
			s.Name, m.Name, asyncKw, params))
		g.pushIndent()
		g.genBlock(m.Body)
		g.popIndent()
		g.writeln("};")
	}
}

// ─── Enum ────────────────────────────────────────────────────────────────────

func (g *Codegen) genEnumDecl(s *parser.EnumDecl) {
	prefix := ""
	if s.Exported {
		prefix = "export "
	}

	g.writeln(fmt.Sprintf("%sconst %s = Object.freeze({", prefix, s.Name))
	g.pushIndent()

	for _, v := range s.Variants {
		if v.Payload == nil {
			// Active: 'Active'
			g.writeln(fmt.Sprintf("%s: '%s',", v.Name, v.Name))
		} else {
			// Failure: (message) => ({ _type: 'Failure', message })
			g.writeln(fmt.Sprintf("%s: (%s) => ({ _type: '%s', %s }),",
				v.Name, v.Payload.Name, v.Name, v.Payload.Name))
		}
	}

	g.popIndent()
	g.writeln("});")
}

// ─── Match ───────────────────────────────────────────────────────────────────

func (g *Codegen) genMatchExpr(e *parser.MatchExpr) string {
	var b strings.Builder
	subject := g.genExpr(e.Subject)
	subjectVar := "__m"

	b.WriteString(fmt.Sprintf("((%s) => {\n", subjectVar))

	for i, arm := range e.Arms {
		prefix := "if"
		if i > 0 {
			prefix = "} else if"
		}

		switch p := arm.Pattern.(type) {
		case *parser.WildcardPattern:
			b.WriteString("} else {\n")
			b.WriteString("  return " + g.genExpr(arm.Body) + ";\n")
			continue

		case *parser.EnumPattern:
			cond := g.genEnumPatternCond(subjectVar, p)
			b.WriteString(fmt.Sprintf("%s (%s) {\n", prefix, cond))
			if p.Binding != "" {
				b.WriteString(fmt.Sprintf("  const %s = %s.%s;\n",
					p.Binding, subjectVar, g.enumPayloadField(p.Variant)))
			}
			b.WriteString("  return " + g.genExpr(arm.Body) + ";\n")

		case *parser.Identifier:
			b.WriteString(fmt.Sprintf("%s (%s === %s) {\n", prefix, subjectVar, p.Name))
			b.WriteString("  return " + g.genExpr(arm.Body) + ";\n")
		}
	}

	b.WriteString("}})(" + subject + ")")
	return b.String()
}

func (g *Codegen) genEnumPatternCond(subjectVar string, p *parser.EnumPattern) string {
	parts := strings.SplitN(p.Variant, ".", 2)
	if len(parts) == 2 {
		// Status.Failure(msg) → __m._type === 'Failure'
		return fmt.Sprintf("%s._type === '%s' || %s === '%s'",
			subjectVar, parts[1], subjectVar, parts[1])
	}
	// Some(h) / None / Ok / Err
	switch p.Variant {
	case "Some":
		return fmt.Sprintf("%s !== null && %s !== undefined", subjectVar, subjectVar)
	case "None":
		return fmt.Sprintf("%s === null || %s === undefined", subjectVar, subjectVar)
	case "Ok":
		return fmt.Sprintf("%s && %s.ok === true", subjectVar, subjectVar)
	case "Err":
		return fmt.Sprintf("%s && %s.ok === false", subjectVar, subjectVar)
	}
	return fmt.Sprintf("%s === '%s'", subjectVar, p.Variant)
}

func (g *Codegen) enumPayloadField(variant string) string {
	parts := strings.SplitN(variant, ".", 2)
	if len(parts) == 2 {
		return strings.ToLower(parts[1][:1]) + parts[1][1:]
	}
	switch variant {
	case "Some":
		return "value"
	case "Ok":
		return "value"
	case "Err":
		return "error"
	}
	return "value"
}

// ─── Control Flow ────────────────────────────────────────────────────────────

func (g *Codegen) genIfStmt(s *parser.IfStmt) {
	if s.Binding != nil {
		// if-let → const __v = expr; if (__v != null) { const name = __v; ... }
		tmpVar := "__iflet_" + s.Binding.Name
		g.writeln(fmt.Sprintf("const %s = %s;", tmpVar, g.genExpr(s.Binding.Value)))
		g.writeln(fmt.Sprintf("if (%s != null) {", tmpVar))
		g.pushIndent()
		g.writeln(fmt.Sprintf("const %s = %s;", s.Binding.Name, tmpVar))
		g.genBlock(s.Then)
		g.popIndent()
	} else {
		g.writeln(fmt.Sprintf("if (%s) {", g.genExpr(s.Condition)))
		g.pushIndent()
		g.genBlock(s.Then)
		g.popIndent()
	}
	if len(s.Else) > 0 {
		g.writeln("} else {")
		g.pushIndent()
		g.genBlock(s.Else)
		g.popIndent()
	}
	g.writeln("}")
}

func (g *Codegen) genForStmt(s *parser.ForStmt) {
	g.writeln(fmt.Sprintf("for (const %s of %s) {", s.Binding, g.genExpr(s.Iter)))
	g.pushIndent()
	g.genBlock(s.Body)
	g.popIndent()
	g.writeln("}")
}

func (g *Codegen) genDestructure(s *parser.Destructure) {
	fields := strings.Join(s.Fields, ", ")
	g.writeln(fmt.Sprintf("const { %s } = %s;", fields, g.genExpr(s.Value)))
}

func (g *Codegen) genThisInit(s *parser.ThisInit) {
	for key, val := range s.Fields {
		g.writeln(fmt.Sprintf("this.%s = %s;", key, g.genExpr(val)))
	}
}

// ─── Import ──────────────────────────────────────────────────────────────────

func (g *Codegen) genImportDecl(s *parser.ImportDecl) {
	byPath := map[string][]parser.ImportSpecifier{}
	order := []string{}
	for _, spec := range s.Specifiers {
		if _, seen := byPath[spec.Path]; !seen {
			order = append(order, spec.Path)
		}
		byPath[spec.Path] = append(byPath[spec.Path], spec)
	}

	for _, path := range order {
		if g.target.IsForbiddenImport(path) {
			g.errorf("import %q is not allowed for target %v", path, g.target)
			continue
		}
		specs := byPath[path]
		var parts []string
		for _, spec := range specs {
			if spec.Star {
				parts = append(parts, fmt.Sprintf("* as %s", spec.Alias))
			} else if spec.Alias != "" {
				parts = append(parts, fmt.Sprintf("{ %s as %s }", spec.Name, spec.Alias))
			} else {
				parts = append(parts, fmt.Sprintf("{ %s }", spec.Name))
			}
		}
		g.writeln(fmt.Sprintf(`import %s from "%s";`,
			strings.Join(parts, ", "), path))
	}
}

// ─── Expressions ─────────────────────────────────────────────────────────────

func (g *Codegen) genExpr(expr parser.Expression) string {
	if expr == nil {
		return ""
	}
	switch e := expr.(type) {
	case *parser.IntLiteral:
		return e.Raw
	case *parser.FloatLiteral:
		return e.Raw
	case *parser.StringLiteral:
		return fmt.Sprintf(`"%s"`, e.Value)
	case *parser.InterpolatedString:
		return g.genInterpolatedString(e)
	case *parser.BoolLiteral:
		if e.Value {
			return "true"
		}
		return "false"
	case *parser.Identifier:
		return e.Name
	case *parser.BinaryExpr:
		return fmt.Sprintf("(%s %s %s)", g.genExpr(e.Left), e.Op, g.genExpr(e.Right))
	case *parser.UnaryExpr:
		return fmt.Sprintf("(%s%s)", e.Op, g.genExpr(e.Operand))
	case *parser.MemberExpr:
		return fmt.Sprintf("%s.%s", g.genExpr(e.Object), e.Property)
	case *parser.CallExpr:
		return g.genCallExpr(e)
	case *parser.MatchExpr:
		return g.genMatchExpr(e)
	case *parser.TryExpr:
		return g.genTryExpr(e)
	case *parser.AwaitExpr:
		return fmt.Sprintf("await %s", g.genExpr(e.Expr))
	case *parser.YieldExpr:
		return fmt.Sprintf("yield %s", g.genExpr(e.Value))
	case *parser.ArrayLiteral:
		return g.genArrayLiteral(e)
	case *parser.LambdaExpr:
		return g.genLambda(e)
	case *parser.SpreadExpr:
		return fmt.Sprintf("...%s", g.genExpr(e.Value))
	case *parser.ThisInit:
		return ""
	case *parser.SuperCall:
		return fmt.Sprintf("super(%s)", g.genArgs(e.Args))
	}
	return ""
}

func (g *Codegen) genCallExpr(e *parser.CallExpr) string {
	callee := g.genExpr(e.Callee)
	args := g.genArgs(e.Args)

	if e.Trailing != nil {
		lambda := g.genLambda(e.Trailing)
		if args == "" {
			return fmt.Sprintf("%s(%s)", callee, lambda)
		}
		return fmt.Sprintf("%s(%s, %s)", callee, args, lambda)
	}

	return fmt.Sprintf("%s(%s)", callee, args)
}

func (g *Codegen) genArgs(args []parser.Expression) string {
	var parts []string
	for _, a := range args {
		parts = append(parts, g.genExpr(a))
	}
	return strings.Join(parts, ", ")
}

func (g *Codegen) genLambda(e *parser.LambdaExpr) string {
	params := g.genParams(e.Params)
	var body strings.Builder

	if len(e.Body) == 1 {
		if es, ok := e.Body[0].(*parser.ExprStmt); ok {
			return fmt.Sprintf("(%s) => %s", params, g.genExpr(es.Expr))
		}
	}

	body.WriteString(fmt.Sprintf("(%s) => {\n", params))
	for _, stmt := range e.Body {
		body.WriteString("  ")
		switch s := stmt.(type) {
		case *parser.ExprStmt:
			if s.IsImplicitReturn {
				body.WriteString("return " + g.genExpr(s.Expr) + ";\n")
			} else {
				body.WriteString(g.genExpr(s.Expr) + ";\n")
			}
		default:
			inner := New(g.target)
			inner.genStatement(stmt)
			body.WriteString(inner.buf.String())
		}
	}
	body.WriteString("}")
	return body.String()
}

func (g *Codegen) genInterpolatedString(e *parser.InterpolatedString) string {
	var parts []string
	for _, part := range e.Parts {
		switch p := part.(type) {
		case *parser.TextPart:
			parts = append(parts, p.Value)
		default:
			parts = append(parts, "${"+g.genExpr(p)+"}")
		}
	}
	return "`" + strings.Join(parts, "") + "`"
}

func (g *Codegen) genArrayLiteral(e *parser.ArrayLiteral) string {
	var parts []string
	for _, el := range e.Elements {
		parts = append(parts, g.genExpr(el))
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

func (g *Codegen) genTryExpr(e *parser.TryExpr) string {
	// try expr → (() => { try { return { ok: true, value: expr }; } catch(e) { return { ok: false, error: e }; } })()
	inner := g.genExpr(e.Expr)
	return fmt.Sprintf(
		"(() => { try { return { ok: true, value: %s }; } catch(__e) { return { ok: false, error: __e }; } })()",
		inner,
	)
}
