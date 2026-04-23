package diagnostic

type Severity int

const (
	Error Severity = iota
	Warning
	Info
	Hint
)

type Diagnostic struct {
	Severity Severity
	Message  string
	Line     int
	Col      int
	EndLine  int
	EndCol   int
}

func (d Diagnostic) String() string {
	switch d.Severity {
	case Error:
		return "[error] " + d.Message
	case Warning:
		return "[warn]  " + d.Message
	case Info:
		return "[info]  " + d.Message
	case Hint:
		return "[hint]  " + d.Message
	}
	return d.Message
}
