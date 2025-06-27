package services

import (
	"mime/multipart"
	"scrapyd/models"
)

func VersionCleanup(version *models.Version) error {
	d, err := NewDaemon()
	if err != nil {
		return err
	}
	defer d.Client.Close()

	if err = d.ImageRemove(version.Image); err != nil {
		return err
	}

	return nil
}

func VersionInit(file multipart.File) string {
	d, err := NewDaemon()
	if err != nil {
		return ""
	}
	defer d.Client.Close()

	imageName, err := d.ImageLoad(file)
	if err != nil {
		return ""
	}

	return imageName
}
