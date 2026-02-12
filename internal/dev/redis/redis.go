package redis

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/unstablemind/pocket/internal/common/config"
	"github.com/unstablemind/pocket/pkg/output"
)

const redisNil = "(nil)"

// Value is LLM-friendly output for a GET result
type Value struct {
	Key   string `json:"key"`
	Value string `json:"value"`
	Type  string `json:"type"`
}

// SetResult is LLM-friendly output for a SET result
type SetResult struct {
	Key   string `json:"key"`
	Value string `json:"value"`
	TTL   int    `json:"ttl,omitempty"`
	OK    bool   `json:"ok"`
}

// DelResult is LLM-friendly output for a DEL result
type DelResult struct {
	Keys    []string `json:"keys"`
	Deleted int64    `json:"deleted"`
}

// Keys is LLM-friendly output for a KEYS result
type Keys struct {
	Pattern string   `json:"pattern"`
	Keys    []string `json:"keys"`
	Count   int      `json:"count"`
}

// Info is LLM-friendly output for an INFO result
type Info struct {
	Version          string `json:"version"`
	Mode             string `json:"mode"`
	OS               string `json:"os"`
	UptimeSeconds    int64  `json:"uptime_seconds"`
	ConnectedClients string `json:"connected_clients"`
	UsedMemory       string `json:"used_memory"`
	UsedMemoryHuman  string `json:"used_memory_human"`
}

// NewCmd returns the redis parent command
func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "redis",
		Aliases: []string{"rd"},
		Short:   "Redis commands",
	}

	cmd.AddCommand(newGetCmd())
	cmd.AddCommand(newSetCmd())
	cmd.AddCommand(newDelCmd())
	cmd.AddCommand(newKeysCmd())
	cmd.AddCommand(newInfoCmd())

	return cmd
}

func connect() (net.Conn, *bufio.Reader, error) {
	redisURL, err := config.Get("redis_url")
	if err != nil || redisURL == "" {
		redisURL = "localhost:6379"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var d net.Dialer
	conn, err := d.DialContext(ctx, "tcp", redisURL)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to Redis at %s: %w", redisURL, err)
	}

	reader := bufio.NewReader(conn)

	// Authenticate if password is configured
	password, passErr := config.Get("redis_password")
	if passErr == nil && password != "" {
		if err := conn.SetDeadline(time.Now().Add(10 * time.Second)); err != nil {
			conn.Close()
			return nil, nil, err
		}
		_, err := sendCommand(conn, reader, "AUTH", password)
		if err != nil {
			conn.Close()
			return nil, nil, fmt.Errorf("authentication failed: %w", err)
		}
	}

	return conn, reader, nil
}

func sendCommand(conn net.Conn, reader *bufio.Reader, args ...string) (string, error) {
	// Build RESP array
	buf := fmt.Sprintf("*%d\r\n", len(args))
	for _, arg := range args {
		buf += fmt.Sprintf("$%d\r\n%s\r\n", len(arg), arg)
	}

	if err := conn.SetDeadline(time.Now().Add(10 * time.Second)); err != nil {
		return "", err
	}

	_, err := conn.Write([]byte(buf))
	if err != nil {
		return "", fmt.Errorf("write failed: %w", err)
	}

	return readResponse(reader)
}

const maxRESPDepth = 64

func readResponse(reader *bufio.Reader) (string, error) {
	return readResponseDepth(reader, 0)
}

//nolint:gocyclo // complex but clear sequential logic
func readResponseDepth(reader *bufio.Reader, depth int) (string, error) {
	if depth > maxRESPDepth {
		return "", fmt.Errorf("RESP nesting depth exceeded maximum of %d", maxRESPDepth)
	}
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("read failed: %w", err)
	}
	line = strings.TrimRight(line, "\r\n")

	if line == "" {
		return "", fmt.Errorf("empty response")
	}

	prefix := line[0]
	payload := line[1:]

	switch prefix {
	case '+':
		// Simple string
		return payload, nil

	case '-':
		// Error
		return "", fmt.Errorf("redis error: %s", payload)

	case ':':
		// Integer
		return payload, nil

	case '$':
		// Bulk string
		length, err := strconv.Atoi(payload)
		if err != nil {
			return "", fmt.Errorf("invalid bulk string length: %s", payload)
		}
		if length == -1 {
			return redisNil, nil
		}
		data := make([]byte, length+2) // +2 for \r\n
		_, err = readFull(reader, data)
		if err != nil {
			return "", fmt.Errorf("read bulk data failed: %w", err)
		}
		return string(data[:length]), nil

	case '*':
		// Array
		count, err := strconv.Atoi(payload)
		if err != nil {
			return "", fmt.Errorf("invalid array length: %s", payload)
		}
		if count == -1 {
			return redisNil, nil
		}
		var parts []string
		for i := 0; i < count; i++ {
			element, err := readResponseDepth(reader, depth+1)
			if err != nil {
				return "", err
			}
			parts = append(parts, element)
		}
		return strings.Join(parts, "\n"), nil

	default:
		return "", fmt.Errorf("unknown RESP type: %c", prefix)
	}
}

func readFull(reader *bufio.Reader, buf []byte) (int, error) {
	total := 0
	for total < len(buf) {
		n, err := reader.Read(buf[total:])
		total += n
		if err != nil {
			return total, err
		}
	}
	return total, nil
}

func sendCommandArray(conn net.Conn, reader *bufio.Reader, args ...string) ([]string, error) {
	// Build RESP array
	buf := fmt.Sprintf("*%d\r\n", len(args))
	for _, arg := range args {
		buf += fmt.Sprintf("$%d\r\n%s\r\n", len(arg), arg)
	}

	if err := conn.SetDeadline(time.Now().Add(10 * time.Second)); err != nil {
		return nil, err
	}

	_, err := conn.Write([]byte(buf))
	if err != nil {
		return nil, fmt.Errorf("write failed: %w", err)
	}

	line, err := reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("read failed: %w", err)
	}
	line = strings.TrimRight(line, "\r\n")

	if line == "" {
		return nil, fmt.Errorf("empty response")
	}

	prefix := line[0]
	payload := line[1:]

	switch prefix {
	case '*':
		count, err := strconv.Atoi(payload)
		if err != nil {
			return nil, fmt.Errorf("invalid array length: %s", payload)
		}
		if count == -1 {
			return []string{}, nil
		}
		result := make([]string, 0, count)
		for i := 0; i < count; i++ {
			element, err := readResponse(reader)
			if err != nil {
				return nil, err
			}
			result = append(result, element)
		}
		return result, nil

	case '-':
		return nil, fmt.Errorf("redis error: %s", payload)

	case '+':
		return []string{payload}, nil

	default:
		// For non-array responses, read via readResponse
		resp := string(prefix) + payload
		return []string{resp}, nil
	}
}

func newGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get [key]",
		Short: "Get a value from Redis",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			conn, reader, err := connect()
			if err != nil {
				return output.PrintError("connection_failed", err.Error(), nil)
			}
			defer conn.Close()

			key := args[0]
			value, err := sendCommand(conn, reader, "GET", key)
			if err != nil {
				return output.PrintError("command_failed", err.Error(), nil)
			}

			valType := "string"
			if value == redisNil {
				valType = "none"
			}

			return output.Print(Value{
				Key:   key,
				Value: value,
				Type:  valType,
			})
		},
	}

	return cmd
}

func newSetCmd() *cobra.Command {
	var ttl int

	cmd := &cobra.Command{
		Use:   "set [key] [value]",
		Short: "Set a value in Redis",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			conn, reader, err := connect()
			if err != nil {
				return output.PrintError("connection_failed", err.Error(), nil)
			}
			defer conn.Close()

			key := args[0]
			value := args[1]

			var resp string
			if ttl > 0 {
				resp, err = sendCommand(conn, reader, "SETEX", key, strconv.Itoa(ttl), value)
			} else {
				resp, err = sendCommand(conn, reader, "SET", key, value)
			}
			if err != nil {
				return output.PrintError("command_failed", err.Error(), nil)
			}

			return output.Print(SetResult{
				Key:   key,
				Value: value,
				TTL:   ttl,
				OK:    resp == "OK",
			})
		},
	}

	cmd.Flags().IntVar(&ttl, "ttl", 0, "TTL in seconds (optional)")

	return cmd
}

func newDelCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "del [key...]",
		Short: "Delete one or more keys from Redis",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			conn, reader, err := connect()
			if err != nil {
				return output.PrintError("connection_failed", err.Error(), nil)
			}
			defer conn.Close()

			cmdArgs := append([]string{"DEL"}, args...)
			resp, err := sendCommand(conn, reader, cmdArgs...)
			if err != nil {
				return output.PrintError("command_failed", err.Error(), nil)
			}

			deleted, _ := strconv.ParseInt(resp, 10, 64)

			return output.Print(DelResult{
				Keys:    args,
				Deleted: deleted,
			})
		},
	}

	return cmd
}

func newKeysCmd() *cobra.Command {
	var limit int

	cmd := &cobra.Command{
		Use:   "keys [pattern]",
		Short: "List keys matching a pattern",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			conn, reader, err := connect()
			if err != nil {
				return output.PrintError("connection_failed", err.Error(), nil)
			}
			defer conn.Close()

			pattern := "*"
			if len(args) > 0 {
				pattern = args[0]
			}

			keys, err := sendCommandArray(conn, reader, "KEYS", pattern)
			if err != nil {
				return output.PrintError("command_failed", err.Error(), nil)
			}

			// Limit the number of keys shown
			if len(keys) > limit {
				keys = keys[:limit]
			}

			return output.Print(Keys{
				Pattern: pattern,
				Keys:    keys,
				Count:   len(keys),
			})
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "l", 100, "Maximum number of keys to show")

	return cmd
}

func newInfoCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "info",
		Short: "Get Redis server information",
		RunE: func(cmd *cobra.Command, args []string) error {
			conn, reader, err := connect()
			if err != nil {
				return output.PrintError("connection_failed", err.Error(), nil)
			}
			defer conn.Close()

			resp, err := sendCommand(conn, reader, "INFO", "server")
			if err != nil {
				return output.PrintError("command_failed", err.Error(), nil)
			}

			info := parseInfo(resp)

			// Also get clients and memory info
			clientsResp, err := sendCommand(conn, reader, "INFO", "clients")
			if err == nil {
				clientsInfo := parseInfo(clientsResp)
				if v, ok := clientsInfo["connected_clients"]; ok {
					info["connected_clients"] = v
				}
			}

			memResp, err := sendCommand(conn, reader, "INFO", "memory")
			if err == nil {
				memInfo := parseInfo(memResp)
				if v, ok := memInfo["used_memory"]; ok {
					info["used_memory"] = v
				}
				if v, ok := memInfo["used_memory_human"]; ok {
					info["used_memory_human"] = v
				}
			}

			uptime, _ := strconv.ParseInt(info["uptime_in_seconds"], 10, 64)

			return output.Print(Info{
				Version:          info["redis_version"],
				Mode:             info["redis_mode"],
				OS:               info["os"],
				UptimeSeconds:    uptime,
				ConnectedClients: info["connected_clients"],
				UsedMemory:       info["used_memory"],
				UsedMemoryHuman:  info["used_memory_human"],
			})
		},
	}

	return cmd
}

func parseInfo(raw string) map[string]string {
	result := make(map[string]string)
	lines := strings.Split(raw, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			result[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}
	return result
}
