//go:build linux

package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	dockernetwork "github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
)

const PaasNetwork = "paas_network"

var Client *client.Client

func Init() {
	c, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Fatalf("Failed to connect to Docker: %v", err)
	}
	Client = c
	ensureNetwork()
	log.Println("Docker client initialized")
}

func ensureNetwork() {
	ctx := context.Background()
	nets, err := Client.NetworkList(ctx, dockernetwork.ListOptions{
		Filters: filters.NewArgs(filters.Arg("name", PaasNetwork)),
	})
	if err != nil || len(nets) > 0 {
		return
	}
	_, err = Client.NetworkCreate(ctx, PaasNetwork, dockernetwork.CreateOptions{Driver: "bridge"})
	if err != nil {
		log.Printf("Warning: could not create %s network: %v", PaasNetwork, err)
	} else {
		log.Printf("Created Docker network: %s", PaasNetwork)
	}
}

func BuildImage(ctx context.Context, buildContext io.Reader, imageTag string) (io.ReadCloser, error) {
	resp, err := Client.ImageBuild(ctx, buildContext, types.ImageBuildOptions{
		Tags:       []string{imageTag},
		Dockerfile: "Dockerfile",
		Remove:     true,
	})
	if err != nil {
		return nil, fmt.Errorf("image build failed: %w", err)
	}
	return resp.Body, nil
}

func RunContainer(ctx context.Context, cfg RunConfig) (string, error) {
	containerName := fmt.Sprintf("ld_%s", cfg.ProjectID)
	routerName := fmt.Sprintf("ld-%s", cfg.ProjectID)
	portStr := fmt.Sprintf("%d", cfg.ContainerPort)

	StopAndRemoveContainer(ctx, containerName)

	// Build Host rule covering all domains
	allDomains := append([]string{cfg.Domain}, cfg.ExtraDomains...)
	hostParts := make([]string, 0, len(allDomains))
	for _, d := range allDomains {
		if d != "" {
			hostParts = append(hostParts, fmt.Sprintf("Host(`%s`)", d))
		}
	}
	hostRule := strings.Join(hostParts, " || ")

	labels := map[string]string{
		"traefik.enable":         "true",
		"traefik.docker.network": PaasNetwork,
		fmt.Sprintf("traefik.http.routers.%s.rule", routerName):                      hostRule,
		fmt.Sprintf("traefik.http.routers.%s.entrypoints", routerName):               "websecure",
		fmt.Sprintf("traefik.http.routers.%s.tls", routerName):                       "true",
		fmt.Sprintf("traefik.http.routers.%s.tls.certresolver", routerName):          "letsencrypt",
		fmt.Sprintf("traefik.http.services.%s.loadbalancer.server.port", routerName): portStr,
		"managed-by": "loomdeploy",
		"project-id": cfg.ProjectID,
	}

	resources := container.Resources{}
	if cfg.CPULimit > 0 {
		resources.CPUQuota = int64(cfg.CPULimit * 100000)
		resources.CPUPeriod = 100000
	}
	if cfg.MemoryLimitMB > 0 {
		resources.Memory = cfg.MemoryLimitMB * 1024 * 1024
	}

	resp, err := Client.ContainerCreate(ctx,
		&container.Config{Image: cfg.ImageTag, Env: cfg.EnvVars, Labels: labels},
		&container.HostConfig{RestartPolicy: container.RestartPolicy{Name: "unless-stopped"}, Resources: resources},
		&dockernetwork.NetworkingConfig{EndpointsConfig: map[string]*dockernetwork.EndpointSettings{PaasNetwork: {}}},
		nil,
		containerName,
	)
	if err != nil {
		return "", fmt.Errorf("container create failed: %w", err)
	}
	if err := Client.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return "", fmt.Errorf("container start failed: %w", err)
	}
	return resp.ID, nil
}

func StopAndRemoveContainer(ctx context.Context, nameOrID string) {
	_ = Client.ContainerStop(ctx, nameOrID, container.StopOptions{})
	_ = Client.ContainerRemove(ctx, nameOrID, container.RemoveOptions{Force: true})
}

func ContainerAction(ctx context.Context, containerID string, action string) error {
	switch action {
	case "start":
		return Client.ContainerStart(ctx, containerID, container.StartOptions{})
	case "stop":
		return Client.ContainerStop(ctx, containerID, container.StopOptions{})
	case "restart":
		return Client.ContainerRestart(ctx, containerID, container.StopOptions{})
	default:
		return fmt.Errorf("unknown action: %s", action)
	}
}

func StreamLogs(ctx context.Context, containerID string) (io.ReadCloser, error) {
	return Client.ContainerLogs(ctx, containerID, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     true,
		Tail:       "200",
	})
}

func RemoveImage(ctx context.Context, imageTag string) {
	items, _ := Client.ImageList(ctx, image.ListOptions{})
	for _, img := range items {
		for _, tag := range img.RepoTags {
			if tag == imageTag {
				_, _ = Client.ImageRemove(ctx, img.ID, image.RemoveOptions{Force: true, PruneChildren: true})
				return
			}
		}
	}
}

type ContainerUsage struct {
	CPUPercent    float64 `json:"cpu_percent"`
	MemoryUsageMB float64 `json:"memory_usage_mb"`
	MemoryLimitMB float64 `json:"memory_limit_mb"`
	MemoryPercent float64 `json:"memory_percent"`
}

func GetContainerStats(ctx context.Context, containerID string) (*ContainerUsage, error) {
	resp, err := Client.ContainerStats(ctx, containerID, false)
	if err != nil {
		return nil, fmt.Errorf("stats error: %w", err)
	}
	defer resp.Body.Close()

	var raw struct {
		CPUStats struct {
			CPUUsage struct {
				TotalUsage  uint64   `json:"total_usage"`
				PercpuUsage []uint64 `json:"percpu_usage"`
			} `json:"cpu_usage"`
			SystemUsage uint64 `json:"system_cpu_usage"`
			OnlineCPUs  uint32 `json:"online_cpus"`
		} `json:"cpu_stats"`
		PreCPUStats struct {
			CPUUsage struct {
				TotalUsage uint64 `json:"total_usage"`
			} `json:"cpu_usage"`
			SystemUsage uint64 `json:"system_cpu_usage"`
		} `json:"precpu_stats"`
		MemoryStats struct {
			Usage uint64            `json:"usage"`
			Limit uint64            `json:"limit"`
			Stats map[string]uint64 `json:"stats"`
		} `json:"memory_stats"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("stats decode: %w", err)
	}

	cpuDelta := float64(raw.CPUStats.CPUUsage.TotalUsage - raw.PreCPUStats.CPUUsage.TotalUsage)
	sysDelta := float64(raw.CPUStats.SystemUsage - raw.PreCPUStats.SystemUsage)
	numCPUs := float64(raw.CPUStats.OnlineCPUs)
	if numCPUs == 0 {
		numCPUs = float64(len(raw.CPUStats.CPUUsage.PercpuUsage))
	}
	cpuPct := 0.0
	if sysDelta > 0 && cpuDelta > 0 && numCPUs > 0 {
		cpuPct = (cpuDelta / sysDelta) * numCPUs * 100.0
	}

	cache := raw.MemoryStats.Stats["cache"]
	memUsed := float64(raw.MemoryStats.Usage-cache) / 1024 / 1024
	memLimit := float64(raw.MemoryStats.Limit) / 1024 / 1024
	memPct := 0.0
	if memLimit > 0 {
		memPct = memUsed / memLimit * 100.0
	}

	return &ContainerUsage{
		CPUPercent:    cpuPct,
		MemoryUsageMB: memUsed,
		MemoryLimitMB: memLimit,
		MemoryPercent: memPct,
	}, nil
}

func GetRunningContainerCount(ctx context.Context) (int, int) {
	containers, err := Client.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return 0, 0
	}
	running := 0
	for _, c := range containers {
		if c.State == "running" {
			running++
		}
	}
	return running, len(containers)
}
