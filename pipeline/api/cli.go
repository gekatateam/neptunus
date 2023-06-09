package api

import (
	"fmt"
	"io"
	"os"

	"github.com/gekatateam/neptunus/config"
	"github.com/gekatateam/neptunus/pipeline"
)

type cliApi struct {
	out io.Writer
	g   pipeline.Service
}

func Cli(gateway pipeline.Service) *cliApi {
	return &cliApi{
		out: os.Stdout,
		g:   gateway,
	}
}

func(a *cliApi) Start(id string) error {
	return nil
}
func(a *cliApi) Stop(id string) error{
	return nil
}
func(a *cliApi) State(id string) error{
	return nil
}
func(a *cliApi) List(format string) error{
	pipes, err := a.g.List()
	if err != nil {
		return err
	}

	for _, pipe := range pipes {
		rawPipe, err := config.MarshalPipeline(pipe, format)
		if err != nil {
			return err
		}
		fmt.Fprint(a.out, string(rawPipe), "\n")
		fmt.Fprint(a.out, "------------")
	}

	return nil
}
func(a *cliApi) Get(id string, format string) error{
	return nil
}
func(a *cliApi) Add(file string) error{
	return nil
}
func(a *cliApi) Update(file string) error{
	return nil
}
func(a *cliApi) Delete(id string) error{
	return nil
}
