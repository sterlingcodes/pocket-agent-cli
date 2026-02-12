package speedtest

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/spf13/cobra"

	"github.com/unstablemind/pocket/pkg/output"
)

var httpClient = &http.Client{Timeout: 60 * time.Second}

const (
	cfDownloadURL = "https://speed.cloudflare.com/__down?bytes=%d"
	cfUploadURL   = "https://speed.cloudflare.com/__up"
	cfMetaURL     = "https://speed.cloudflare.com/meta"
)

// SpeedResult is the LLM-friendly speed test result
type SpeedResult struct {
	Download    *BandwidthResult `json:"download,omitempty"`
	Upload      *BandwidthResult `json:"upload,omitempty"`
	Latency     *LatencyResult   `json:"latency,omitempty"`
	ServerInfo  map[string]any   `json:"server_info,omitempty"`
	TestedAt    string           `json:"tested_at"`
	TestType    string           `json:"test_type"`
	DurationSec float64          `json:"duration_sec"`
}

// BandwidthResult holds bandwidth measurement
type BandwidthResult struct {
	SpeedMbps  float64 `json:"speed_mbps"`
	Bytes      int64   `json:"bytes"`
	DurationMs int64   `json:"duration_ms"`
}

// LatencyResult holds latency measurement
type LatencyResult struct {
	MinMs float64 `json:"min_ms"`
	AvgMs float64 `json:"avg_ms"`
	MaxMs float64 `json:"max_ms"`
}

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "speedtest",
		Aliases: []string{"speed", "st"},
		Short:   "Internet speed test (via Cloudflare)",
	}

	cmd.AddCommand(newRunCmd())
	cmd.AddCommand(newDownloadCmd())
	cmd.AddCommand(newUploadCmd())
	cmd.AddCommand(newLatencyCmd())

	return cmd
}

func newRunCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "run",
		Short: "Run full speed test (download + upload + latency)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runFullTest()
		},
	}
}

func newDownloadCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "download",
		Short: "Test download speed only",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDownloadTest()
		},
	}
}

func newUploadCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "upload",
		Short: "Test upload speed only",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUploadTest()
		},
	}
}

func newLatencyCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "latency",
		Short: "Test latency only",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLatencyTest()
		},
	}
}

func runFullTest() error {
	start := time.Now()

	latency, err := measureLatency()
	if err != nil {
		return output.PrintError("latency_error", err.Error(), nil)
	}

	dl, err := measureDownload()
	if err != nil {
		return output.PrintError("download_error", err.Error(), nil)
	}

	ul, err := measureUpload()
	if err != nil {
		return output.PrintError("upload_error", err.Error(), nil)
	}

	result := SpeedResult{
		Download:    dl,
		Upload:      ul,
		Latency:     latency,
		TestedAt:    time.Now().Format(time.RFC3339),
		TestType:    "full",
		DurationSec: time.Since(start).Seconds(),
	}

	return output.Print(result)
}

func runDownloadTest() error {
	start := time.Now()
	dl, err := measureDownload()
	if err != nil {
		return output.PrintError("download_error", err.Error(), nil)
	}
	return output.Print(SpeedResult{
		Download:    dl,
		TestedAt:    time.Now().Format(time.RFC3339),
		TestType:    "download",
		DurationSec: time.Since(start).Seconds(),
	})
}

func runUploadTest() error {
	start := time.Now()
	ul, err := measureUpload()
	if err != nil {
		return output.PrintError("upload_error", err.Error(), nil)
	}
	return output.Print(SpeedResult{
		Upload:      ul,
		TestedAt:    time.Now().Format(time.RFC3339),
		TestType:    "upload",
		DurationSec: time.Since(start).Seconds(),
	})
}

func runLatencyTest() error {
	start := time.Now()
	lat, err := measureLatency()
	if err != nil {
		return output.PrintError("latency_error", err.Error(), nil)
	}
	return output.Print(SpeedResult{
		Latency:     lat,
		TestedAt:    time.Now().Format(time.RFC3339),
		TestType:    "latency",
		DurationSec: time.Since(start).Seconds(),
	})
}

func measureLatency() (*LatencyResult, error) {
	var times []float64
	// Small download to measure latency (1 byte)
	for i := 0; i < 5; i++ {
		start := time.Now()
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf(cfDownloadURL, 1), http.NoBody)
		if err != nil {
			cancel()
			return nil, err
		}
		resp, err := httpClient.Do(req)
		if err != nil {
			cancel()
			return nil, fmt.Errorf("latency test failed: %w", err)
		}
		_, _ = io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		cancel()
		elapsed := float64(time.Since(start).Microseconds()) / 1000.0
		times = append(times, elapsed)
	}

	var minT, maxT, sumT float64
	minT = times[0]
	maxT = times[0]
	for _, t := range times {
		sumT += t
		if t < minT {
			minT = t
		}
		if t > maxT {
			maxT = t
		}
	}

	return &LatencyResult{
		MinMs: minT,
		AvgMs: sumT / float64(len(times)),
		MaxMs: maxT,
	}, nil
}

func measureDownload() (*BandwidthResult, error) {
	// Test with 25MB download
	testBytes := int64(25 * 1024 * 1024)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf(cfDownloadURL, testBytes), http.NoBody)
	if err != nil {
		return nil, err
	}

	start := time.Now()
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download test failed: %w", err)
	}
	defer resp.Body.Close()

	n, err := io.Copy(io.Discard, resp.Body)
	if err != nil {
		return nil, fmt.Errorf("download read failed: %w", err)
	}
	elapsed := time.Since(start)

	speedMbps := float64(n*8) / elapsed.Seconds() / 1_000_000

	return &BandwidthResult{
		SpeedMbps:  speedMbps,
		Bytes:      n,
		DurationMs: elapsed.Milliseconds(),
	}, nil
}

func measureUpload() (*BandwidthResult, error) {
	// Test with 10MB upload
	testBytes := int64(10 * 1024 * 1024)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	reader := io.LimitReader(&zeroReader{}, testBytes)

	req, err := http.NewRequestWithContext(ctx, "POST", cfUploadURL, reader)
	if err != nil {
		return nil, err
	}
	req.ContentLength = testBytes

	start := time.Now()
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("upload test failed: %w", err)
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)

	elapsed := time.Since(start)
	speedMbps := float64(testBytes*8) / elapsed.Seconds() / 1_000_000

	return &BandwidthResult{
		SpeedMbps:  speedMbps,
		Bytes:      testBytes,
		DurationMs: elapsed.Milliseconds(),
	}, nil
}

// zeroReader produces zero bytes for upload tests
type zeroReader struct{}

func (z *zeroReader) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = 0
	}
	return len(p), nil
}
