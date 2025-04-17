package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/docker/docker/api/types/container"
	"github.com/hibiken/asynq"
	"github.com/rs/zerolog/log"
	"scrapyd/models"
	"scrapyd/services"
)

type Task struct {
	ID string
}

func NewTask(typeName string, ID string) error {
	client := asynq.NewClient(asynq.RedisClientOpt{Addr: "127.0.0.1:6379"})
	defer client.Close()

	payload, err := json.Marshal(Task{ID: ID})
	if err != nil {
		log.Error().
			Err(err).
			Str("type", typeName).
			Str("id", ID).
			Msgf("failed to create new task")
		return err
	}
	task := asynq.NewTask(typeName, payload)

	_, err = client.Enqueue(task, asynq.TaskID(ID), asynq.MaxRetry(1))
	if err != nil {
		log.Error().
			Err(err).
			Str("type", typeName).
			Str("task", ID).
			Msg("failed to enqueue task")
		return err
	}

	return nil
}

func HandleJobTask(ctx context.Context, t *asynq.Task) error {
	var task Task
	var job models.Job

	err := json.Unmarshal(t.Payload(), &task)
	if err != nil {
		return err
	}

	if err := models.DB.Preload("Project").Preload("Version").First(&job, "id = ?", task.ID).Error; err != nil {
		return err
	}

	d, err := services.NewDaemon()
	if err != nil {
		return err
	}
	defer d.Client.Close()

	contName := fmt.Sprintf("%s_%s_%s_%s", job.ID, job.ProjectID, job.VersionID, job.Spider)
	contID, err := d.ContainerCreate(contName, &container.Config{
		Image:      job.Version.Image,
		Entrypoint: []string{"scrapy"},
		Cmd:        []string{"crawl", job.Spider},
		Labels: map[string]string{
			"label": "scrapyd",
		},
	})
	if err != nil {
		return err
	}

	if err := d.ContainerStart(contID); err != nil {
		return err
	}
	job.Status = "running"

	models.DB.Save(&job)
	return nil
}

func HandleInspectTask(ctx context.Context, t *asynq.Task) error {
	var task Task
	var version models.Version

	err := json.Unmarshal(t.Payload(), &task)
	if err != nil {
		return err
	}

	if err := models.DB.First(&version, "id = ?", task.ID).Error; err != nil {
		return err
	}

	d, err := services.NewDaemon()
	if err != nil {
		return err
	}
	defer d.Client.Close()

	spiders, err := d.SpiderList(&version)
	if err != nil {
		return err
	}
	version.Spiders = spiders

	models.DB.Save(&version)
	return nil
}

func HandleCancelTask(ctx context.Context, t *asynq.Task) error {
	var task Task
	var job models.Job

	err := json.Unmarshal(t.Payload(), &task)
	if err != nil {
		return err
	}

	if err := models.DB.Preload("Project").First(&job, "id = ?", task.ID).Error; err != nil {
		log.Error().
			Err(err).
			Str("type", t.Type()).
			Str("job", task.ID).
			Msg("job not found")
		return err
	}

	d, err := services.NewDaemon()
	if err != nil {
		return err
	}
	defer d.Client.Close()

	contName := fmt.Sprintf("%s_%s_%s_%s", job.ID, job.ProjectID, job.VersionID, job.Spider)
	if err := d.ContainerStop(contName); err != nil {
		return err
	}
	job.Status = "cancelled"

	models.DB.Save(&job)
	return nil
}
