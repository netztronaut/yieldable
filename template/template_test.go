package template

import (
	"bytes"
	"strings"
	"testing"
)

func TestYieldBasic(t *testing.T) {
	src := `{{- yield "a" }}hello{{- end }}`
	tmpl := Must(New("test").Parse(src))

	var got []string
	tmpl.OnYield(func(key, content string) error {
		got = append(got, key+":"+content)
		return nil
	})

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, nil); err != nil {
		t.Fatal(err)
	}
	if buf.Len() != 0 {
		t.Errorf("main output should be empty, got %q", buf.String())
	}
	if len(got) != 1 || got[0] != "a:hello" {
		t.Errorf("unexpected yield calls: %v", got)
	}
}

func TestYieldNestedRange(t *testing.T) {
	src := `{{- range $e := . }}{{- yield $e }}x{{- end }}{{- end }}`
	tmpl := Must(New("test").Parse(src))

	var keys []string
	tmpl.OnYield(func(key, content string) error {
		keys = append(keys, key)
		return nil
	})

	if err := tmpl.Execute(bytes.NewBuffer(nil), []string{"a", "b"}); err != nil {
		t.Fatal(err)
	}
	if len(keys) != 2 || keys[0] != "a" || keys[1] != "b" {
		t.Errorf("unexpected keys: %v", keys)
	}
}

func TestNestedYieldError(t *testing.T) {
	src := `{{- yield "outer" }}{{- yield "inner" }}x{{- end }}{{- end }}`
	tmpl := Must(New("test").Parse(src))
	tmpl.OnYield(func(_, _ string) error { return nil })

	err := tmpl.Execute(bytes.NewBuffer(nil), nil)
	if err == nil || !strings.Contains(err.Error(), "nested") {
		t.Errorf("expected nested yield error, got %v", err)
	}
}

func TestReservedFuncPanic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for reserved func name")
		}
	}()
	New("x").Funcs(FuncMap{"yieldBegin": func() {}})
}

func TestPreprocessUnclosed(t *testing.T) {
	_, err := preprocess(`{{yield "k"}}body`)
	if err == nil {
		t.Error("expected error for unclosed yield")
	}
}
