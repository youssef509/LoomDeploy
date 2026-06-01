//go:build windows

package docker

import (
	"context"
	"fmt"
	"io"
	"log"
)

const PaasNetwork = "paas_network"

func Init() {
	log.Println("Docker integration disabled on Windows (stubs active)")
}

func BuildImage(_ context.Context, _ io.Reader, _ string) (io.ReadCloser, error) {
	return nil, fmt.Errorf("Docker not available on Windows build")
}

func RunContainer(_ context.Context, _ RunConfig) (string, error) {
	return "", fmt.Errorf("Docker not available on Windows build")
}

func StopAndRemoveContainer(_ context.Context, _ string) {}

func ContainerAction(_ context.Context, _ string, _ string) error {
	return fmt.Errorf("Docker not available on Windows build")
}

func StreamLogs(_ context.Context, _ string) (io.ReadCloser, error) {
	return nil, fmt.Errorf("Docker not available on Windows build")
}

func RemoveImage(_ context.Context, _ string) {}

func GetRunningContainerCount(_ context.Context) (int, int) { return 0, 0 }

func PullImage(_ context.Context, _ string, _ func(string)) error {
	return fmt.Errorf("Docker not available on Windows build")
}

func GetContainerStats(_ context.Context, _ string) (*ContainerUsage, error) {
	return nil, fmt.Errorf("Docker not available on Windows build")
}
