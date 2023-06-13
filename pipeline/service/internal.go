package service

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/gekatateam/neptunus/config"
	"github.com/gekatateam/neptunus/logger"
	"github.com/gekatateam/neptunus/logger/logrus"
	"github.com/gekatateam/neptunus/pipeline"
)

var _ pipeline.Service = &internalService{}

type pipeUnit struct {
	p *pipeline.Pipeline
	c context.CancelFunc
}

type internalService struct {
	pipes map[string]pipeUnit
	log   logger.Logger
	wg    *sync.WaitGroup
	s     pipeline.Storage
}

func Internal(s pipeline.Storage, log logger.Logger) *internalService {
	return &internalService{
		log:   log,
		s:     s,
		pipes: map[string]pipeUnit{},
		wg:    &sync.WaitGroup{},
	}
}

func (m *internalService) StartAll() error {
	pipes, err := m.s.List()
	if err != nil {
		return err
	}

	for _, pipeCfg := range pipes {
		m.pipes[pipeCfg.Settings.Id] = pipeUnit{pipeline.New(pipeCfg, nil), nil}
		if pipeCfg.Settings.Run {
			m.runPipeline(pipeCfg)
		}
	}

	return nil
}

func (m *internalService) StopAll() {
	for _, u := range m.pipes {
		if u.p.State() == pipeline.StateRunning {
			u.c()
		}
	}
	m.wg.Wait()
}

func (m *internalService) Start(id string) error {
	if unit, ok := m.pipes[id]; ok {
		switch unit.p.State() {
		case pipeline.StateRunning:
			return &pipeline.ConflictError{Err: errors.New("pipeline already running")}
		case pipeline.StateStopping:
			return &pipeline.ConflictError{Err: errors.New("pipeline stopping, please wait")}
		case pipeline.StateStarting:
			return &pipeline.ConflictError{Err: errors.New("pipeline starting, please wait")}
		}
	}

	pipeCfg, err := m.s.Get(id)
	if err != nil {
		return err
	}

	return m.runPipeline(pipeCfg)
}

func (m *internalService) Stop(id string) error {
	unit, ok := m.pipes[id]
	if !ok {
		return &pipeline.NotFoundError{Err: errors.New("pipeline unit not found in runtime registry")}
	}

	switch unit.p.State() {
	case pipeline.StateStopped, pipeline.StateCreated:
		return &pipeline.ConflictError{Err: errors.New("pipeline already stopped")}
	case pipeline.StateStopping:
		return &pipeline.ConflictError{Err: errors.New("pipeline stopping, please wait")}
	case pipeline.StateStarting:
		return &pipeline.ConflictError{Err: errors.New("pipeline starting, please wait")}
	}

	unit.c()
	return nil
}

func (m *internalService) State(id string) (string, error, error) {
	unit, ok := m.pipes[id]
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

func (m *internalService) Add(pipe *config.Pipeline) error {
	if err := m.s.Add(pipe); err != nil {
		return err
	}

	m.pipes[pipe.Settings.Id] = pipeUnit{pipeline.New(pipe, nil), nil}
	return nil
}

func (m *internalService) Update(pipe *config.Pipeline) error {
	unit, ok := m.pipes[pipe.Settings.Id]
	if !ok {
		return &pipeline.NotFoundError{Err: errors.New("pipeline unit not found in runtime registry")}
	}

	switch unit.p.State() {
	case pipeline.StateStopped, pipeline.StateCreated:
		m.pipes[pipe.Settings.Id] = pipeUnit{pipeline.New(pipe, nil), nil}
	default:
		return &pipeline.ConflictError{Err: errors.New("only stopped pipelines can be updated")}
	}

	return m.s.Update(pipe)
}

func (m *internalService) Delete(id string) error {
	unit, ok := m.pipes[id]
	if !ok {
		return &pipeline.NotFoundError{Err: errors.New("pipeline unit not found in runtime registry")}
	}

	switch unit.p.State() {
	case pipeline.StateStopped, pipeline.StateCreated:
		delete(m.pipes, id)
	default:
		return &pipeline.ConflictError{Err: errors.New("only stopped pipelines can be deleted")}
	}

	return m.s.Delete(id)
}

func (m *internalService) runPipeline(pipeCfg *config.Pipeline) error {
	pipe := pipeline.New(pipeCfg, logrus.NewLogger(map[string]any{
		"scope": "pipeline",
		"id":    pipeCfg.Settings.Id,
	}))

	m.log.Infof("building pipeline %v", pipeCfg.Settings.Id)
	if err := pipe.Build(); err != nil {
		m.log.Error(fmt.Errorf("pipeline %v building failed: %v", pipeCfg.Settings.Id, err.Error()))
		pipe.Close()
		m.log.Warnf("pipeline %v was closed as not ready for event processing", pipeCfg.Settings.Id)
		return &pipeline.ValidationError{Err: fmt.Errorf("pipeline %v building failed: %v", pipeCfg.Settings.Id, err.Error())}
	}

	ctx, cancel := context.WithCancel(context.Background())
	m.pipes[pipeCfg.Settings.Id] = pipeUnit{pipe, cancel}
	m.wg.Add(1)
	go func(p *pipeline.Pipeline) {
		p.Run(ctx)
		m.wg.Done()
		cancel()
	}(pipe)

	return nil
}
