package services

import (
	"context"
	"fmt"
	"github.com/docker/docker/api/types/system"
	"github.com/docker/docker/client"
	"log"
	"scrapyd/models"
	"time"
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

func (d *Daemon) GetSystemInfo() (*system.Info, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	systemInfo, err := d.Client.Info(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get system info: %w", err)
	}
	return &systemInfo, nil
}

func (d *Daemon) SetServerStatus(status string) {
	models.DB.Model(d.Server).
		Where("Id = ?", d.Server.Id).
		Update("status", status)
	log.Printf("server [%s] status changed: %s -> %s", d.Server.Id, d.Server.Status, status)
}
