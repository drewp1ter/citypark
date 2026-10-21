package config

import (
	"bufio"
	"os"
	"sort"
	"strings"
	"testing"
)

func TestEnvExampleIsComplete(t *testing.T) {
	f, err := os.Open("../../.env.example")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	example := map[string]string{}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		name, value, ok := strings.Cut(line, "=")
		if !ok {
			t.Fatalf("line is not NAME=value: %q", line)
		}
		if n := len(value); n >= 2 && (value[0] == '\'' || value[0] == '"') && value[n-1] == value[0] {
			value = value[1 : n-1]
		}
		example[name] = value
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}

	asked := map[string]bool{}
	_, err = Load(func(key string) (string, bool) {
		asked[key] = true
		v, ok := example[key]
		return v, ok
	})
	if err != nil {
		t.Fatalf("Load with the example values: %v", err)
	}

	var missing, unknown []string
	for k := range asked {
		if _, ok := example[k]; !ok {
			missing = append(missing, k)
		}
	}
	for k := range example {
		if !asked[k] {
			unknown = append(unknown, k)
		}
	}
	sort.Strings(missing)
	sort.Strings(unknown)
	if len(missing) > 0 {
		t.Errorf("read by config but absent from .env.example: %v", missing)
	}
	if len(unknown) > 0 {
		t.Errorf("in .env.example but never read by config: %v", unknown)
	}
}
