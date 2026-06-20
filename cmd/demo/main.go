package main

import (
	"fmt"
	"os"

	"netztronaut.de/yieldable/template"
)

const tmplSrc = `{{- range $env := list "prod" "staging" }}
{{- yield $env }}
{{- range $name := list "hello" "world" }}
---
kind: ConfigMap
apiVersion: v1
metadata:
  name: {{ $name }}-{{ $env }}
{{- end }}
{{- end }}
{{- end }}`

func main() {
	t := template.Must(template.New("root").Funcs(template.FuncMap{
		"list": func(items ...any) []any { return items },
	}).Parse(tmplSrc))

	t.OnYield(func(key, content string) error {
		fmt.Printf("=== yield key=%q ===\n%s\n", key, content)
		return nil
	})

	if err := t.Execute(os.Stdout, nil); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
