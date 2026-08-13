package tools

import (
	"fmt"
	"os"
	"strings"
)

// resolveBody reads a message body from either the command line or a file.
// Files are the safe path for multiline text because shell quoting can turn
// escaped \n characters into literal text in a customer email.
func resolveBody(body, bodyFile string) (string, error) {
	if body == "" && bodyFile == "" {
		return "", fmt.Errorf("body or body-file is required")
	}
	if body != "" && bodyFile != "" {
		return "", fmt.Errorf("body and body-file cannot be used together")
	}

	resolved := body
	if bodyFile != "" {
		data, err := os.ReadFile(bodyFile)
		if err != nil {
			return "", fmt.Errorf("read body file %s: %v", bodyFile, err)
		}
		resolved = string(data)
	}
	if strings.TrimSpace(resolved) == "" {
		return "", fmt.Errorf("body cannot be empty")
	}
	if strings.Contains(resolved, `\n`) || strings.Contains(resolved, `\r`) {
		return "", fmt.Errorf("body contains literal escape sequences; use --body-file with real line breaks")
	}

	return resolved, nil
}
