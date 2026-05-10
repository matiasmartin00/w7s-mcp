// Package parser provides utilities for template interpolation and variable extraction
// used by the w7s workflow orchestration engine.
package parser

import (
	"fmt"
	"regexp"
	"strings"
)

var placeholderRe = regexp.MustCompile(`\{\{(\w+)\}\}`)

// ErrMissingVariable is returned when a template contains a placeholder that
// is not present in the provided variables map.
type ErrMissingVariable struct {
	Variable string
}

func (e ErrMissingVariable) Error() string {
	return fmt.Sprintf("missing variable %q", e.Variable)
}

// Interpolate replaces all {{variable}} placeholders in template with the
// corresponding values from vars. Placeholders for which no variable exists
// are left unchanged in the output.
func Interpolate(template string, vars map[string]string) string {
	return placeholderRe.ReplaceAllStringFunc(template, func(match string) string {
		// Extract the variable name (strip {{ and }})
		key := match[2 : len(match)-2]
		if val, ok := vars[key]; ok {
			return val
		}
		// Missing variable: leave placeholder as-is.
		return match
	})
}

// InterpolateStrict replaces all {{variable}} placeholders in template with
// values from vars. Returns ErrMissingVariable when at least one placeholder is
// not found in vars.
func InterpolateStrict(template string, vars map[string]string) (string, error) {
	var missing string
	out := placeholderRe.ReplaceAllStringFunc(template, func(match string) string {
		key := match[2 : len(match)-2]
		if val, ok := vars[key]; ok {
			return val
		}
		if missing == "" {
			missing = key
		}
		return match
	})

	if missing != "" {
		return "", ErrMissingVariable{Variable: missing}
	}

	return out, nil
}

// Extract applies the given extraction patterns (map of varName → regex with
// one capture group) against output and returns only the variables whose
// pattern produced a successful match. Non-matching patterns are silently
// skipped — they do not produce an entry in the result map.
func Extract(output string, patterns map[string]string) map[string]string {
	result := make(map[string]string, len(patterns))
	for varName, pattern := range patterns {
		re, err := regexp.Compile(pattern)
		if err != nil {
			// Invalid regex: skip silently.
			continue
		}
		m := re.FindStringSubmatch(output)
		if len(m) < 2 {
			// No match or no capture group: skip.
			continue
		}
		value := strings.TrimSpace(m[1])
		if value != "" {
			result[varName] = value
		}
	}
	return result
}
