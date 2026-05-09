// Package parser provides utilities for template interpolation and variable extraction
// used by the w7s workflow orchestration engine.
package parser

import (
	"regexp"
	"strings"
)

var placeholderRe = regexp.MustCompile(`\{\{(\w+)\}\}`)

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
