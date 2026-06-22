package service

import (
	"log/slog"

	"github.com/gekatateam/neptunus/pipeline"
)

type workerService struct {
	p   *pipeline.Pipeline
	log *slog.Logger
}