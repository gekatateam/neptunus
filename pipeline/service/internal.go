package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"

	"github.com/gekatateam/neptunus/config"
	"github.com/gekatateam/neptunus/logger"
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/pipeline"
	"github.com/gekatateam/neptunus/pkg/syncs"
)

var _ pipeline.Service = (*internalService)(nil)

type pipeUnit struct {
	p  *pipeline.Pipeline
	mu *sync.Mutex
	c  context.CancelFunc
}

type internalService struct {
	pipes *syncs.Map[string, pipeUnit]
	log   *slog.Logger
	wg    *sync.WaitGroup
	s     pipeline.Storage
}

func Internal(s pipeline.Storage, log *slog.Logger) *internalService {
	return &internalService{
		log:   log,
		s:     s,
		pipes: &syncs.Map[string, pipeUnit]{},
		wg:    &sync.WaitGroup{},
	}
}

func (m *internalService) StartAll() error {
	pipes, err := m.s.List()
	if err != nil {
		return err
	}

	for _, pipeCfg := range pipes {
		unit := pipeUnit{m.createPipeline(pipeCfg), &sync.Mutex{}, nil}
		m.pipes.Store(pipeCfg.Settings.Id, unit)
		if pipeCfg.Settings.Run {
			m.runPipeline(unit)
		}
	}

	return nil
}

func (m *internalService) StopAll() {
	m.pipes.Range(func(_ string, u pipeUnit) bool {
		u.mu.Lock()
		if u.c != nil {
			println(u.p.Config().Settings.Id)
			u.c()
		}
		u.mu.Unlock()

		return true
	})

	m.wg.Wait()
}

func (m *internalService) Start(id string) error {
	pipeCfg, err := m.s.Get(id)
	if err != nil {
		return err
	}

	unit, _ := m.pipes.LoadOrStore(id, pipeUnit{m.createPipeline(pipeCfg), &sync.Mutex{}, nil})

	switch unit.p.State() {
	case pipeline.StateRunning:
		return &pipeline.ConflictError{Err: errors.New("pipeline already running")}
	case pipeline.StateStopping:
		return &pipeline.ConflictError{Err: errors.New("pipeline stopping, please wait")}
	case pipeline.StateBuilding:
		return &pipeline.ConflictError{Err: errors.New("pipeline building, please wait")}
	case pipeline.StateStarting:
		return &pipeline.ConflictError{Err: errors.New("pipeline starting, please wait")}
	}

	// double-check here because pipeline buid and start stages
	// may take a lot of time
	// so, if several parallel calls occured, only one will go to runPipeline()
	unit.mu.Lock()
	defer unit.mu.Unlock()

	switch unit.p.State() {
	case pipeline.StateRunning:
		return &pipeline.ConflictError{Err: errors.New("pipeline already running")}
	case pipeline.StateStopping:
		return &pipeline.ConflictError{Err: errors.New("pipeline stopping, please wait")}
	case pipeline.StateBuilding:
		return &pipeline.ConflictError{Err: errors.New("pipeline building, please wait")}
	case pipeline.StateStarting:
		return &pipeline.ConflictError{Err: errors.New("pipeline starting, please wait")}
	}

	return m.runPipeline(unit)
}

func (m *internalService) Stop(id string) error {
	unit, ok := m.pipes.Load(id)
	if !ok {
		return &pipeline.NotFoundError{Err: errors.New("pipeline unit not found in runtime registry")}
	}

	switch unit.p.State() {
	case pipeline.StateStopped, pipeline.StateCreated:
		return &pipeline.ConflictError{Err: errors.New("pipeline already stopped")}
	case pipeline.StateStopping:
		return &pipeline.ConflictError{Err: errors.New("pipeline stopping, please wait")}
	case pipeline.StateBuilding:
		return &pipeline.ConflictError{Err: errors.New("pipeline building, please wait")}
	case pipeline.StateStarting:
		return &pipeline.ConflictError{Err: errors.New("pipeline starting, please wait")}
	}

	unit.mu.Lock()
	defer unit.mu.Unlock()

	switch unit.p.State() {
	case pipeline.StateStopped, pipeline.StateCreated:
		return &pipeline.ConflictError{Err: errors.New("pipeline already stopped")}
	case pipeline.StateStopping:
		return &pipeline.ConflictError{Err: errors.New("pipeline stopping, please wait")}
	case pipeline.StateBuilding:
		return &pipeline.ConflictError{Err: errors.New("pipeline building, please wait")}
	case pipeline.StateStarting:
		return &pipeline.ConflictError{Err: errors.New("pipeline starting, please wait")}
	}

	unit.c()
	return nil
}

func (m *internalService) State(id string) (string, error, error) {
	unit, ok := m.pipes.Load(id)
	if !ok {
		return "", nil, &pipeline.NotFoundError{Err: errors.New("pipeline unit not found in runtime registry")}
	}

	return string(unit.p.State()), unit.p.LastError(), nil
}

func (m *internalService) List() ([]*config.Pipeline, error) {
	return m.s.List()
}

func (m *internalService) Get(id string) (*config.Pipeline, error) {
	return m.s.Get(id)
}

func (m *internalService) Add(pipeCfg *config.Pipeline) error {
	_, ok := m.pipes.Load(pipeCfg.Settings.Id)
	if ok {
		return &pipeline.ConflictError{Err: errors.New("pipeline unit exists in runtime registry")}
	}

	if err := m.s.Add(pipeCfg); err != nil {
		return err
	}

	m.pipes.Store(pipeCfg.Settings.Id, pipeUnit{m.createPipeline(pipeCfg), &sync.Mutex{}, nil})
	return nil
}

func (m *internalService) Update(pipeCfg *config.Pipeline) error {
	unit, ok := m.pipes.Load(pipeCfg.Settings.Id)
	if !ok {
		return &pipeline.NotFoundError{Err: errors.New("pipeline unit not found in runtime registry")}
	}

	switch unit.p.State() {
	case pipeline.StateStopped, pipeline.StateCreated:
	default:
		return &pipeline.ConflictError{Err: errors.New("only stopped pipelines can be updated")}
	}

	unit.mu.Lock()
	defer unit.mu.Unlock()

	switch unit.p.State() {
	case pipeline.StateStopped, pipeline.StateCreated:
	default:
		return &pipeline.ConflictError{Err: errors.New("only stopped pipelines can be updated")}
	}

	if err := m.s.Update(pipeCfg); err != nil {
		return err
	}

	m.pipes.Store(pipeCfg.Settings.Id, pipeUnit{m.createPipeline(pipeCfg), unit.mu, nil})
	return nil
}

func (m *internalService) Delete(id string) error {
	unit, ok := m.pipes.Load(id)
	if !ok {
		return &pipeline.NotFoundError{Err: errors.New("pipeline unit not found in runtime registry")}
	}

	switch unit.p.State() {
	case pipeline.StateStopped, pipeline.StateCreated:
	default:
		return &pipeline.ConflictError{Err: errors.New("only stopped pipelines can be deleted")}
	}

	unit.mu.Lock()
	defer unit.mu.Unlock()

	switch unit.p.State() {
	case pipeline.StateStopped, pipeline.StateCreated:
	default:
		return &pipeline.ConflictError{Err: errors.New("only stopped pipelines can be deleted")}
	}

	if err := m.s.Delete(id); err != nil {
		return err
	}

	m.pipes.Delete(id)
	return nil
}

func (m *internalService) createPipeline(pipeCfg *config.Pipeline) *pipeline.Pipeline {
	return pipeline.New(pipeCfg, logger.Default.With(
		slog.Group("pipeline",
			"id", pipeCfg.Settings.Id,
		),
	))
}

func (m *internalService) runPipeline(pipeUnit pipeUnit) error {
	id := pipeUnit.p.Config().Settings.Id
	m.log.Info("building pipeline",
		"id", id,
	)
	if err := pipeUnit.p.Build(); err != nil {
		m.log.Error("pipeline building failed",
			"error", err.Error(),
			"id", id,
		)

		pipeUnit.p.Close()
		m.log.Warn("pipeline was closed as not ready for event processing",
			"id", id,
		)
		return &pipeline.ValidationError{Err: fmt.Errorf("pipeline %v building failed: %v", id, err)}
	}

	ctx, cancel := context.WithCancel(context.Background())
	pipeUnit.c = cancel
	m.pipes.Store(id, pipeUnit)

	m.wg.Add(1)
	go func(p *pipeline.Pipeline) {
		p.Run(ctx)
		m.wg.Done()
		cancel()
	}(pipeUnit.p)

	return nil
}

func (m *internalService) Stats() []metrics.PipelineStats {
	pipesState := []metrics.PipelineStats{}

	m.pipes.Range(func(name string, pipe pipeUnit) bool {
		pipesState = append(pipesState, metrics.PipelineStats{
			Pipeline: name,
			State:    pipeline.StateCode[pipe.p.State()],
			Lines:    pipe.p.Config().Settings.Lines,
			Chans:    pipe.p.ChansStats(),
		})

		return true
	})
	return pipesState
}
