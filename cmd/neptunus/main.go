package main

import (
	"fmt"
	"os"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/urfave/cli/v2"
	"kythe.io/kythe/go/util/datasize"

	"github.com/gekatateam/neptunus/config"
	"github.com/gekatateam/neptunus/logger"
	"github.com/gekatateam/neptunus/pkg/memory"
)

var Version = "v.0.0.0"

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
		Version: Version,
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
				Name:  "worker",
				Usage: "run worker process",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "config",
						Value: "config.toml",
						Usage: "path to configuration file",
					},
					&cli.StringFlag{
						Name:     "pipeline",
						Required: true,
						Usage:    "pipeline name to run in worker",
					},
				},
				Action: worker,
			},
			{
				Name:  "pipeline",
				Usage: "cli commands for pipeline management",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "server-address",
						Aliases: []string{"s"},
						Value:   "http://localhost" + config.Default.Common.HttpPort,
						Usage:   "daemon http api server address; if sheme is HTTPS, tls transport will be used",
					},
					&cli.DurationFlag{
						Name:    "request-timeout",
						Aliases: []string{"t"},
						Value:   10 * time.Second,
						Usage:   "api call timeout",
					},
					&cli.StringFlag{
						Name:    "tls-key-file",
						Aliases: []string{"tk"},
						Value:   "",
						Usage:   "path to TLS key file",
					},
					&cli.StringFlag{
						Name:    "tls-cert-file",
						Aliases: []string{"tc"},
						Value:   "",
						Usage:   "path to TLS certificate file",
					},
					&cli.StringFlag{
						Name:    "tls-ca-file",
						Aliases: []string{"ta"},
						Value:   "",
						Usage:   "path to TLS CA file",
					},
					&cli.BoolFlag{
						Name:    "tls-skip-verify",
						Aliases: []string{"ts"},
						Value:   false,
						Usage:   "skip TLS certificate verification",
					},
					&cli.StringSliceFlag{
						Name:    "header",
						Aliases: []string{"H"},
						Usage:   "custom header to add to request (can be repeated), format: Key:Value",
					},
				},
				Before: cliController.Init,
				Subcommands: []*cli.Command{
					{
						Name:   "list",
						Usage:  "list all pipelines (short info)",
						Action: cliController.List,
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:  "format",
								Value: "plain",
								Usage: "list format (plain, json, yaml supported)",
							},
						},
					},
					{
						Name:      "describe",
						Usage:     "describe pipeline by name",
						UsageText: "describe --name my-pipeline [--format yaml]",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:     "name",
								Aliases:  []string{"n", "id"},
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
								Aliases:  []string{"n", "id"},
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
								Aliases:  []string{"n", "id"},
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
								Aliases:  []string{"n", "id"},
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
		logger.Default.Error("we're failed",
			"error", err,
		)
		os.Exit(1)
	}
}

func SetRuntimeParameters(config *config.Runtime) error {
	if len(config.GCPercent) > 0 {
		gcPercent, err := strconv.Atoi(strings.TrimSuffix(config.GCPercent, "%"))
		if err != nil {
			return err
		}

		debug.SetGCPercent(gcPercent)
		logger.Default.Info(fmt.Sprintf("GC percent is set to %v", config.GCPercent))
	}

	if config.MaxThreads > 0 {
		debug.SetMaxThreads(config.MaxThreads)
		logger.Default.Info(fmt.Sprintf("max threads number is set to %v", config.MaxThreads))
	}

	if config.MaxProcs > 0 {
		runtime.GOMAXPROCS(config.MaxProcs)
		logger.Default.Info(fmt.Sprintf("max procs number is set to %v", config.MaxProcs))
	}

	if len(config.MemLimit) > 0 {
		var memLimit uint64
		if strings.HasSuffix(config.MemLimit, "%") {
			if memory.TotalMemory() == 0 {
				logger.Default.Warn("unable to set percentage memory limit on current system")
				return nil
			}

			memLimitPercent, err := strconv.ParseUint(strings.TrimSuffix(config.MemLimit, "%"), 10, 0)
			if err != nil {
				return err
			}

			memLimit = memory.TotalMemory() * memLimitPercent / 100
		} else {
			size, err := datasize.Parse(config.MemLimit)
			if err != nil {
				return err
			}

			memLimit = size.Bytes()
		}

		debug.SetMemoryLimit(int64(memLimit))
		logger.Default.Info(fmt.Sprintf("memory limit is set to %v", datasize.Size(memLimit)))
	}

	return nil
}
