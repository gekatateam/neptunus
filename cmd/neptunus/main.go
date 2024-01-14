package main

import (
	"os"
	"runtime/debug"

	"github.com/urfave/cli/v2"

	"github.com/gekatateam/neptunus/config"
	"github.com/gekatateam/neptunus/logger"
)

func main() {
	defer func() {
		if r := recover(); r != nil {
			logger.Default.Error(
				"unexpected panic recovered",
				"error", r,
				"stack_trace", string(debug.Stack()),
			)
		}
	}()

	app := &cli.App{
		Name:    "neptunus",
		Version: "v0.1.0",
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
				Usage: "test pipelines from configured storage without run",
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
				Usage: "cli commands for pipeline management",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "server-address",
						Value: "http://localhost" + config.Default.Common.HttpPort,
						Usage: "daemon http api server address",
					},
				},
				Before: cliController.Init,
				Subcommands: []*cli.Command{
					{
						Name:   "list",
						Usage:  "list all pipelines",
						Action: cliController.List,
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
						Action: cliController.Describe,
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
						Action: cliController.Deploy,
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
						Action: cliController.Update,
					},
					{
						Name:      "delete",
						Usage:     "delete pipeline by name",
						UsageText: "delete --name my-pipeline",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:     "name",
								Aliases:  []string{"n"},
								Required: true,
								Usage:    "pipeline name",
							},
							// &cli.BoolFlag{
							// 	Name:  "force",
							// 	Value: false,
							// 	Usage: "stop pipeline, if it's running, then delete",
							// },
						},
						Action: cliController.Delete,
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
						Action: cliController.Start,
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
						Action: cliController.Stop,
					},
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		logger.Default.Error("application startup failed",
			"error", err,
		)
	}
}
