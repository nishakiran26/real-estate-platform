package config

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// LoadDotEnv reads KEY=VALUE pairs from a .env file into the process
// environment. It exists so a local `go run ./cmd/server` picks up the same
// settings Docker Compose passes as real environment variables.
//
// Values already present in the environment always win, so a deployment is
// never overridden by a stray file on disk.
//
// It searches the working directory and its parents, which lets the server be
// started from either the repository root or backend/.
func LoadDotEnv(filenames ...string) {
	if len(filenames) == 0 {
		filenames = []string{".env"}
	}
	for _, name := range filenames {
		if path := findUpwards(name, 4); path != "" {
			applyEnvFile(path)
		}
	}
}

// findUpwards looks for name in the working directory and up to depth parents.
func findUpwards(name string, depth int) string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	for i := 0; i <= depth; i++ {
		candidate := filepath.Join(dir, name)
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

func applyEnvFile(path string) {
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		key, value, ok := parseEnvLine(scanner.Text())
		if !ok {
			continue
		}
		if _, exists := os.LookupEnv(key); exists {
			continue
		}
		_ = os.Setenv(key, value)
	}
}

// parseEnvLine handles `KEY=value`, `export KEY=value`, quoted values and
// trailing comments, and skips blank or comment-only lines.
func parseEnvLine(line string) (key, value string, ok bool) {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "#") {
		return "", "", false
	}
	line = strings.TrimPrefix(line, "export ")

	name, raw, found := strings.Cut(line, "=")
	if !found {
		return "", "", false
	}
	key = strings.TrimSpace(name)
	if key == "" {
		return "", "", false
	}

	raw = strings.TrimSpace(raw)
	if quoted, isQuoted := unquote(raw); isQuoted {
		return key, quoted, true
	}
	// An unquoted value ends at the first inline comment.
	if idx := strings.Index(raw, " #"); idx >= 0 {
		raw = raw[:idx]
	}
	return key, strings.TrimSpace(raw), true
}

func unquote(s string) (string, bool) {
	if len(s) < 2 {
		return s, false
	}
	first, last := s[0], s[len(s)-1]
	if (first == '"' && last == '"') || (first == '\'' && last == '\'') {
		return s[1 : len(s)-1], true
	}
	return s, false
}
