package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"

	dynamic "github.com/gekatateam/dynamic-level-handler"

	"github.com/gekatateam/neptunus/config"
	"github.com/gekatateam/neptunus/logger"
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/pipeline"
	xerrors "github.com/gekatateam/neptunus/pkg/errors"
	"github.com/gekatateam/neptunus/pkg/syncs"
)

var _ pipeline.Service = (*internalService)(nil)

type pipeUnit struct {
	p  *pipeline.Pipeline
	mu *sync.Mutex
	c  context.CancelFunc
}

type internalService struct {
	pipes syncs.Map[string, pipeUnit]
	log   *slog.Logger
	wg    *sync.WaitGroup
	s     pipeline.Storage
}

func Internal(s pipeline.Storage, log *slog.Logger) *internalService {
	return &internalService{
		log:   log,
		s:     s,
		pipes: syncs.New[string, pipeUnit](),
		wg:    &sync.WaitGroup{},
	}
}

func (m *internalService) StartAll() error {
	pipes, err := m.s.List()
	if err != nil {
		return err
	}

	var errs xerrors.Errorlist
	for _, pipeCfg := range pipes {
		unit := pipeUnit{m.createPipeline(pipeCfg), &sync.Mutex{}, nil}
		m.pipes.Store(pipeCfg.Settings.Id, unit)
		if pipeCfg.Settings.Run {
			if err := m.startPipeline(unit); err != nil {
				errs = append(errs, err)
			}
		}
	}

	if len(errs) > 0 {
		return &errs
	}

	return nil
}

func (m *internalService) StopAll() {
	m.pipes.Range(func(_ string, u pipeUnit) bool {
		u.mu.Lock()
		m.stopPipeline(u)
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
	// so, if several parallel calls occurred, only one will go to startPipeline()
	unit.mu.Lock()
	defer unit.mu.Unlock()

	// Load() returns unit that already updated by startPipeline()
	// it saves us from multiple runs of one pipeline, because we MUST recreate it
	// to run with new config
	unit, _ = m.pipes.Load(id)

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

	return m.startPipeline(pipeUnit{m.createPipeline(pipeCfg), unit.mu, nil})
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

	m.stopPipeline(unit)
	return nil
}

func (m *internalService) State(id string) (string, error, error) {
	unit, ok := m.pipes.Load(id)
	if !ok {
		return "", nil, &pipeline.NotFoundError{Err: errors.New("pipeline unit not found in runtime registry")}
	}

	return unit.p.State().String(), unit.p.LastError(), nil
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
	log := logger.Default.With(slog.Group("pipeline",
		"id", pipeCfg.Settings.Id,
	))
	dynamic.OverrideLevel(log.Handler(), logger.ShouldLevelToLeveler(pipeCfg.Settings.LogLevel))

	return pipeline.New(pipeCfg, log)
}

func (m *internalService) startPipeline(pipeUnit pipeUnit) error {
	id := pipeUnit.p.Config().Settings.Id
	m.pipes.Store(id, pipeUnit)
	m.log.Info("building pipeline",
		"id", id,
	)

	if err := pipeUnit.p.Build(); err != nil {
		pipeUnit.p.Close()
		m.log.Error("pipeline building failed, pipeline closed as not ready for event processing",
			"error", err.Error(),
			slog.Group("pipeline",
				"id", id,
			),
		)
		return &pipeline.ValidationError{Err: fmt.Errorf("pipeline %v build failed: %v", id, err)}
	}

	if err := m.s.Acquire(id); err != nil {
		pipeUnit.p.Close()
		m.log.Error("pipeline lock not acquired, pipeline closed to prevent inconsistent state",
			"error", err,
			slog.Group("pipeline",
				"id", id,
			),
		)
		return &pipeline.ConflictError{Err: fmt.Errorf("pipeline %v lock failed: %v", id, err)}
	}

	ctx, cancel := context.WithCancel(context.Background())
	pipeUnit.c = cancel
	m.pipes.Store(id, pipeUnit)

	m.wg.Add(1)
	go func(p *pipeline.Pipeline) {
		p.Run(ctx)
		p.Close()
		m.wg.Done()
		cancel()
	}(pipeUnit.p)

	return nil
}

func (m *internalService) stopPipeline(pipeUnit pipeUnit) {
	id := pipeUnit.p.Config().Settings.Id

	if pipeUnit.c != nil {
		pipeUnit.c()
	}

	if err := m.s.Release(id); err != nil {
		m.log.Error("pipeline lock not released",
			"error", err,
			slog.Group("pipeline",
				"id", id,
			),
		)
	}
}

func (m *internalService) Stats() []metrics.PipelineStats {
	pipesState := []metrics.PipelineStats{}

	m.pipes.Range(func(name string, pipe pipeUnit) bool {
		pipesState = append(pipesState, metrics.PipelineStats{
			Pipeline: name,
			Run:      pipe.p.Config().Settings.Run,
			State:    int(pipe.p.State()),
			Lines:    pipe.p.Config().Settings.Lines,
			Chans:    pipe.p.ChansStats(),
		})

		return true
	})
	return pipesState
}
