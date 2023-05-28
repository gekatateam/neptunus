package main

import (
	"os"
	"runtime/debug"

	"github.com/urfave/cli/v2"

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
		Name:    "pipeline",
		Version: "v0.0.1",
		Commands: []*cli.Command{
			{
				Name:  "run",
				Usage: "run configured pipelines",
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
				Usage: "test configured pipelines",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "config",
						Value: "config.toml",
						Usage: "path to configuration file",
					},
				},
				Action: test,
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatalf("application error: %v", err.Error())
	}
}
