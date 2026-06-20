package template

import (
	"bytes"
	"fmt"
	"io"
	"text/template"
)

// YieldFunc is the callback signature for yield blocks.
// key is the argument passed to {{yield <key>}}.
// content is the fully rendered body of the block.
type YieldFunc func(key, content string) error

// Template wraps text/template.Template with yield block support.
type Template struct {
	inner   *template.Template
	onYield YieldFunc
	// sw is set only for the duration of a single Execute call.
	sw *switchWriter
}

const (
	fnBegin = "yieldBegin"
	fnEnd   = "yieldEnd"
)

// New creates a new Template with the given name, mirroring text/template.New.
func New(name string) *Template {
	t := &Template{}
	t.inner = template.New(name).Funcs(template.FuncMap{
		fnBegin: func(key any) (string, error) {
			if t.sw == nil {
				return "", fmt.Errorf("yieldable: %s called outside Execute", fnBegin)
			}
			return "", t.sw.begin(fmt.Sprint(key))
		},
		fnEnd: func() (string, error) {
			if t.sw == nil {
				return "", fmt.Errorf("yieldable: %s called outside Execute", fnEnd)
			}
			return "", t.sw.end()
		},
	})
	return t
}

// Must panics if err is non-nil, mirroring text/template.Must.
func Must(t *Template, err error) *Template {
	if err != nil {
		panic(err)
	}
	return t
}

// Funcs adds functions to the template, mirroring text/template.Template.Funcs.
// Panics if yieldBegin or yieldEnd are supplied.
func (t *Template) Funcs(fm FuncMap) *Template {
	for name := range fm {
		if name == fnBegin || name == fnEnd {
			panic(fmt.Sprintf("yieldable: function name %q is reserved", name))
		}
	}
	t.inner = t.inner.Funcs(fm)
	return t
}

// Option sets options, mirroring text/template.Template.Option.
func (t *Template) Option(opt ...string) *Template {
	t.inner = t.inner.Option(opt...)
	return t
}

// Parse preprocesses src and parses the result.
func (t *Template) Parse(src string) (*Template, error) {
	rewritten, err := preprocess(src)
	if err != nil {
		return nil, err
	}
	if _, err = t.inner.Parse(rewritten); err != nil {
		return nil, err
	}
	return t, nil
}

// OnYield registers the callback invoked for each yield block.
func (t *Template) OnYield(fn YieldFunc) {
	t.onYield = fn
}

// Execute renders the template to w, mirroring text/template.Template.Execute.
func (t *Template) Execute(w io.Writer, data any) error {
	sw := &switchWriter{main: w, onYield: t.onYield}
	t.sw = sw
	defer func() { t.sw = nil }()
	return t.inner.Execute(sw, data)
}

// ExecuteTemplate renders a named associated template.
func (t *Template) ExecuteTemplate(w io.Writer, name string, data any) error {
	sw := &switchWriter{main: w, onYield: t.onYield}
	t.sw = sw
	defer func() { t.sw = nil }()
	return t.inner.ExecuteTemplate(sw, name, data)
}

// Lookup returns the template with the given name, or nil.
// The returned template inherits the onYield callback.
func (t *Template) Lookup(name string) *Template {
	inner := t.inner.Lookup(name)
	if inner == nil {
		return nil
	}
	child := &Template{inner: inner, onYield: t.onYield}
	// Re-wire sentinel functions to the child's sw field.
	child.inner = inner.Funcs(template.FuncMap{
		fnBegin: func(key any) (string, error) {
			if child.sw == nil {
				return "", fmt.Errorf("yieldable: %s called outside Execute", fnBegin)
			}
			return "", child.sw.begin(fmt.Sprint(key))
		},
		fnEnd: func() (string, error) {
			if child.sw == nil {
				return "", fmt.Errorf("yieldable: %s called outside Execute", fnEnd)
			}
			return "", child.sw.end()
		},
	})
	return child
}

// Name returns the template's name.
func (t *Template) Name() string { return t.inner.Name() }

// switchWriter routes template output to main or to a capture buffer.
type switchWriter struct {
	main    io.Writer
	buf     *bytes.Buffer
	key     string
	active  io.Writer
	onYield YieldFunc
}

func (sw *switchWriter) Write(p []byte) (int, error) {
	if sw.active != nil {
		return sw.active.Write(p)
	}
	return sw.main.Write(p)
}

func (sw *switchWriter) begin(key string) error {
	if sw.buf != nil {
		return fmt.Errorf("yieldable: nested yield blocks are not supported")
	}
	sw.key = key
	sw.buf = &bytes.Buffer{}
	sw.active = sw.buf
	return nil
}

func (sw *switchWriter) end() error {
	if sw.buf == nil {
		return fmt.Errorf("yieldable: yieldEnd called without matching yieldBegin")
	}
	content := sw.buf.String()
	key := sw.key
	sw.buf = nil
	sw.active = nil
	sw.key = ""
	if sw.onYield != nil {
		return sw.onYield(key, content)
	}
	return nil
}
