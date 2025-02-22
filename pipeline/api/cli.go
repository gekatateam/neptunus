package api

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"

	"github.com/gekatateam/neptunus/config"
	"github.com/gekatateam/neptunus/pipeline"
	"github.com/gekatateam/neptunus/pipeline/gateway"
	"github.com/urfave/cli/v2"
)

type cliApi struct {
	gw pipeline.Service
}

func Cli(gateway pipeline.Service) *cliApi {
	return &cliApi{
		gw: gateway,
	}
}

func (c *cliApi) Init(cCtx *cli.Context) error {
	c.gw = gateway.Rest(cCtx.String("server-address"), "api/v1/pipelines", cCtx.Duration("request-timeout"))
	return nil
}

func (c *cliApi) List(_ *cli.Context) error {
	pipes, err := c.gw.List()
	if err != nil {
		fmt.Printf("cli list: exec failed - %v\n", err)
		os.Exit(1)
	}

	b := new(bytes.Buffer)
	w := tabwriter.NewWriter(b, 1, 1, 1, ' ', 0)
	fmt.Fprintf(w, "%v\t%v\t%v\t%v\n", "id", "state", "autorun", "error")
	fmt.Fprintf(w, "%v\t%v\t%v\t%v\n", "--", "-----", "-------", "-----")

	for _, pipe := range pipes {
		state, lastErr, err := c.gw.State(pipe.Settings.Id)
		if err != nil {
			fmt.Printf("cli list: exec failed - %v", err)
			os.Exit(1)
		}
		fmt.Fprintf(w, "%v\t%v\t%v\t%v\n", pipe.Settings.Id, state, pipe.Settings.Run, lastErr)
	}
	w.Flush()
	fmt.Print(b.String())

	return nil
}

func (c *cliApi) Describe(cCtx *cli.Context) error {
	id := cCtx.String("name")
	pipe, err := c.gw.Get(id)
	switch {
	case err == nil:
	case errors.As(err, &pipeline.NotFoundErr):
		fmt.Printf("pipeline %v not found: %v\n", id, err.Error())
		os.Exit(1)
	default:
		fmt.Printf("cli describe: exec failed - %v\n", err.Error())
		os.Exit(1)
	}

	state, lastErr, err := c.gw.State(id)
	if err != nil {
		fmt.Printf("cli describe: exec failed - %v\n", err.Error())
		os.Exit(1)
	}

	rawPipe, err := config.MarshalPipeline(pipe, "."+cCtx.String("format"))
	if err != nil {
		fmt.Printf("cli describe: exec failed - %v\n", err.Error())
		os.Exit(1)
	}

	fmt.Printf("Manifest:\n")
	fmt.Printf("%v\n", string(rawPipe))
	fmt.Printf("\n")
	fmt.Printf("Runtime:\n")
	fmt.Printf("state: %v\n", state)
	fmt.Printf("error: %v\n", lastErr)

	return nil
}

func (c *cliApi) Start(cCtx *cli.Context) error {
	id := cCtx.String("name")
	fmt.Printf("starting pipeline %v\n", id)
	err := c.gw.Start(id)
	switch {
	case err == nil:
		fmt.Printf("pipeline %v accepts start signal\n", id)
		return nil
	case errors.As(err, &pipeline.NotFoundErr):
		fmt.Printf("pipeline %v startup failed: pipeline not found: %v\n", id, err.Error())
		os.Exit(1)
	case errors.As(err, &pipeline.ConflictErr):
		fmt.Printf("pipeline %v startup failed: conflict state: %v\n", id, err.Error())
		os.Exit(1)
	default:
		fmt.Printf("cli start: exec failed - %v\n", err.Error())
		os.Exit(1)
	}
	return nil
}

func (c *cliApi) Stop(cCtx *cli.Context) error {
	id := cCtx.String("name")
	fmt.Printf("stopping pipeline %v\n", id)
	err := c.gw.Stop(id)
	switch {
	case err == nil:
		fmt.Printf("pipeline %v accepts stop signal\n", id)
		return nil
	case errors.As(err, &pipeline.NotFoundErr):
		fmt.Printf("pipeline %v stop failed: pipeline not found: %v\n", id, err.Error())
		os.Exit(1)
	case errors.As(err, &pipeline.ConflictErr):
		fmt.Printf("pipeline %v stop failed: conflict state: %v\n", id, err.Error())
		os.Exit(1)
	default:
		fmt.Printf("cli stop: exec failed - %v\n", err.Error())
		os.Exit(1)
	}
	return nil
}

func (c *cliApi) Deploy(cCtx *cli.Context) error {
	file := cCtx.String("file")
	fmt.Printf("deploying new pipeline from file %v\n", file)

	rawPipe, err := os.ReadFile(file)
	if err != nil {
		fmt.Printf("cli deploy: exec failed - file read error: %v\n", err.Error())
		os.Exit(1)
	}

	pipe, err := config.UnmarshalPipeline(rawPipe, filepath.Ext(file))
	if err != nil {
		fmt.Printf("cli deploy: exec failed - unmarshal pipeline error: %v\n", err.Error())
		os.Exit(1)
	}

	err = c.gw.Add(pipe)
	switch {
	case err == nil:
		fmt.Printf("pipeline %v successfully deployed\n", pipe.Settings.Id)
	case errors.As(err, &pipeline.ConflictErr):
		fmt.Printf("pipeline %v deploy failed: conflict state: %v\n", pipe.Settings.Id, err.Error())
		os.Exit(1)
	default:
		fmt.Printf("cli deploy: exec failed - %v\n", err.Error())
		os.Exit(1)
	}

	return nil
}

func (c *cliApi) Update(cCtx *cli.Context) error {
	file := cCtx.String("file")
	fmt.Printf("updating exists pipeline from file %v\n", file)

	rawPipe, err := os.ReadFile(file)
	if err != nil {
		fmt.Printf("cli update: exec failed - file read error: %v\n", err.Error())
		os.Exit(1)
	}

	pipe, err := config.UnmarshalPipeline(rawPipe, filepath.Ext(file))
	if err != nil {
		fmt.Printf("cli update: exec failed - unmarshal pipeline error: %v\n", err.Error())
		os.Exit(1)
	}

	err = c.gw.Update(pipe)
	switch {
	case err == nil:
		fmt.Printf("pipeline %v successfully updated\n", pipe.Settings.Id)
	case errors.As(err, &pipeline.NotFoundErr):
		fmt.Printf("pipeline %v update failed: pipeline not found: %v\n", pipe.Settings.Id, err.Error())
		os.Exit(1)
	case errors.As(err, &pipeline.ConflictErr):
		fmt.Printf("pipeline %v update failed: conflict state: %v\n", pipe.Settings.Id, err.Error())
		os.Exit(1)
	default:
		fmt.Printf("cli update: exec failed - %v\n", err.Error())
		os.Exit(1)
	}

	return nil
}

func (c *cliApi) Delete(cCtx *cli.Context) error {
	id := cCtx.String("name")
	fmt.Printf("deleting pipeline %v\n", id)
	err := c.gw.Delete(id)
	switch {
	case err == nil:
		fmt.Printf("pipeline %v successfully deleted\n", id)
	case errors.As(err, &pipeline.NotFoundErr):
		fmt.Printf("pipeline %v delete failed: pipeline not found: %v\n", id, err.Error())
		os.Exit(1)
	case errors.As(err, &pipeline.ConflictErr):
		fmt.Printf("pipeline %v delete failed: conflict state: %v\n", id, err.Error())
		os.Exit(1)
	default:
		fmt.Printf("cli delete: exec failed - %v\n", err.Error())
		os.Exit(1)
	}

	return nil
}
