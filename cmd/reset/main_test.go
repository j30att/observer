package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateResetMethods(t *testing.T) {
	root := t.TempDir()
	input := `package sample

// generate:reset
type ResetableStruct struct {
	i int
	str string
	strP *string
	s []int
	m map[string]string
	child *ResetableStruct
	nested Nested
	anon struct {
		values []string
	}
}

type Nested struct {
	called bool
}

func (n *Nested) Reset() {
	n.called = true
}
`

	if err := os.WriteFile(filepath.Join(root, "sample.go"), []byte(input), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := generate(root); err != nil {
		t.Fatal(err)
	}

	generated, err := os.ReadFile(filepath.Join(root, generatedFileName))
	if err != nil {
		t.Fatal(err)
	}

	source := string(generated)
	wantSnippets := []string{
		"func (r *ResetableStruct) Reset()",
		"if r == nil",
		"r.s = r.s[:0]",
		"clear(r.m)",
		"if r.strP != nil",
		"*r.strP = zero",
		"if resetter, ok := any(r.child).(interface{ Reset() }); ok",
		"if resetter, ok := any(&r.nested).(interface{ Reset() }); ok",
		"r.anon.values = r.anon.values[:0]",
	}
	for _, want := range wantSnippets {
		if !strings.Contains(source, want) {
			t.Fatalf("generated source does not contain %q:\n%s", want, source)
		}
	}
}
