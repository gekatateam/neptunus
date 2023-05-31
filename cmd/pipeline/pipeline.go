package main

import "github.com/urfave/cli/v2"

func pipelineList(cCtx *cli.Context) error {
	var headers = "name\tstate"
	var status = "%v\t%v"
	return nil
}

func pipelineDescribe(cCtx *cli.Context) error {
	var okMessage = `
state: %v

configuration:\n%v
\n`

	return nil
}

