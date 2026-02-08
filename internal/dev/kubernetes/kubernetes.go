package kubernetes

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os/exec"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/unstablemind/pocket/pkg/output"
)

// Pod is LLM-friendly pod output
type Pod struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Status    string `json:"status"`
	Ready     string `json:"ready"`
	Restarts  int    `json:"restarts"`
	Age       string `json:"age"`
	IP        string `json:"ip,omitempty"`
}

// LogResult is LLM-friendly log output
type LogResult struct {
	Pod       string   `json:"pod"`
	Namespace string   `json:"namespace"`
	Lines     []string `json:"lines"`
	LineCount int      `json:"line_count"`
}

// Deployment is LLM-friendly deployment output
type Deployment struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Ready     string `json:"ready"`
	UpToDate  int    `json:"up_to_date"`
	Available int    `json:"available"`
	Age       string `json:"age"`
}

// Service is LLM-friendly service output
type Service struct {
	Name       string `json:"name"`
	Namespace  string `json:"namespace"`
	Type       string `json:"type"`
	ClusterIP  string `json:"cluster_ip"`
	ExternalIP string `json:"external_ip,omitempty"`
	Ports      string `json:"ports"`
	Age        string `json:"age"`
}

// DescribeResult is LLM-friendly describe output
type DescribeResult struct {
	Resource  string `json:"resource"`
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Output    string `json:"output"`
}

// NewCmd returns the kubernetes parent command
func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "kube",
		Aliases: []string{"k8s", "kubernetes"},
		Short:   "Kubernetes commands",
	}

	cmd.AddCommand(newPodsCmd())
	cmd.AddCommand(newLogsCmd())
	cmd.AddCommand(newDeploymentsCmd())
	cmd.AddCommand(newServicesCmd())
	cmd.AddCommand(newDescribeCmd())

	return cmd
}

func kubectlAvailable() error {
	_, err := exec.LookPath("kubectl")
	if err != nil {
		return output.PrintError("kubectl_not_found", "kubectl is not installed or not in PATH", nil)
	}
	return nil
}

func runKubectl(ctx context.Context, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "kubectl", args...)
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("kubectl error: %s", strings.TrimSpace(string(exitErr.Stderr)))
		}
		return nil, fmt.Errorf("kubectl failed: %w", err)
	}
	return out, nil
}

func formatAge(t time.Time) string {
	diff := time.Since(t)
	if diff < 0 {
		return "0s"
	}

	switch {
	case diff < time.Minute:
		return fmt.Sprintf("%ds", int(diff.Seconds()))
	case diff < time.Hour:
		return fmt.Sprintf("%dm", int(diff.Minutes()))
	case diff < 24*time.Hour:
		return fmt.Sprintf("%dh", int(diff.Hours()))
	default:
		days := int(math.Floor(diff.Hours() / 24))
		return fmt.Sprintf("%dd", days)
	}
}

func newPodsCmd() *cobra.Command {
	var namespace string
	var all bool

	cmd := &cobra.Command{
		Use:   "pods",
		Short: "List Kubernetes pods",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := kubectlAvailable(); err != nil {
				return err
			}

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			kubectlArgs := []string{"get", "pods", "-o", "json"}
			if all {
				kubectlArgs = append(kubectlArgs, "--all-namespaces")
			} else {
				kubectlArgs = append(kubectlArgs, "--namespace", namespace)
			}

			out, err := runKubectl(ctx, kubectlArgs...)
			if err != nil {
				return output.PrintError("kubectl_failed", err.Error(), nil)
			}

			var podList map[string]any
			if err := json.Unmarshal(out, &podList); err != nil {
				return output.PrintError("parse_failed", fmt.Sprintf("Failed to parse kubectl output: %s", err.Error()), nil)
			}

			items, _ := podList["items"].([]any)
			pods := make([]Pod, 0, len(items))

			for _, item := range items {
				m, ok := item.(map[string]any)
				if !ok {
					continue
				}

				metadata, _ := m["metadata"].(map[string]any)
				status, _ := m["status"].(map[string]any)

				pod := Pod{
					Name:      getString(metadata, "name"),
					Namespace: getString(metadata, "namespace"),
					Status:    getString(status, "phase"),
					IP:        getString(status, "podIP"),
				}

				// Calculate age from creationTimestamp
				if creationTS := getString(metadata, "creationTimestamp"); creationTS != "" {
					if t, err := time.Parse(time.RFC3339, creationTS); err == nil {
						pod.Age = formatAge(t)
					}
				}

				// Calculate ready count and restarts from containerStatuses
				containerStatuses, _ := status["containerStatuses"].([]any)
				readyCount := 0
				totalCount := len(containerStatuses)
				totalRestarts := 0

				for _, cs := range containerStatuses {
					if container, ok := cs.(map[string]any); ok {
						if ready, ok := container["ready"].(bool); ok && ready {
							readyCount++
						}
						if restartCount, ok := container["restartCount"].(float64); ok {
							totalRestarts += int(restartCount)
						}
					}
				}

				// If no containerStatuses, check spec.containers for total count
				if totalCount == 0 {
					if spec, ok := m["spec"].(map[string]any); ok {
						if containers, ok := spec["containers"].([]any); ok {
							totalCount = len(containers)
						}
					}
				}

				pod.Ready = fmt.Sprintf("%d/%d", readyCount, totalCount)
				pod.Restarts = totalRestarts

				pods = append(pods, pod)
			}

			return output.Print(pods)
		},
	}

	cmd.Flags().StringVarP(&namespace, "namespace", "n", "default", "Kubernetes namespace")
	cmd.Flags().BoolVarP(&all, "all", "a", false, "All namespaces")

	return cmd
}

func newLogsCmd() *cobra.Command {
	var namespace string
	var tail int
	var container string

	cmd := &cobra.Command{
		Use:   "logs [pod-name]",
		Short: "Get logs from a Kubernetes pod",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := kubectlAvailable(); err != nil {
				return err
			}

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			podName := args[0]
			kubectlArgs := []string{"logs", podName, "-n", namespace, fmt.Sprintf("--tail=%d", tail)}
			if container != "" {
				kubectlArgs = append(kubectlArgs, "-c", container)
			}

			out, err := runKubectl(ctx, kubectlArgs...)
			if err != nil {
				return output.PrintError("kubectl_failed", err.Error(), nil)
			}

			rawLines := strings.Split(string(out), "\n")
			// Remove trailing empty line from split
			lines := make([]string, 0, len(rawLines))
			for _, line := range rawLines {
				if line != "" {
					lines = append(lines, line)
				}
			}

			return output.Print(LogResult{
				Pod:       podName,
				Namespace: namespace,
				Lines:     lines,
				LineCount: len(lines),
			})
		},
	}

	cmd.Flags().StringVarP(&namespace, "namespace", "n", "default", "Kubernetes namespace")
	cmd.Flags().IntVarP(&tail, "tail", "t", 100, "Number of log lines")
	cmd.Flags().StringVarP(&container, "container", "c", "", "Container name (optional)")

	return cmd
}

func newDeploymentsCmd() *cobra.Command {
	var namespace string
	var all bool

	cmd := &cobra.Command{
		Use:     "deployments",
		Aliases: []string{"deploy"},
		Short:   "List Kubernetes deployments",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := kubectlAvailable(); err != nil {
				return err
			}

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			kubectlArgs := []string{"get", "deployments", "-o", "json"}
			if all {
				kubectlArgs = append(kubectlArgs, "--all-namespaces")
			} else {
				kubectlArgs = append(kubectlArgs, "-n", namespace)
			}

			out, err := runKubectl(ctx, kubectlArgs...)
			if err != nil {
				return output.PrintError("kubectl_failed", err.Error(), nil)
			}

			var depList map[string]any
			if err := json.Unmarshal(out, &depList); err != nil {
				return output.PrintError("parse_failed", fmt.Sprintf("Failed to parse kubectl output: %s", err.Error()), nil)
			}

			items, _ := depList["items"].([]any)
			deployments := make([]Deployment, 0, len(items))

			for _, item := range items {
				m, ok := item.(map[string]any)
				if !ok {
					continue
				}

				metadata, _ := m["metadata"].(map[string]any)
				spec, _ := m["spec"].(map[string]any)
				status, _ := m["status"].(map[string]any)

				dep := Deployment{
					Name:      getString(metadata, "name"),
					Namespace: getString(metadata, "namespace"),
					UpToDate:  getInt(status, "updatedReplicas"),
					Available: getInt(status, "availableReplicas"),
				}

				// Calculate age from creationTimestamp
				if creationTS := getString(metadata, "creationTimestamp"); creationTS != "" {
					if t, err := time.Parse(time.RFC3339, creationTS); err == nil {
						dep.Age = formatAge(t)
					}
				}

				// Ready = readyReplicas / spec.replicas
				replicas := getInt(spec, "replicas")
				readyReplicas := getInt(status, "readyReplicas")
				dep.Ready = fmt.Sprintf("%d/%d", readyReplicas, replicas)

				deployments = append(deployments, dep)
			}

			return output.Print(deployments)
		},
	}

	cmd.Flags().StringVarP(&namespace, "namespace", "n", "default", "Kubernetes namespace")
	cmd.Flags().BoolVarP(&all, "all", "a", false, "All namespaces")

	return cmd
}

func newServicesCmd() *cobra.Command {
	var namespace string
	var all bool

	cmd := &cobra.Command{
		Use:     "services",
		Aliases: []string{"svc"},
		Short:   "List Kubernetes services",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := kubectlAvailable(); err != nil {
				return err
			}

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			kubectlArgs := []string{"get", "services", "-o", "json"}
			if all {
				kubectlArgs = append(kubectlArgs, "--all-namespaces")
			} else {
				kubectlArgs = append(kubectlArgs, "-n", namespace)
			}

			out, err := runKubectl(ctx, kubectlArgs...)
			if err != nil {
				return output.PrintError("kubectl_failed", err.Error(), nil)
			}

			var svcList map[string]any
			if err := json.Unmarshal(out, &svcList); err != nil {
				return output.PrintError("parse_failed", fmt.Sprintf("Failed to parse kubectl output: %s", err.Error()), nil)
			}

			items, _ := svcList["items"].([]any)
			services := make([]Service, 0, len(items))

			for _, item := range items {
				m, ok := item.(map[string]any)
				if !ok {
					continue
				}

				metadata, _ := m["metadata"].(map[string]any)
				spec, _ := m["spec"].(map[string]any)

				svc := Service{
					Name:      getString(metadata, "name"),
					Namespace: getString(metadata, "namespace"),
					Type:      getString(spec, "type"),
					ClusterIP: getString(spec, "clusterIP"),
				}

				// Calculate age from creationTimestamp
				if creationTS := getString(metadata, "creationTimestamp"); creationTS != "" {
					if t, err := time.Parse(time.RFC3339, creationTS); err == nil {
						svc.Age = formatAge(t)
					}
				}

				// Build external IP
				if lbIngress, ok := m["status"].(map[string]any); ok {
					if lb, ok := lbIngress["loadBalancer"].(map[string]any); ok {
						if ingress, ok := lb["ingress"].([]any); ok && len(ingress) > 0 {
							if first, ok := ingress[0].(map[string]any); ok {
								ip := getString(first, "ip")
								if ip == "" {
									ip = getString(first, "hostname")
								}
								svc.ExternalIP = ip
							}
						}
					}
				}

				// Also check spec.externalIPs
				if svc.ExternalIP == "" {
					if externalIPs, ok := spec["externalIPs"].([]any); ok && len(externalIPs) > 0 {
						ips := make([]string, 0, len(externalIPs))
						for _, ip := range externalIPs {
							if s, ok := ip.(string); ok {
								ips = append(ips, s)
							}
						}
						svc.ExternalIP = strings.Join(ips, ",")
					}
				}

				// Build ports string: "80/TCP,443/TCP"
				if ports, ok := spec["ports"].([]any); ok {
					portStrs := make([]string, 0, len(ports))
					for _, p := range ports {
						if port, ok := p.(map[string]any); ok {
							portNum := getInt(port, "port")
							protocol := getString(port, "protocol")
							if protocol == "" {
								protocol = "TCP"
							}
							portStrs = append(portStrs, fmt.Sprintf("%d/%s", portNum, protocol))
						}
					}
					svc.Ports = strings.Join(portStrs, ",")
				}

				services = append(services, svc)
			}

			return output.Print(services)
		},
	}

	cmd.Flags().StringVarP(&namespace, "namespace", "n", "default", "Kubernetes namespace")
	cmd.Flags().BoolVarP(&all, "all", "a", false, "All namespaces")

	return cmd
}

func newDescribeCmd() *cobra.Command {
	var namespace string

	cmd := &cobra.Command{
		Use:   "describe [resource] [name]",
		Short: "Describe a Kubernetes resource",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := kubectlAvailable(); err != nil {
				return err
			}

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			resource := args[0]
			name := args[1]

			out, err := runKubectl(ctx, "describe", resource, name, "-n", namespace)
			if err != nil {
				return output.PrintError("kubectl_failed", err.Error(), nil)
			}

			return output.Print(DescribeResult{
				Resource:  resource,
				Name:      name,
				Namespace: namespace,
				Output:    string(out),
			})
		},
	}

	cmd.Flags().StringVarP(&namespace, "namespace", "n", "default", "Kubernetes namespace")

	return cmd
}

func getString(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func getInt(m map[string]any, key string) int {
	if m == nil {
		return 0
	}
	if v, ok := m[key].(float64); ok {
		return int(v)
	}
	return 0
}
