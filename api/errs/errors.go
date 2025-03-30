package errs

import (
	"errors"
	"net/http"
)

var (
	ErrDaemonNotResponding    = errors.New("docker daemon not responding")
	ErrDaemonConnectionFailed = errors.New("unable to connect to the Docker daemon")
	ErrInvalidGitUrl          = errors.New("invalid git url")
	ErrInvalidDockerfile      = errors.New("invalid dockerfile")
	ErrSpidersNotFound        = errors.New("spiders not found in the project")
	ErrServerNotFound         = errors.New("server not found")
)

var ErrStatusMap = map[error]int{
	ErrDaemonNotResponding:    http.StatusInternalServerError,
	ErrDaemonConnectionFailed: http.StatusUnprocessableEntity,
	ErrInvalidGitUrl:          http.StatusUnprocessableEntity,
	ErrInvalidDockerfile:      http.StatusUnprocessableEntity,
	ErrSpidersNotFound:        http.StatusUnprocessableEntity,
	ErrServerNotFound:         http.StatusNotFound,
}
