package template

import (
	"fmt"
	"regexp"
	"strings"
)

// tokenRe matches any template action tag, capturing optional whitespace trim
// markers and the inner content.
var tokenRe = regexp.MustCompile(`(\{\{-?\s*)([\s\S]*?)(\s*-?\}\})`)

// blockOpeners are text/template keywords that open a nested block scope.
var blockOpeners = map[string]bool{
	"range":  true,
	"if":     true,
	"with":   true,
	"block":  true,
	"define": true,
}

// preprocess rewrites {{yield <key>}}...{{end}} blocks into
// {{yieldBegin <key>}}...{{yieldEnd}} sentinel calls that the template engine
// can invoke via registered functions.
//
// It maintains a keyword stack so that {{end}} tags belonging to nested
// range/if/with blocks are left untouched; only the {{end}} that closes the
// yield block itself is rewritten to {{yieldEnd}}.
func preprocess(src string) (string, error) {
	type frame struct{ keyword string }

	var (
		stack  []frame // tracks open block keywords
		result strings.Builder
		pos    int
	)

	matches := tokenRe.FindAllStringIndex(src, -1)

	for _, loc := range matches {
		// Emit literal text before this token.
		result.WriteString(src[pos:loc[0]])
		pos = loc[1]

		full := src[loc[0]:loc[1]]
		inner := strings.TrimSpace(tokenRe.ReplaceAllString(full, "$2"))
		left := tokenRe.ReplaceAllString(full, "$1")
		right := tokenRe.ReplaceAllString(full, "$3")

		fields := strings.Fields(inner)
		if len(fields) == 0 {
			result.WriteString(full)
			continue
		}

		keyword := fields[0]

		switch {
		case keyword == "yield":
			// Rewrite to yieldBegin sentinel.
			args := strings.TrimSpace(strings.TrimPrefix(inner, "yield"))
			rewritten := left + "yieldBegin " + args + right
			result.WriteString(rewritten)
			stack = append(stack, frame{"yield"})

		case keyword == "end":
			if len(stack) > 0 && stack[len(stack)-1].keyword == "yield" {
				// This end closes a yield block.
				stack = stack[:len(stack)-1]
				result.WriteString(left + "yieldEnd" + right)
			} else {
				// This end closes a nested block; pop it and pass through.
				if len(stack) > 0 {
					stack = stack[:len(stack)-1]
				}
				result.WriteString(full)
			}

		default:
			if blockOpeners[keyword] {
				stack = append(stack, frame{keyword})
			}
			result.WriteString(full)
		}
	}

	if len(stack) > 0 {
		return "", fmt.Errorf("yieldable: unclosed block %q", stack[len(stack)-1].keyword)
	}

	// Emit any trailing literal text.
	result.WriteString(src[pos:])
	return result.String(), nil
}
