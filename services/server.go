package services

import (
	"context"
	"fmt"
	"github.com/docker/docker/api/types/system"
	"github.com/docker/docker/client"
	"log"
	"scrapyd/models"
)

type Daemon struct {
	Server *models.Server
	Client *client.Client
}

func NewDaemon(server *models.Server) (*Daemon, error) {
	addr := "tcp://" + server.Address + ":2375"
	apiClient, err := client.NewClientWithOpts(
		client.WithHost(addr),
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Docker daemon: %w", err)
	}

	return &Daemon{
		Server: server,
		Client: apiClient,
	}, nil
}

func (d *Daemon) GetSystemInfo(ctx context.Context) (*system.Info, error) {
	systemInfo, err := d.Client.Info(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get system info: %w", err)
	}
	return &systemInfo, nil
}

func SetServerStatus(server *models.Server, status string) {
	models.DB.Model(server).
		Where("Id = ?", server.Id).
		Update("status", status)
	log.Printf("server [%s] status changed: %s -> %s", server.Id, server.Status, status)
}
