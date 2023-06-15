package service

// import (
// 	"github.com/gekatateam/neptunus/config"
// 	"github.com/gekatateam/neptunus/pipeline"
// )

// type externalService struct {
// 	g pipeline.Service
// }

// func External(g pipeline.Service) *externalService {
// 	return &externalService{g: g}
// }

// func (s *externalService) Start(id string) error
// func (s *externalService) Stop(id string) error
// func (s *externalService) State(id string) (string, error, error)
// func (s *externalService) List() ([]*config.Pipeline, error)
// func (s *externalService) Get(id string) (*config.Pipeline, error)
// func (s *externalService) Add(pipe *config.Pipeline) error
// func (s *externalService) Update(pipe *config.Pipeline) error
// func (s *externalService) Delete(id string) error
