package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/alehbelskidev/barescript/internal/checker"
	"github.com/alehbelskidev/barescript/internal/codegen"
	"github.com/alehbelskidev/barescript/internal/lexer"
	"github.com/alehbelskidev/barescript/internal/parser"
)

func main() {
	targetFlag := flag.String("target", "universal", "browser | node | universal")
	checkOnly := flag.Bool("check", false, "type check only, no output")
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: bsc [--target browser|node|universal] [--check] <file.bs>")
		os.Exit(1)
	}

	target := codegen.ParseTarget(*targetFlag)

	for _, path := range args {
		if err := processFile(path, target, *checkOnly); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	}
}

func processFile(path string, target codegen.Target, checkOnly bool) error {
	src, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("cannot read %q: %w", path, err)
	}

	l := lexer.New(string(src))
	tokens, err := l.Tokenize()
	if err != nil {
		return fmt.Errorf("lex error in %q: %w", path, err)
	}

	for i, tok := range tokens {
		fmt.Printf("%d: %v %q\n", i, tok.Type, tok.Literal)
	}

	p := parser.New(tokens)
	program := p.Parse()
	if errs := p.Errors(); len(errs) > 0 {
		for _, e := range errs {
			fmt.Fprintf(os.Stderr, "%s: parse error: %s\n", path, e)
		}
		return fmt.Errorf("parse failed")
	}

	c := checker.New()
	diags := c.Check(program)
	hasErrors := false
	for _, d := range diags {
		fmt.Fprintf(os.Stderr, "%s: %s\n", path, d)
		if d.Severity == 0 { // Error
			hasErrors = true
		}
	}
	if hasErrors {
		return fmt.Errorf("type check failed")
	}

	if checkOnly {
		fmt.Printf("%s: ok\n", path)
		return nil
	}

	g := codegen.New(target)
	js := g.Generate(program)
	if errs := g.Errors(); len(errs) > 0 {
		for _, e := range errs {
			fmt.Fprintf(os.Stderr, "%s: codegen error: %s\n", path, e)
		}
		return fmt.Errorf("codegen failed")
	}

	outPath := strings.TrimSuffix(path, filepath.Ext(path)) + ".js"
	if err := os.WriteFile(outPath, []byte(js), 0644); err != nil {
		return fmt.Errorf("cannot write %q: %w", outPath, err)
	}

	fmt.Printf("%s → %s\n", path, outPath)
	return nil
}
