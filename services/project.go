package services

import "scrapyd/models"

func ProjectCleanup(project *models.Project) error {
	d, err := NewDaemon()
	if err != nil {
		return err
	}
	defer d.Client.Close()

	for _, v := range project.Versions {
		if err := VersionCleanup(&v); err != nil {
			return err
		}
	}

	return nil
}
