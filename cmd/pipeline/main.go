package main

import (
	"os"
	"runtime/debug"

	"github.com/urfave/cli/v2"

	"github.com/gekatateam/neptunus/config"
	"github.com/gekatateam/neptunus/logger/logrus"
)

var log = logrus.NewLogger(map[string]any{"scope": "main"})

func main() {
	defer func() {
		if r := recover(); r != nil {
			log.Fatalf("unexpected panic recovered: %v; stack trace: %v", r, string(debug.Stack()))
		}
	}()

	app := &cli.App{
		Name:    "neptunus",
		Version: "v0.0.1",
		Commands: []*cli.Command{
			{
				Name:  "run",
				Usage: "run daemon with configured pipelines",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "config",
						Value: "config.toml",
						Usage: "path to configuration file",
					},
				},
				Action: run,
			},
			{
				Name:  "test",
				Usage: "test pipelines in configured storage without run",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "config",
						Value: "config.toml",
						Usage: "path to configuration file",
					},
				},
				Action: test,
			},
			{
				Name:  "pipeline",
				Usage: "daemon commands for pipeline management",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "server-address",
						Value: "http://localhost" + config.Default.Common.HttpPort,
						Usage: "daemon http api server address",
					},
				},
				Subcommands: []*cli.Command{
					{
						Name:  "list",
						Usage: "list all pipelines",
					},
					{
						Name:      "describe",
						Usage:     "describe pipeline by name",
						UsageText: "describe my-pipeline [--format yaml]",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:  "format",
								Value: "toml",
								Usage: "pipeline printing format (json, toml, yaml formats supported)",
							},
						},
					},
					{
						Name:      "deploy",
						Usage:     "deploy new pipeline from file (json, toml, yaml formats supported)",
						UsageText: "deploy pipeline.toml",
					},
					{
						Name:      "update",
						Usage:     "update existing pipeline from file (json, toml, yaml formats supported)",
						UsageText: "update pipeline.toml",
					},
					{
						Name:      "delete",
						Usage:     "delete pipeline by name",
						UsageText: "delete my-pipeline",
					},
					{
						Name:      "start",
						Usage:     "start pipeline by name",
						UsageText: "start my-pipeline",
					},
					{
						Name:      "stop",
						Usage:     "stop pipeline by name",
						UsageText: "stop my-pipeline",
					},
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatalf("application error: %v", err.Error())
	}
}
