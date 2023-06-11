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
						Name:   "list",
						Usage:  "list all pipelines",
						Action: pipelineList,
					},
					{
						Name:      "describe",
						Usage:     "describe pipeline by name",
						UsageText: "describe --name my-pipeline [--format yaml]",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:     "name",
								Aliases:  []string{"n"},
								Required: true,
								Usage:    "pipeline name",
							},
							&cli.StringFlag{
								Name:  "format",
								Value: "toml",
								Usage: "pipeline printing format (json, toml, yaml supported)",
							},
						},
						Action: pipelineDescribe,
					},
					{
						Name:      "deploy",
						Usage:     "deploy new pipeline from file (json, toml, yaml supported)",
						UsageText: "deploy --file pipeline.toml",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:     "file",
								Aliases:  []string{"f"},
								Required: true,
								Usage:    "pipeline manifest file (json, toml, yaml supported)",
							},
						},
					},
					{
						Name:      "update",
						Usage:     "update existing pipeline from file (json, toml, yaml supported)",
						UsageText: "update --file pipeline.toml",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:     "file",
								Aliases:  []string{"f"},
								Required: true,
								Usage:    "pipeline manifest file (json, toml, yaml supported)",
							},
						},
					},
					{
						Name:      "delete",
						Usage:     "delete pipeline by name",
						UsageText: "delete --name my-pipeline [--force]",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:     "name",
								Aliases:  []string{"n"},
								Required: true,
								Usage:    "pipeline name",
							},
							&cli.BoolFlag{
								Name:  "force",
								Value: false,
								Usage: "stop pipeline, if it's running, then delete",
							},
						},
					},
					{
						Name:      "start",
						Usage:     "start pipeline by name",
						UsageText: "start --name my-pipeline",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:     "name",
								Aliases:  []string{"n"},
								Required: true,
								Usage:    "pipeline name",
							},
						},
					},
					{
						Name:      "stop",
						Usage:     "stop pipeline by name",
						UsageText: "stop --name my-pipeline",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:     "name",
								Aliases:  []string{"n"},
								Required: true,
								Usage:    "pipeline name",
							},
						},
					},
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatalf("application error: %v", err.Error())
	}
}
