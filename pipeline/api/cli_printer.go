package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"text/tabwriter"

	"github.com/goccy/go-yaml"

	"github.com/gekatateam/neptunus/config"
)

type pipelineShortInfo struct {
	Id      string `json:"id"      yaml:"id"`
	State   string `json:"state"   yaml:"state"`
	Autorun bool   `json:"autorun" yaml:"autorun"`
	LastErr error  `json:"error"   yaml:"error"`
}

func printShortInfo(format string, info []pipelineShortInfo) (string, error) {
	switch format {
	case "plain":
		b := new(bytes.Buffer)
		w := tabwriter.NewWriter(b, 1, 1, 1, ' ', 0)
		fmt.Fprintf(w, "%v\t%v\t%v\t%v\n", "id", "state", "autorun", "error")
		fmt.Fprintf(w, "%v\t%v\t%v\t%v\n", "--", "-----", "-------", "-----")

		for _, pipe := range info {
			fmt.Fprintf(w, "%v\t%v\t%v\t%v\n", pipe.Id, pipe.State, pipe.Autorun, pipe.LastErr)
		}
		w.Flush()
		return b.String(), nil
	case "json":
		result, err := json.Marshal(info)
		return string(result), err
	case "yaml":
		result, err := yaml.Marshal(info)
		return string(result), err
	default:
		return "", fmt.Errorf("unknown format: %v", format)
	}
}

type pipelineFullInfo struct {
	Manifest config.Pipeline     `toml:"manifest" yaml:"manifest" json:"manifest"`
	Runtime  pipelineRuntimeInfo `toml:"runtime"  yaml:"runtime"  json:"runtime"`
}

type pipelineRuntimeInfo struct {
	State   string `json:"state" yaml:"state"`
	LastErr error  `json:"error" yaml:"error"`
}
