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
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/moby/term"
	"github.com/rs/zerolog/log"
	"io"
	"strings"
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

func (d *Daemon) ImageIDByName(imageName string) (string, error) {
	imageFilters := filters.NewArgs()
	imageFilters.Add("reference", imageName)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	images, err := d.Client.ImageList(
		ctx,
		image.ListOptions{Filters: imageFilters},
	)

	if err != nil {
		log.Error().
			Err(err).
			Str("daemon", d.Address).
			Msg("failed to get image id")
		return "", err
	}
	if len(images) == 0 {
		return "", nil
	}

	return images[0].ID, nil
}

func (d *Daemon) ImageRemove(imageID string) error {
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
		return err
	}

	return nil
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

	var buildLogs bytes.Buffer
	fd, _ := term.GetFdInfo(io.Discard)
	err = jsonmessage.DisplayJSONMessagesStream(reader.Body, &buildLogs, fd, false, nil)
	if err != nil {
		log.Error().
			Err(err).
			Str("daemon", d.Address).
			Msg("failed to get image build logs")
		return err
	}

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

func (d *Daemon) ContainerWait(containerID string, cond container.WaitCondition) (int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	statusChan, errChan := d.Client.ContainerWait(
		ctx,
		containerID,
		cond,
	)
	result := <-statusChan
	exitCode := result.StatusCode
	if err := <-errChan; err != nil {
		return exitCode, err
	}
	return exitCode, nil
}

func (d *Daemon) ContainerLogs(containerID string) (string, error) {
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
		return "", err
	}
	defer reader.Close()
	var stdout bytes.Buffer
	if _, err = stdcopy.StdCopy(&stdout, io.Discard, reader); err != nil {
		log.Error().
			Err(err).
			Str("daemon", d.Address).
			Msg("failed to read container logs")
		return "", err
	}
	return stdout.String(), nil
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

func (d *Daemon) ContainerIDByName(containerName string) (string, error) {
	conFilters := filters.NewArgs()
	conFilters.Add("name", containerName)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	containers, err := d.Client.ContainerList(
		ctx,
		container.ListOptions{
			Filters: conFilters,
		},
	)

	if err != nil {
		log.Error().
			Err(err).
			Str("daemon", d.Address).
			Msg("failed to list containers")
		return "", err
	}
	if len(containers) == 0 {
		return "", nil
	}

	return containers[0].ID, nil
}

func (d *Daemon) SpiderList(projectName string) ([]string, error) {
	imageName := projectName + ":latest"
	containerName := projectName + "_container"

	containerID, err := d.ContainerIDByName(containerName)
	if err != nil {
		return nil, err
	}
	if containerID != "" {
		if err := d.ContainerRemove(containerID, container.RemoveOptions{
			Force:         true,
			RemoveVolumes: true,
			RemoveLinks:   false,
		}); err != nil {
			return nil, err
		}
	}

	containerID, err = d.ContainerCreate(containerName, &container.Config{
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

	exitCode, err := d.ContainerWait(containerID, container.WaitConditionNotRunning)
	if err != nil {
		return nil, err
	}
	if exitCode == 0 {
		stdout, err := d.ContainerLogs(containerID)
		if err != nil || stdout == "" {
			return nil, err
		}
		spiders := strings.Split(stdout, "\n")
		return spiders, nil
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
