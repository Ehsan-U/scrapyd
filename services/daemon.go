package services

import (
	"bytes"
	"context"
	"fmt"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/pkg/stdcopy"
	"io"
	"log"
	"os"
	"time"

	"github.com/docker/cli/cli/command/image/build"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/system"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/archive"
	"github.com/docker/docker/pkg/idtools"
)

type Daemon struct {
	Address string
	Client  *client.Client
}

func NewDaemon(addr string) (*Daemon, error) {
	addr = "tcp://" + addr + ":2375"
	c, err := client.NewClientWithOpts(
		client.WithHost(addr),
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Docker daemon [%s]: %w", addr, err)
	}
	return &Daemon{
		Address: addr,
		Client:  c,
	}, nil
}

func (d *Daemon) GetSystemInfo() (*system.Info, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	systemInfo, err := d.Client.Info(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get system info: %w", err)
	}
	return &systemInfo, nil
}

func (d *Daemon) ImageIDByName(projectName string) string {
	imageName := projectName + ":latest"
	imageFilters := filters.NewArgs()
	imageFilters.Add("reference", imageName)

	images, err := d.Client.ImageList(
		context.Background(),
		image.ListOptions{Filters: imageFilters},
	)
	if err != nil || len(images) == 0 {
		return ""
	}
	return images[0].ID
}

func (d *Daemon) ImageRemove(imageID string) {
	_, err := d.Client.ImageRemove(
		context.Background(),
		imageID,
		image.RemoveOptions{Force: true, PruneChildren: true},
	)
	if err != nil {
		log.Print(err)
	}
}

func (d *Daemon) ImageBuild(contextDir string, Dockerfile string, projectName string) error {
	excludes, _ := build.ReadDockerignore(contextDir)
	buildCtx, _ := archive.TarWithOptions(
		contextDir,
		&archive.TarOptions{
			ExcludePatterns: excludes,
			ChownOpts:       &idtools.Identity{GID: 0, UID: 0},
		},
	)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*360)
	defer cancel()
	reader, err := d.Client.ImageBuild(
		ctx,
		buildCtx,
		types.ImageBuildOptions{
			Tags:        []string{projectName},
			ForceRemove: true,
			Dockerfile:  Dockerfile,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to build image: %w", err)
	}
	defer reader.Body.Close()

	//var stdout, stderr bytes.Buffer
	if _, err = io.Copy(os.Stdout, reader.Body); err != nil {
		return fmt.Errorf("failed to build image:\n %w", err)
	}
	//stderrContent := stderr.String()
	//if stderrContent != "" {
	//	return fmt.Errorf("failed to build image:\n %s", stderrContent)
	//}

	return nil
}

func (d *Daemon) ContainerCreate(containerName string, config *container.Config) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 360*time.Second)
	defer cancel()
	c, err := d.Client.ContainerCreate(
		ctx,
		config,
		nil, nil, nil,
		containerName,
	)
	if err != nil {
		return "", fmt.Errorf("failed to create container: %w", err)
	}
	return c.ID, nil
}

func (d *Daemon) ContainerStart(containerID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err := d.Client.ContainerStart(
		ctx,
		containerID,
		container.StartOptions{},
	)
	if err != nil {
		return fmt.Errorf("failed to start container: %w", err)
	}
	return nil
}

func (d *Daemon) ContainerWait(containerID string, cond container.WaitCondition) int64 {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	statusChan, _ := d.Client.ContainerWait(
		ctx,
		containerID,
		cond,
	)
	result := <-statusChan
	exitCode := result.StatusCode
	return exitCode
}

func (d *Daemon) ContainerLogs(containerID string) (string, string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	reader, err := d.Client.ContainerLogs(
		ctx,
		containerID,
		container.LogsOptions{
			ShowStdout: true,
			ShowStderr: true,
		},
	)
	if err != nil {
		return "", "", fmt.Errorf("failed to get container logs: %w", err)
	}
	defer reader.Close()
	var stdout, stderr bytes.Buffer
	if _, err = stdcopy.StdCopy(&stdout, &stderr, reader); err != nil {
		return "", "", fmt.Errorf("failed to get container logs: %w", err)
	}
	return stdout.String(), stderr.String(), nil
}

func (d *Daemon) ContainerRemove(containerID string, options container.RemoveOptions) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := d.Client.ContainerRemove(
		ctx,
		containerID,
		options,
	); err != nil {
		return fmt.Errorf("failed to remove container: %w", err)
	}
	return nil
}

func (d *Daemon) SpiderList(projectName string) ([]string, error) {
	imageName := projectName + ":latest"
	containerName := projectName + "_container"

	containerID, err := d.ContainerCreate(containerName, &container.Config{
		Image:      imageName,
		Entrypoint: []string{"scrapy"},
		Cmd:        []string{"list"},
	})
	if err != nil {
		return nil, err
	}

	if err = d.ContainerStart(containerID); err != nil {
		return nil, err
	}

	exitCode := d.ContainerWait(containerID, container.WaitConditionNotRunning)
	if exitCode == 0 {
		stdout, _, err := d.ContainerLogs(containerID)
		if err != nil {
			return nil, err
		}
		log.Printf(stdout)
	}

	if err := d.ContainerRemove(containerID, container.RemoveOptions{
		Force:         true,
		RemoveVolumes: true,
		RemoveLinks:   false,
	}); err != nil {
		return nil, err
	}

	return []string{}, nil
}
