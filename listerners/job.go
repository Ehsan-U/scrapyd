package listerners

import (
	"context"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"
	"github.com/rs/zerolog/log"
	"scrapyd/models"
	"scrapyd/services"
	"strings"
)

func StartDockerEventListener(ctx context.Context) {
	d, err := services.NewDaemon()
	if err != nil {
		log.Error().Err(err).Msg("")
		return
	}
	defer d.Client.Close()

	eventFilters := filters.NewArgs()
	eventFilters.Add("type", "container")
	eventFilters.Add("event", "die")
	eventFilters.Add("event", "stop")
	eventFilters.Add("event", "oom")

	msgChan, errChan := d.Client.Events(ctx, events.ListOptions{
		Filters: eventFilters,
	})
	for {
		select {
		case msg, ok := <-msgChan:
			if !ok {
				log.Info().Msg("Message channel closed")
				return
			}
			log.Debug().Msgf("%s %s\n", msg.Actor.ID, msg.Action)
			label := msg.Actor.Attributes["label"]
			if msg.Action == "die" && label == "scrapyd" {
				var job models.Job
				jobID := strings.Split(msg.Actor.Attributes["name"], "_")[0]
				if err := models.DB.First(&job, "id = ?", jobID).Error; err != nil {
					log.Debug().
						Str("job", jobID).
						Msg("job not found")
					continue
				}
				job.Status = "finished"
				models.DB.Save(&job)
			}

		case err, ok := <-errChan:
			if !ok {
				log.Info().Msg("Error channel closed")
				return
			}
			log.Error().Err(err).Msg("")
			return

		case <-ctx.Done():
			log.Info().Msg("Context Done signal received. Stopping event monitoring.")
			return
		}
	}
}
