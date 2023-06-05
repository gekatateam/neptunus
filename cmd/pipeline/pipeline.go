package main

import (
	"fmt"

	"github.com/urfave/cli/v2"
)

func pipelineList(cCtx *cli.Context) error {
	return nil
}

func pipelineDescribe(cCtx *cli.Context) error {
	fmt.Println(cCtx.String("name"))
	fmt.Println(cCtx.String("format"))
	fmt.Println(cCtx.String("server-address"))
	return nil
}
