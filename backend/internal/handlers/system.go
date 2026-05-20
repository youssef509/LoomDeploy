package handlers

import (
	"context"
	"net/http"

	"loomdeploy/internal/docker"

	"github.com/gin-gonic/gin"
)

type ServerStats struct {
	CPUUsage          float64 `json:"cpu_usage"`
	MemoryUsed        int64   `json:"memory_used"`
	MemoryTotal       int64   `json:"memory_total"`
	DiskUsed          int64   `json:"disk_used"`
	DiskTotal         int64   `json:"disk_total"`
	UptimeSeconds     int64   `json:"uptime_seconds"`
	RunningContainers int     `json:"running_containers"`
	TotalContainers   int     `json:"total_containers"`
}

func GetSystemStats(c *gin.Context) {
	stats := ServerStats{}

	stats.UptimeSeconds = getUptime()
	stats.MemoryUsed, stats.MemoryTotal = getMemory()
	stats.DiskUsed, stats.DiskTotal = getDisk()
	stats.CPUUsage = getCPU()

	running, total := docker.GetRunningContainerCount(context.Background())
	stats.RunningContainers = running
	stats.TotalContainers = total

	c.JSON(http.StatusOK, stats)
}
