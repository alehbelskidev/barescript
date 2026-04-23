package codegen

type Target int

const (
	TargetUniversal Target = iota
	TargetBrowser
	TargetNode
)

func ParseTarget(s string) Target {
	switch s {
	case "browser":
		return TargetBrowser
	case "node":
		return TargetNode
	default:
		return TargetUniversal
	}
}

var forbidden = map[Target][]string{
	TargetBrowser: {"node:fs", "node:path", "node:os", "node:crypto", "node:http"},
	TargetNode:    {"window", "document", "navigator", "localStorage"},
}

func (t Target) IsForbiddenImport(path string) bool {
	for _, f := range forbidden[t] {
		if f == path {
			return true
		}
	}
	return false
}

func (t Target) IsForbiddenIdent(name string) bool {
	for _, f := range forbidden[t] {
		if f == name {
			return true
		}
	}
	return false
}
