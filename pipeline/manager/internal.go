package manager

import (
	"fmt"
	"sync"

	"github.com/gekatateam/pipeline/config"
	"github.com/gekatateam/pipeline/logger/logrus"
	"github.com/gekatateam/pipeline/pipeline"
)

type internalManager struct {
	pipes map[string]*pipeline.Pipeline
	wg    *sync.WaitGroup
	s     pipeline.Storage
}

func (m *internalManager) StartAll() error {
	pipes, err := m.s.List()
	if err != nil {
		return err
	}

	for _, pipeCfg := range pipes {
		pipe := pipeline.New(pipeCfg, logrus.NewLogger(map[string]any{
			"scope": "pipeline",
			"id":    pipeCfg.Settings.Id,
		}))
		if err = pipe.Build(); err != nil {
			return fmt.Errorf("pipeline %v building failed: %v", pipeCfg.Settings.Id, err.Error())
		}
		m.wg.Add(1)
		go func() {
			pipe.Run()
		}()
	}
}

func (m *internalManager) StopAll()
func (m *internalManager) Start(id string) error
func (m *internalManager) Stop(id string)
func (m *internalManager) List() ([]*config.Pipeline, error)
func (m *internalManager) Get(id string) (*config.Pipeline, error)
func (m *internalManager) Add(pipeline *config.Pipeline) error
func (m *internalManager) Update(pipeline *config.Pipeline) error
func (m *internalManager) Delete(id string) (*config.Pipeline, error)