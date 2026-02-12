package dnsbench

import (
	"context"
	"fmt"
	"net"
	"sort"
	"time"

	"github.com/spf13/cobra"

	"github.com/unstablemind/pocket/pkg/output"
)

// Resolver represents a DNS resolver to benchmark
type Resolver struct {
	Name    string `json:"name"`
	Address string `json:"address"`
}

// BenchResult is the LLM-friendly benchmark result for a single resolver
type BenchResult struct {
	Name        string  `json:"name"`
	Address     string  `json:"address"`
	AvgMs       float64 `json:"avg_ms"`
	MinMs       float64 `json:"min_ms"`
	MaxMs       float64 `json:"max_ms"`
	SuccessRate float64 `json:"success_rate"`
	Queries     int     `json:"queries"`
	Failures    int     `json:"failures"`
}

// BenchReport is the full benchmark report
type BenchReport struct {
	Resolvers  []BenchResult `json:"resolvers"`
	Fastest    string        `json:"fastest"`
	TestDomain string        `json:"test_domain"`
	TestedAt   string        `json:"tested_at"`
}

// TestResult is the result of testing a single resolver
type TestResult struct {
	Name    string        `json:"name"`
	Address string        `json:"address"`
	AvgMs   float64       `json:"avg_ms"`
	Results []QueryResult `json:"results"`
}

// QueryResult is a single query result
type QueryResult struct {
	Domain string   `json:"domain"`
	TimeMs float64  `json:"time_ms"`
	IPs    []string `json:"ips,omitempty"`
	Error  string   `json:"error,omitempty"`
}

var defaultResolvers = []Resolver{
	{Name: "Cloudflare", Address: "1.1.1.1:53"},
	{Name: "Cloudflare-2", Address: "1.0.0.1:53"},
	{Name: "Google", Address: "8.8.8.8:53"},
	{Name: "Google-2", Address: "8.8.4.4:53"},
	{Name: "Quad9", Address: "9.9.9.9:53"},
	{Name: "OpenDNS", Address: "208.67.222.222:53"},
	{Name: "AdGuard", Address: "94.140.14.14:53"},
	{Name: "CleanBrowsing", Address: "185.228.168.9:53"},
}

var testDomains = []string{
	"google.com",
	"facebook.com",
	"amazon.com",
	"apple.com",
	"microsoft.com",
	"github.com",
	"cloudflare.com",
	"wikipedia.org",
	"reddit.com",
	"netflix.com",
}

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "dnsbench",
		Aliases: []string{"dnsb", "dns-bench"},
		Short:   "DNS resolver benchmark",
	}

	cmd.AddCommand(newRunCmd())
	cmd.AddCommand(newTestCmd())

	return cmd
}

func newRunCmd() *cobra.Command {
	var queries int

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Benchmark all popular DNS resolvers",
		Long:  "Benchmark popular public DNS resolvers to find the fastest one from your location.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBenchmark(queries)
		},
	}

	cmd.Flags().IntVarP(&queries, "queries", "q", 5, "Number of queries per resolver")

	return cmd
}

func newTestCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "test [resolver-ip]",
		Short: "Test a specific DNS resolver",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return testResolver(args[0])
		},
	}
}

func runBenchmark(queriesPerResolver int) error {
	results := make([]BenchResult, 0, len(defaultResolvers))

	for _, resolver := range defaultResolvers {
		result := benchmarkResolver(resolver, queriesPerResolver)
		results = append(results, result)
	}

	sort.Slice(results, func(i, j int) bool {
		// Sort by success rate first, then by avg latency
		if results[i].SuccessRate != results[j].SuccessRate {
			return results[i].SuccessRate > results[j].SuccessRate
		}
		return results[i].AvgMs < results[j].AvgMs
	})

	fastest := ""
	if len(results) > 0 && results[0].SuccessRate > 0 {
		fastest = results[0].Name
	}

	report := BenchReport{
		Resolvers:  results,
		Fastest:    fastest,
		TestDomain: "multiple domains",
		TestedAt:   time.Now().Format(time.RFC3339),
	}

	return output.Print(report)
}

func benchmarkResolver(resolver Resolver, numQueries int) BenchResult {
	var times []float64
	failures := 0

	r := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{Timeout: 5 * time.Second}
			return d.DialContext(ctx, "udp", resolver.Address)
		},
	}

	for i := 0; i < numQueries; i++ {
		domain := testDomains[i%len(testDomains)]

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		start := time.Now()
		_, err := r.LookupHost(ctx, domain)
		elapsed := float64(time.Since(start).Microseconds()) / 1000.0
		cancel()

		if err != nil {
			failures++
		} else {
			times = append(times, elapsed)
		}
	}

	if len(times) == 0 {
		return BenchResult{
			Name:        resolver.Name,
			Address:     resolver.Address,
			SuccessRate: 0,
			Queries:     numQueries,
			Failures:    failures,
		}
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

	return BenchResult{
		Name:        resolver.Name,
		Address:     resolver.Address,
		AvgMs:       sumT / float64(len(times)),
		MinMs:       minT,
		MaxMs:       maxT,
		SuccessRate: float64(numQueries-failures) / float64(numQueries) * 100,
		Queries:     numQueries,
		Failures:    failures,
	}
}

func testResolver(address string) error {
	if _, _, err := net.SplitHostPort(address); err != nil {
		address += ":53"
	}

	resolver := Resolver{Name: "custom", Address: address}

	r := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, addr string) (net.Conn, error) {
			d := net.Dialer{Timeout: 5 * time.Second}
			return d.DialContext(ctx, "udp", resolver.Address)
		},
	}

	queryResults := make([]QueryResult, 0, len(testDomains))
	var totalMs float64

	for _, domain := range testDomains {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		start := time.Now()
		ips, err := r.LookupHost(ctx, domain)
		elapsed := float64(time.Since(start).Microseconds()) / 1000.0
		cancel()

		qr := QueryResult{
			Domain: domain,
			TimeMs: elapsed,
		}

		if err != nil {
			qr.Error = fmt.Sprintf("%v", err)
		} else {
			qr.IPs = ips
			totalMs += elapsed
		}

		queryResults = append(queryResults, qr)
	}

	successCount := 0
	for _, qr := range queryResults {
		if qr.Error == "" {
			successCount++
		}
	}

	avgMs := 0.0
	if successCount > 0 {
		avgMs = totalMs / float64(successCount)
	}

	return output.Print(TestResult{
		Name:    resolver.Name,
		Address: resolver.Address,
		AvgMs:   avgMs,
		Results: queryResults,
	})
}
