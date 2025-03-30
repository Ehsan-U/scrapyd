package services

import (
	"bytes"
	"context"
	"github.com/docker/cli/cli/command/image/build"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/system"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/archive"
	"github.com/docker/docker/pkg/idtools"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/rs/zerolog/log"
	"io"
	"os"
	"time"
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
		log.Error().
			Err(err).
			Str("daemon", addr).
			Msg("failed to connect to docker daemon")
		return nil, err
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
		log.Error().
			Err(err).
			Str("daemon", d.Address).
			Msg("failed to get system info")
		return nil, err
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
	if err != nil {
		log.Error().
			Err(err).
			Str("daemon", d.Address).
			Msg("failed to get image id")
		return ""
	}
	if len(images) == 0 {
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
		log.Error().
			Err(err).
			Str("daemon", d.Address).
			Msg("failed to remove image")
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
		log.Error().
			Err(err).
			Str("daemon", d.Address).
			Msg("failed to build image")
		return err
	}
	defer reader.Body.Close()

	//var stdout, stderr bytes.Buffer
	if _, err = io.Copy(os.Stdout, reader.Body); err != nil {
		log.Error().
			Err(err).
			Str("daemon", d.Address).
			Msg("failed to build image")
		return err
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
		log.Error().
			Err(err).
			Str("daemon", d.Address).
			Msg("failed to create container")
		return "", err
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
		log.Error().
			Err(err).
			Str("daemon", d.Address).
			Msg("failed to start container")
		return err
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
		log.Error().
			Err(err).
			Str("daemon", d.Address).
			Msg("failed to get container logs")
		return "", "", err
	}
	defer reader.Close()
	var stdout, stderr bytes.Buffer
	if _, err = stdcopy.StdCopy(&stdout, &stderr, reader); err != nil {
		log.Error().
			Err(err).
			Str("daemon", d.Address).
			Msg("failed to get container logs")
		return "", "", err
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
		log.Error().
			Err(err).
			Str("daemon", d.Address).
			Msg("failed to remove container")
		return err
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
