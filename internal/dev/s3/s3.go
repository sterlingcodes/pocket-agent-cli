package s3

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/unstablemind/pocket/internal/common/config"
	"github.com/unstablemind/pocket/pkg/output"
)

// BucketsResult holds the list of S3 buckets
type BucketsResult struct {
	Buckets []Bucket `json:"buckets"`
	Count   int      `json:"count"`
}

// Bucket holds S3 bucket metadata
type Bucket struct {
	Name      string `json:"name"`
	CreatedAt string `json:"created_at"`
}

// ListResult holds directory listing results
type ListResult struct {
	Path    string   `json:"path"`
	Objects []Object `json:"objects"`
	Count   int      `json:"count"`
}

// Object holds a single S3 object entry
type Object struct {
	Key          string `json:"key"`
	Size         string `json:"size,omitempty"`
	LastModified string `json:"last_modified,omitempty"`
	IsPrefix     bool   `json:"is_prefix"`
}

// DownloadResult holds download operation result
type DownloadResult struct {
	Source      string `json:"source"`
	Destination string `json:"destination"`
	OK          bool   `json:"ok"`
}

// UploadResult holds upload operation result
type UploadResult struct {
	Source      string `json:"source"`
	Destination string `json:"destination"`
	OK          bool   `json:"ok"`
}

// PresignResult holds presigned URL result
type PresignResult struct {
	URL       string `json:"url"`
	Path      string `json:"path"`
	ExpiresIn int    `json:"expires_in"`
}

// NewCmd returns the S3 command
func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "s3",
		Aliases: []string{"aws-s3", "storage"},
		Short:   "S3-compatible storage commands (requires AWS CLI)",
	}

	cmd.AddCommand(newBucketsCmd())
	cmd.AddCommand(newLsCmd())
	cmd.AddCommand(newGetCmd())
	cmd.AddCommand(newPutCmd())
	cmd.AddCommand(newPresignCmd())

	return cmd
}

func checkAWSCLI() error {
	_, err := exec.LookPath("aws")
	if err != nil {
		return output.PrintError("aws_not_found", "AWS CLI is not installed or not in PATH", map[string]any{
			"hint": "Install from https://aws.amazon.com/cli/ or via: brew install awscli",
		})
	}
	return nil
}

func getAWSArgs() []string {
	var extraArgs []string

	profile, err := config.Get("aws_profile")
	if err == nil && profile != "" {
		extraArgs = append(extraArgs, "--profile", profile)
	}

	region, err := config.Get("aws_region")
	if err == nil && region != "" {
		extraArgs = append(extraArgs, "--region", region)
	}

	return extraArgs
}

func runAWS(args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	allArgs := append([]string{}, args...)
	allArgs = append(allArgs, getAWSArgs()...)
	cmd := exec.CommandContext(ctx, "aws", allArgs...)

	out, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			stderr := strings.TrimSpace(string(exitErr.Stderr))
			if stderr != "" {
				return nil, fmt.Errorf("aws CLI error: %s", stderr)
			}
		}
		return nil, fmt.Errorf("aws CLI error: %s", err.Error())
	}

	return out, nil
}

func newBucketsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "buckets",
		Short: "List S3 buckets",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := checkAWSCLI(); err != nil {
				return err
			}

			data, err := runAWS("s3api", "list-buckets", "--output", "json")
			if err != nil {
				return output.PrintError("aws_error", err.Error(), nil)
			}

			var resp struct {
				Buckets []struct {
					Name         string `json:"Name"`
					CreationDate string `json:"CreationDate"`
				} `json:"Buckets"`
			}

			if err := json.Unmarshal(data, &resp); err != nil {
				return output.PrintError("parse_failed", fmt.Sprintf("Failed to parse AWS response: %s", err.Error()), nil)
			}

			buckets := make([]Bucket, 0, len(resp.Buckets))
			for _, b := range resp.Buckets {
				buckets = append(buckets, Bucket{
					Name:      b.Name,
					CreatedAt: formatTime(b.CreationDate),
				})
			}

			result := BucketsResult{
				Buckets: buckets,
				Count:   len(buckets),
			}

			return output.Print(result)
		},
	}

	return cmd
}

func newLsCmd() *cobra.Command {
	var recursive bool

	cmd := &cobra.Command{
		Use:   "ls [s3-path]",
		Short: "List objects in S3 path",
		Long:  `List objects. Path format: "s3://bucket/prefix/"`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := checkAWSCLI(); err != nil {
				return err
			}

			s3Path := args[0]
			awsArgs := []string{"s3", "ls", s3Path}
			if recursive {
				awsArgs = append(awsArgs, "--recursive")
			}

			data, err := runAWS(awsArgs...)
			if err != nil {
				return output.PrintError("aws_error", err.Error(), nil)
			}

			objects := parseLsOutput(string(data))

			result := ListResult{
				Path:    s3Path,
				Objects: objects,
				Count:   len(objects),
			}

			return output.Print(result)
		},
	}

	cmd.Flags().BoolVarP(&recursive, "recursive", "r", false, "List recursively")

	return cmd
}

func newGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get [s3-path] [local-path]",
		Short: "Download file from S3",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := checkAWSCLI(); err != nil {
				return err
			}

			s3Path := args[0]
			localPath := args[1]

			_, err := runAWS("s3", "cp", s3Path, localPath)
			if err != nil {
				return output.PrintError("download_failed", err.Error(), nil)
			}

			result := DownloadResult{
				Source:      s3Path,
				Destination: localPath,
				OK:          true,
			}

			return output.Print(result)
		},
	}

	return cmd
}

func newPutCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "put [local-path] [s3-path]",
		Short: "Upload file to S3",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := checkAWSCLI(); err != nil {
				return err
			}

			localPath := args[0]
			s3Path := args[1]

			_, err := runAWS("s3", "cp", localPath, s3Path)
			if err != nil {
				return output.PrintError("upload_failed", err.Error(), nil)
			}

			result := UploadResult{
				Source:      localPath,
				Destination: s3Path,
				OK:          true,
			}

			return output.Print(result)
		},
	}

	return cmd
}

func newPresignCmd() *cobra.Command {
	var expires int

	cmd := &cobra.Command{
		Use:   "presign [s3-path]",
		Short: "Generate a presigned URL for an S3 object",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := checkAWSCLI(); err != nil {
				return err
			}

			s3Path := args[0]

			data, err := runAWS("s3", "presign", s3Path, "--expires-in", fmt.Sprintf("%d", expires))
			if err != nil {
				return output.PrintError("presign_failed", err.Error(), nil)
			}

			presignedURL := strings.TrimSpace(string(data))

			result := PresignResult{
				URL:       presignedURL,
				Path:      s3Path,
				ExpiresIn: expires,
			}

			return output.Print(result)
		},
	}

	cmd.Flags().IntVar(&expires, "expires", 3600, "URL expiration time in seconds")

	return cmd
}

func parseLsOutput(output string) []Object {
	var objects []Object
	lines := strings.Split(strings.TrimSpace(output), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Directory prefix line: "                           PRE prefix/"
		if strings.Contains(line, "PRE ") {
			parts := strings.SplitN(line, "PRE ", 2)
			if len(parts) == 2 {
				objects = append(objects, Object{
					Key:      strings.TrimSpace(parts[1]),
					IsPrefix: true,
				})
				continue
			}
		}

		// Object line: "2024-01-01 12:00:00    12345 filename"
		// Format: date(10) space time(8) space+ size space+ key
		if len(line) < 20 {
			continue
		}

		// Split into at most 4 parts: date, time, size, key
		fields := strings.Fields(line)
		if len(fields) >= 4 {
			dateStr := fields[0]
			timeStr := fields[1]
			sizeStr := fields[2]
			key := strings.Join(fields[3:], " ")

			objects = append(objects, Object{
				Key:          key,
				Size:         formatBytes(sizeStr),
				LastModified: dateStr + " " + timeStr,
				IsPrefix:     false,
			})
		}
	}

	if objects == nil {
		objects = []Object{}
	}

	return objects
}

func formatTime(isoTime string) string {
	if isoTime == "" {
		return ""
	}
	t, err := time.Parse(time.RFC3339, isoTime)
	if err != nil {
		// Try alternate format
		t, err = time.Parse("2006-01-02T15:04:05+00:00", isoTime)
		if err != nil {
			return isoTime
		}
	}
	return t.Format("2006-01-02 15:04:05")
}

func formatBytes(sizeStr string) string {
	var size int64
	if _, err := fmt.Sscanf(sizeStr, "%d", &size); err != nil {
		return sizeStr
	}

	switch {
	case size >= 1<<30:
		return fmt.Sprintf("%.1f GB", float64(size)/float64(1<<30))
	case size >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(size)/float64(1<<20))
	case size >= 1<<10:
		return fmt.Sprintf("%.1f KB", float64(size)/float64(1<<10))
	default:
		return fmt.Sprintf("%d B", size)
	}
}
