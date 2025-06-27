package services

import (
	"context"
	"fmt"
	"io"
	"scrapyd/models"
)

func JobCleanup(job *models.Job) error {
	d, err := NewDaemon()
	if err != nil {
		return err
	}
	defer d.Client.Close()

	contName := fmt.Sprintf("%s_%s_%s_%s", job.ID, job.ProjectID, job.VersionID, job.Spider)
	cont, err := d.FindContainerByName(contName)
	if err != nil {
		return err
	}

	err = d.ContainerStop(cont.ID)
	if err != nil {
		return err
	}

	err = d.ContainerRemove(cont.ID)
	if err != nil {
		return err
	}

	return nil
}

func JobLogReader(reqCtx context.Context, job *models.Job) (io.ReadCloser, error) {
	d, err := NewDaemon()
	if err != nil {
		return nil, err
	}
	defer d.Client.Close()

	contName := fmt.Sprintf("%s_%s_%s_%s", job.ID, job.ProjectID, job.VersionID, job.Spider)
	cont, err := d.FindContainerByName(contName)
	if err != nil {
		return nil, err
	}

	reader, err := d.ContainerLogs(reqCtx, cont.ID, true)
	if err != nil {
		return nil, err
	}

	return reader, nil
}
