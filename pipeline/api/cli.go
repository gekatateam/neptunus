package api

import (
	"bytes"
	"errors"
	"fmt"
	"text/tabwriter"

	"github.com/gekatateam/neptunus/pipeline"
)

var errCliFailed = errors.New("cli command exec failed")

type cliApi struct {
	g pipeline.Service
}

func Cli(gateway pipeline.Service) *cliApi {
	return &cliApi{
		g: gateway,
	}
}

func (a *cliApi) Start(id string) error {
	return nil
}
func (a *cliApi) Stop(id string) error {
	return nil
}
func (a *cliApi) State(id string) error {
	return nil
}
func (a *cliApi) List() error {
	pipes, err := a.g.List()
	if err != nil {
		fmt.Printf("cli list: exec failed - %v\n", err)
		return errCliFailed
	}

	b := new(bytes.Buffer)
	w := tabwriter.NewWriter(b, 1, 1, 1, ' ', 0)
	fmt.Fprintf(w, "%v\t%v\t%v\n", "id", "state", "autorun")

	for _, pipe := range pipes {
		state, err := a.g.State(pipe.Settings.Id)
		if err != nil {
			fmt.Printf("cli list: exec failed - %v", err)
			return errCliFailed
		}
		fmt.Fprintf(w, "%v\t%v\t%v\n", pipe.Settings.Id, state, pipe.Settings.Run)
	}
	w.Flush()
	fmt.Printf(b.String())

	return nil
}
func (a *cliApi) Get(id string, format string) error {
	return nil
}
func (a *cliApi) Add(file string) error {
	return nil
}
func (a *cliApi) Update(file string) error {
	return nil
}
func (a *cliApi) Delete(id string) error {
	return nil
}
