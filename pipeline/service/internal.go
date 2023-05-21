package manager

import (
	"context"
	"fmt"
	"sync"

	"github.com/gekatateam/pipeline/config"
	"github.com/gekatateam/pipeline/logger/logrus"
	"github.com/gekatateam/pipeline/pipeline"
)

type internalService struct {
	pipes map[string]*pipeline.Pipeline
	ctxcf map[string]context.CancelFunc
	wg    *sync.WaitGroup
	s     pipeline.Storage
}

func NewInternalService(s pipeline.Storage) *internalService {
	return &internalService{
		s:     s,
		pipes: make(map[string]*pipeline.Pipeline),
		ctxcf: make(map[string]context.CancelFunc),
		wg:    &sync.WaitGroup{},
	}
}

func (m *internalService) StartAll() error {
	pipes, err := m.s.List()
	if err != nil {
		return err
	}

	for _, pipeCfg := range pipes {
		if err = m.runPipeline(pipeCfg); err != nil {
			m.StopAll()
			return err
		}
	}

	return nil
}

func (m *internalService) StopAll() {
	for _, f := range m.ctxcf {
		f()
	}
	m.wg.Wait()
}

func (m *internalService) Start(id string) error {
	pipeCfg, err := m.s.Get(id)
	if err != nil {
		return err
	}

	if _, ok := m.pipes[id]; ok {
		return pipeline.ErrPipelineRunning
	}

	return m.runPipeline(pipeCfg)
}

func (m *internalService) Stop(id string) error {
	pipeCfg, err := m.s.Get(id)
	if err != nil {
		return err
	}

	stop, ok := m.ctxcf[id]
	if !ok {
		return fmt.Errorf("pipeline %v cancelFunc not found in map", id)
	}

	stop()
	delete(m.ctxcf, id)
	delete(m.pipes, id)

	pipeCfg.Settings.Run = false
	err = m.s.Update(pipeCfg)
	if err != nil {
		return err
	}

	return nil
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

	if pipe.Settings.Run {
		return m.runPipeline(pipe)
	}

	return nil
}

func (m *internalService) Update(pipe *config.Pipeline) error {
	return m.s.Update(pipe)
}

func (m *internalService) Delete(id string) (*config.Pipeline, error) {
	err := m.Stop(id)
	if err != nil {
		return nil, err
	}

	return m.s.Delete(id)
}

func (m *internalService) runPipeline(pipeCfg *config.Pipeline) error {
	pipe := pipeline.New(pipeCfg, logrus.NewLogger(map[string]any{
		"scope": "pipeline",
		"id":    pipeCfg.Settings.Id,
	}))

	if err := pipe.Build(); err != nil {
		return fmt.Errorf("pipeline %v building failed: %v", pipeCfg.Settings.Id, err.Error())
	}

	ctx, cancel := context.WithCancel(context.Background())
	m.wg.Add(1)
	go func() {
		pipe.Run(ctx)
		m.wg.Done()
		delete(m.ctxcf, pipeCfg.Settings.Id)
		delete(m.pipes, pipeCfg.Settings.Id)
	}()

	m.pipes[pipeCfg.Settings.Id] = pipe
	m.ctxcf[pipeCfg.Settings.Id] = cancel

	return nil
}
