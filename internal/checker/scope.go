package checker

type Scope struct {
	parent    *Scope
	symbols   map[string]*Type
	immutable map[string]bool
}

func NewScope(parent *Scope) *Scope {
	return &Scope{
		parent:    parent,
		symbols:   map[string]*Type{},
		immutable: map[string]bool{},
	}
}

func (s *Scope) Define(name string, typ *Type) {
	s.symbols[name] = typ
}

func (s *Scope) MarkImmutable(name string) {
	s.immutable[name] = true
}

func (s *Scope) IsImmutable(name string) bool {
	if s.immutable[name] {
		return true
	}
	if s.parent != nil {
		return s.parent.IsImmutable(name)
	}
	return false
}

func (s *Scope) Lookup(name string) (*Type, bool) {
	if typ, ok := s.symbols[name]; ok {
		return typ, true
	}
	if s.parent != nil {
		return s.parent.Lookup(name)
	}
	return nil, false
}

func (s *Scope) Assign(name string, typ *Type) bool {
	if s.IsImmutable(name) {
		return false
	}
	cur := s
	for cur != nil {
		if _, ok := cur.symbols[name]; ok {
			cur.symbols[name] = typ
			return true
		}
		cur = cur.parent
	}
	return false
}
