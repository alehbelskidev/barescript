package lexer

type TokenType int

const (
	// Literals
	INT TokenType = iota
	FLOAT
	STRING
	IDENT

	// Keywords
	FN
	ASYNC
	MUT
	OBJECT
	PROTOTYPE
	INTERFACE
	ENUM
	MATCH
	IMPORT
	EXPORT
	FROM
	AS
	IN
	SUPER
	THIS
	YIELD
	IF
	ELSE
	FOR
	AWAIT
	RETURN
	TRY
	SOME
	NONE
	OK
	ERR
	UNPACK

	// Types
	TYPE_NUMBER
	TYPE_STRING
	TYPE_BOOL
	TYPE_VOID
	TYPE_NEVER
	TYPE_UNKNOWN
	TYPE_OPTION
	TYPE_RESULT
	TYPE_PROMISE

	// Annotations
	AT_SERIALIZABLE // @serializable
	AT_BUILDER      // @builder

	// Punctuation
	LPAREN     // (
	RPAREN     // )
	LBRACE     // {
	RBRACE     // }
	LBRACKET   // [
	RBRACKET   // ]
	COMMA      // ,
	DOT        // .
	COLON      // :
	SEMICOLON  // ;
	QUESTION   // ?
	STAR       // *
	ELLIPSIS   // ...
	ARROW      // ->
	HASH       // # (string interpolation)
	PIPE       // |
	UNDERSCORE // _

	// Assignment
	ASSIGN // =

	// Arithmetic
	PLUS
	MINUS
	SLASH
	PERCENT

	// Comparison
	EQ  // ==
	NEQ // !=
	LT  //
	GT  // >
	LTE // <=
	GTE // >=

	// Logical
	AND // &&
	OR  // ||
	NOT // !

	// Special
	FN_STAR  // fn*  — generator
	FN_ANGLE // fn<  — generic

	EOF
	ILLEGAL
)

type Token struct {
	Type    TokenType
	Literal string
	Line    int
	Col     int
}
