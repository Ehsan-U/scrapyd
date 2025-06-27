package errs

import (
	"errors"
	"net/http"
)

var (
	ErrProjectNotFound = errors.New("project not found")
	ErrProjectConflict = errors.New("project already exists")

	ErrVersionNotFound         = errors.New("version not found")
	ErrVersionConflict         = errors.New("version already exists")
	ErrVersionImageTarNotFound = errors.New("image_tar not found")
	ErrVersionImageTarInvalid  = errors.New("image_tar is invalid")

	ErrJobNotFound = errors.New("job not found")
	ErrJobCreate   = errors.New("job failed to create")
	ErrJobConflict = errors.New("job already exists")

	ErrSpiderNotFound = errors.New("spider not found")
)

var ErrStatusMap = map[error]int{
	ErrProjectNotFound: http.StatusNotFound,
	ErrProjectConflict: http.StatusConflict,

	ErrVersionNotFound:         http.StatusNotFound,
	ErrVersionConflict:         http.StatusConflict,
	ErrVersionImageTarNotFound: http.StatusBadRequest,
	ErrVersionImageTarInvalid:  http.StatusBadRequest,

	ErrJobNotFound: http.StatusNotFound,
	ErrJobCreate:   http.StatusInternalServerError,
	ErrJobConflict: http.StatusConflict,

	ErrSpiderNotFound: http.StatusNotFound,
}
