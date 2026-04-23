package checker

import "fmt"

type TypeKind int

const (
	KindUnknown TypeKind = iota
	KindNumber
	KindString
	KindBool
	KindVoid
	KindNever
	KindObject
	KindEnum
	KindFn
	KindArray
	KindOption
	KindResult
	KindPromise
)

type Type struct {
	Kind     TypeKind
	Name     string
	Nullable bool
	Array    bool
	Params   []*Type
}

func (t *Type) String() string {
	if t == nil {
		return "void"
	}
	s := t.Name
	if len(t.Params) > 0 {
		s += "<"
		for i, p := range t.Params {
			if i > 0 {
				s += ", "
			}
			s += p.String()
		}
		s += ">"
	}
	if t.Array {
		s = "[]" + s
	}
	if t.Nullable {
		s += "?"
	}
	return s
}

var (
	TypeNumber  = &Type{Kind: KindNumber, Name: "number"}
	TypeString  = &Type{Kind: KindString, Name: "string"}
	TypeBool    = &Type{Kind: KindBool, Name: "bool"}
	TypeVoid    = &Type{Kind: KindVoid, Name: "void"}
	TypeNever   = &Type{Kind: KindNever, Name: "never"}
	TypeUnknown = &Type{Kind: KindUnknown, Name: "unknown"}
	TypeFn      = &Type{Kind: KindFn, Name: "fn"}
)

func kindFromName(name string) TypeKind {
	switch name {
	case "number":
		return KindNumber
	case "string":
		return KindString
	case "bool":
		return KindBool
	case "void":
		return KindVoid
	case "never":
		return KindNever
	case "unknown":
		return KindUnknown
	case "Option":
		return KindOption
	case "Result":
		return KindResult
	case "Promise":
		return KindPromise
	default:
		return KindObject
	}
}

func (k TypeKind) String() string {
	return fmt.Sprintf("TypeKind(%d)", k)
}
