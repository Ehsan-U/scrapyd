package services

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/system"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/namesgenerator"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/rs/zerolog/log"
	"io"
	"scrapyd/api/errs"
	"strings"
	"time"
)

type Daemon struct {
	Client *client.Client
}

func NewDaemon() (*Daemon, error) {
	addr := "tcp://127.0.0.1:2375"
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
		Client: c,
	}, nil
}

func (d *Daemon) ContainerCreate(containerName string, config *container.Config) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 360*time.Second)
	defer cancel()

	c, err := d.Client.ContainerCreate(
		ctx,
		config,
		&container.HostConfig{
			RestartPolicy: container.RestartPolicy{Name: "no"},
			LogConfig: container.LogConfig{
				Type: "json-file",
				Config: map[string]string{
					"max-size": "100mb",
					"max-file": "3",
				},
			},
		},
		nil, nil,
		containerName,
	)
	if err != nil {
		log.Error().
			Err(err).
			Str("container", containerName).
			Msg("failed to create container")
		return "", err
	}

	return c.ID, nil
}

func (d *Daemon) ContainerStart(containerID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := d.Client.ContainerStart(
		ctx,
		containerID,
		container.StartOptions{},
	); err != nil {
		log.Error().
			Err(err).
			Str("container", containerID).
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
	select {
	case err := <-errChan:
		if err != nil {
			return -1, err
		}
	case status := <-statusChan:
		return status.StatusCode, nil
	}

	return -1, errors.New("timeout while waiting for container wait api call")
}

func (d *Daemon) ContainerLogs(ctx context.Context, containerID string, follow bool) (io.ReadCloser, error) {
	reader, err := d.Client.ContainerLogs(
		ctx,
		containerID,
		container.LogsOptions{
			ShowStdout: true,
			ShowStderr: true,
			Follow:     follow,
		},
	)
	if err != nil {
		log.Error().
			Err(err).
			Str("container", containerID).
			Msg("failed to get container logs")
		return nil, err
	}

	return reader, nil
}

func (d *Daemon) ContainerRemove(containerID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := d.Client.ContainerRemove(
		ctx,
		containerID,
		container.RemoveOptions{
			Force:         true,
			RemoveVolumes: true,
			RemoveLinks:   false,
		},
	); err != nil {
		log.Error().
			Err(err).
			Str("container", containerID).
			Msg("failed to remove container")
		return err
	}

	return nil
}

func (d *Daemon) ContainerStop(containerID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	timeout := 15
	if err := d.Client.ContainerStop(ctx, containerID, container.StopOptions{
		Timeout: &timeout,
	}); err != nil {
		log.Error().
			Err(err).
			Str("container", containerID).
			Msg("failed to stop the container")
		return err
	}

	return nil
}

func (d *Daemon) FindContainerByName(containerName string) (*container.Summary, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	conFilters := filters.NewArgs()
	conFilters.Add("name", fmt.Sprintf("/%s", containerName))

	containers, err := d.Client.ContainerList(
		ctx,
		container.ListOptions{
			All:     true,
			Filters: conFilters,
		},
	)

	if err != nil {
		log.Error().
			Err(err).
			Msg("failed to list containers")
		return nil, err
	}
	if len(containers) == 0 {
		log.Debug().
			Str("container", containerName).
			Msg("no container found")
		return nil, errors.New("no container found")
	}

	result := containers[0]
	return &result, nil
}

func (d *Daemon) FindContainersByImageName(imageName string) ([]container.Summary, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	imgFilters := filters.NewArgs()
	imgFilters.Add("ancestor", imageName)

	containers, err := d.Client.ContainerList(ctx, container.ListOptions{
		All:     true,
		Filters: imgFilters,
	})
	if err != nil {
		log.Error().
			Err(err).
			Str("image", imageName).
			Msg("failed to list containers by image name")
		return nil, err
	}

	if len(containers) == 0 {
		log.Debug().
			Str("image", imageName).
			Msg("no container found")
		return nil, errors.New("no container found")
	}

	return containers, nil
}

func (d *Daemon) ImageLoad(reader io.Reader) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	loadResponse, err := d.Client.ImageLoad(ctx, reader)
	if err != nil {
		log.Error().
			Err(err).
			Msg("failed to load image")
		return "", err
	}
	defer loadResponse.Body.Close()

	type Line struct {
		Stream string `json:"stream"` // stream is key in each line from docker response
	}

	scanner := bufio.NewScanner(loadResponse.Body)
	for scanner.Scan() {
		var line Line
		json.Unmarshal(scanner.Bytes(), &line)
		if strings.HasPrefix(line.Stream, "Loaded image:") {
			imageName := strings.TrimSpace(strings.TrimPrefix(line.Stream, "Loaded image:"))
			return imageName, nil
		}
	}

	return "", errs.ErrVersionImageTarInvalid
}

func (d *Daemon) ImageRemove(imageName string) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	// find all running containers using image
	containers, err := d.FindContainersByImageName(imageName)
	if err != nil {
		return err
	}

	for _, con := range containers {
		if err := d.ContainerStop(con.ID); err != nil {
			return err
		}
		if err := d.ContainerRemove(con.ID); err != nil {
			return err
		}
	}

	_, err = d.Client.ImageRemove(ctx, imageName, image.RemoveOptions{
		Force:         true,
		PruneChildren: true,
	})
	if err != nil {
		return err
	}

	return nil
}

func (d *Daemon) SpiderList(imageName string) ([]string, error) {
	var spiders []string

	contName := namesgenerator.GetRandomName(1)
	containerID, err := d.ContainerCreate(contName, &container.Config{
		Image:      imageName,
		Entrypoint: []string{"scrapy"},
		Cmd:        []string{"list"},
	})
	if err != nil {
		return nil, err
	}
	// cleanup
	defer func() {
		_ = d.ContainerRemove(containerID)
	}()

	if err = d.ContainerStart(containerID); err != nil {
		return nil, err
	}

	exitCode, err := d.ContainerWait(containerID, container.WaitConditionNotRunning)
	if err != nil {
		return nil, err
	}
	if exitCode == 0 {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		reader, err := d.ContainerLogs(ctx, containerID, false)
		defer reader.Close()

		var stdout bytes.Buffer
		if _, err = stdcopy.StdCopy(&stdout, io.Discard, reader); err != nil {
			log.Error().
				Err(err).
				Str("container", containerID).
				Msg("failed to read container logs")
			return nil, err
		}
		logs := stdout.String()

		if logs == "" {
			return nil, err
		}
		for _, spider := range strings.Split(logs, "\n") {
			if strings.TrimSpace(spider) != "" {
				spiders = append(spiders, spider)
			}
		}
		return spiders, nil
	}

	return spiders, nil
}

func (d *Daemon) GetSystemInfo() (*system.Info, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	info, err := d.Client.Info(ctx)
	if err != nil {
		log.Error().
			Err(err).
			Msg("failed to get system info")
		return nil, err
	}

	return &info, nil
}

////////////////////

func DaemonStatus() (*system.Info, error) {
	d, err := NewDaemon()
	if err != nil {
		return nil, err
	}
	defer d.Client.Close()

	info, err := d.GetSystemInfo()
	if err != nil {
		return nil, err
	}

	return info, nil
}
