package bff

import (
	"bytes"
	"strings"

	"github.com/yosssi/gohtml"
)

// formatHTML formats HTML for snapshot testing with 2-space indentation
func formatHTML(html string) string {
	// Configure gohtml for 2-space indentation
	formatted := gohtml.Format(html)

	// Post-process to ensure consistent formatting:
	// 1. Remove excessive blank lines
	lines := strings.Split(formatted, "\n")
	var result []string
	prevBlank := false

	for _, line := range lines {
		isBlank := strings.TrimSpace(line) == ""
		if isBlank && prevBlank {
			// Skip consecutive blank lines
			continue
		}
		result = append(result, line)
		prevBlank = isBlank
	}

	// Join and trim
	output := strings.Join(result, "\n")
	output = strings.TrimSpace(output)

	return output
}

// execTemplate is a helper for executing templates and formatting the output
func execTemplate(tmpl interface{}, templateName string, data interface{}) (string, error) {
	var buf bytes.Buffer

	type executor interface {
		ExecuteTemplate(*bytes.Buffer, string, interface{}) error
	}

	t, ok := tmpl.(executor)
	if !ok {
		panic("template does not implement ExecuteTemplate")
	}

	err := t.ExecuteTemplate(&buf, templateName, data)
	if err != nil {
		return "", err
	}

	return formatHTML(buf.String()), nil
}
