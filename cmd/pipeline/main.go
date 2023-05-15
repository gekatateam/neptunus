package main

import (
	"fmt"
	"os"

	"github.com/urfave/cli/v2"

	"github.com/gekatateam/pipeline/logger/logrus"
)

var log = logrus.NewLogger(map[string]any{"scope": "main"})

func main() {
    app := &cli.App{
        Name:  "boom",
        Usage: "make an explosive entrance",
        Action: func(*cli.Context) error {
            fmt.Println("boom! I say!")
            return nil
        },
    }

    if err := app.Run(os.Args); err != nil {
        log.Fatalf("application startup error: %v", err.Error())
    }
}
