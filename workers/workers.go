package main

import (
	"github.com/hibiken/asynq"
	"github.com/rs/zerolog/log"
	"scrapyd/models"
	"scrapyd/tasks"
)

func main() {
	models.ConnectDatabase()
	srv := asynq.NewServer(
		asynq.RedisClientOpt{Addr: "127.0.0.1:6379"},
		asynq.Config{
			Concurrency: 10,
		},
	)

	mux := asynq.NewServeMux()
	mux.HandleFunc("execute:job", tasks.HandleJobTask)
	mux.HandleFunc("inspect:version", tasks.HandleInspectTask)
	mux.HandleFunc("cancel:job", tasks.HandleCancelTask)

	if err := srv.Run(mux); err != nil {
		log.Fatal().Err(err).Msg("failed to start workers")
	}
}
