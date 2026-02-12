package redis

import (
	"bufio"
	"bytes"
	"strings"
	"testing"
)

func TestNewCmd(t *testing.T) {
	cmd := NewCmd()
	if cmd.Use != "redis" {
		t.Errorf("expected Use 'redis', got %q", cmd.Use)
	}
	subs := map[string]bool{}
	for _, s := range cmd.Commands() {
		subs[s.Name()] = true
	}
	for _, name := range []string{"get", "set", "del", "keys", "info"} {
		if !subs[name] {
			t.Errorf("missing subcommand %q", name)
		}
	}
}

func TestReadResponseSimpleString(t *testing.T) {
	input := "+OK\r\n"
	reader := bufio.NewReader(strings.NewReader(input))
	result, err := readResponse(reader)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result != "OK" {
		t.Errorf("expected 'OK', got %q", result)
	}
}

func TestReadResponseError(t *testing.T) {
	input := "-ERR unknown command\r\n"
	reader := bufio.NewReader(strings.NewReader(input))
	_, err := readResponse(reader)
	if err == nil {
		t.Error("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "unknown command") {
		t.Errorf("expected error to contain 'unknown command', got %v", err)
	}
}

func TestReadResponseInteger(t *testing.T) {
	input := ":42\r\n"
	reader := bufio.NewReader(strings.NewReader(input))
	result, err := readResponse(reader)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result != "42" {
		t.Errorf("expected '42', got %q", result)
	}
}

func TestReadResponseBulkString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple", "$5\r\nhello\r\n", "hello"},
		{"empty", "$0\r\n\r\n", ""},
		{"nil", "$-1\r\n", "(nil)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := bufio.NewReader(strings.NewReader(tt.input))
			result, err := readResponse(reader)
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestReadResponseArray(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			"simple array",
			"*2\r\n$3\r\nfoo\r\n$3\r\nbar\r\n",
			[]string{"foo", "bar"},
		},
		{
			"empty array",
			"*0\r\n",
			[]string{},
		},
		{
			"nil array",
			"*-1\r\n",
			[]string{"(nil)"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := bufio.NewReader(strings.NewReader(tt.input))
			result, err := readResponse(reader)
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			parts := strings.Split(result, "\n")
			// Filter empty strings
			filtered := []string{}
			for _, p := range parts {
				if p != "" {
					filtered = append(filtered, p)
				}
			}

			if len(filtered) != len(tt.expected) {
				t.Errorf("expected %d elements, got %d", len(tt.expected), len(filtered))
			}
			for i, exp := range tt.expected {
				if i < len(filtered) && filtered[i] != exp {
					t.Errorf("element %d: expected %q, got %q", i, exp, filtered[i])
				}
			}
		})
	}
}

func TestReadResponseDepthLimit(t *testing.T) {
	// Create a deeply nested array that exceeds maxRESPDepth
	var buf bytes.Buffer
	// Create nested arrays
	for i := 0; i < maxRESPDepth+5; i++ {
		buf.WriteString("*1\r\n")
	}
	buf.WriteString("+OK\r\n")

	reader := bufio.NewReader(&buf)
	_, err := readResponse(reader)
	if err == nil {
		t.Error("expected depth exceeded error, got nil")
	}
	if !strings.Contains(err.Error(), "depth exceeded") {
		t.Errorf("expected 'depth exceeded' error, got %v", err)
	}
}

func TestParseInfo(t *testing.T) {
	input := `# Server
redis_version:7.0.0
redis_mode:standalone
os:Linux 5.10.0-18-amd64 x86_64

# Clients
connected_clients:5

# Memory
used_memory:1024000
used_memory_human:1000.00K
`

	result := parseInfo(input)
	if result["redis_version"] != "7.0.0" {
		t.Errorf("expected redis_version '7.0.0', got %q", result["redis_version"])
	}
	if result["redis_mode"] != "standalone" {
		t.Errorf("expected redis_mode 'standalone', got %q", result["redis_mode"])
	}
	if result["connected_clients"] != "5" {
		t.Errorf("expected connected_clients '5', got %q", result["connected_clients"])
	}
	if result["used_memory_human"] != "1000.00K" {
		t.Errorf("expected used_memory_human '1000.00K', got %q", result["used_memory_human"])
	}
}

func TestParseInfoIgnoresComments(t *testing.T) {
	input := `# This is a comment
key1:value1
# Another comment

key2:value2
`

	result := parseInfo(input)
	if len(result) != 2 {
		t.Errorf("expected 2 entries, got %d", len(result))
	}
	if result["key1"] != "value1" {
		t.Errorf("expected key1 'value1', got %q", result["key1"])
	}
	if result["key2"] != "value2" {
		t.Errorf("expected key2 'value2', got %q", result["key2"])
	}
}

func TestReadFull(t *testing.T) {
	input := "Hello, World!"
	reader := bufio.NewReader(strings.NewReader(input))
	buf := make([]byte, 13)

	n, err := readFull(reader, buf)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if n != 13 {
		t.Errorf("expected to read 13 bytes, got %d", n)
	}
	if string(buf) != input {
		t.Errorf("expected %q, got %q", input, string(buf))
	}
}
