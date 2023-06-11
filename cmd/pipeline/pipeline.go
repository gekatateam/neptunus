package main

import (
	"fmt"

	"github.com/urfave/cli/v2"

	"github.com/gekatateam/neptunus/pipeline/api"
	"github.com/gekatateam/neptunus/pipeline/gateway"
)

func pipelineList(cCtx *cli.Context) error {
	cli := api.Cli(gateway.Rest(cCtx.String("server-address")))
	cli.List()
	return nil
}

func pipelineDescribe(cCtx *cli.Context) error {
	fmt.Println(cCtx.String("name"))
	fmt.Println(cCtx.String("format"))
	fmt.Println(cCtx.String("server-address"))
	return nil
}
