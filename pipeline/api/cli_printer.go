package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"text/tabwriter"

	"github.com/BurntSushi/toml"
	"github.com/goccy/go-yaml"

	"github.com/gekatateam/neptunus/config"
)

type pipelineShortInfo struct {
	Id      string `json:"id"         yaml:"id"`
	State   string `json:"state"      yaml:"state"`
	Autorun bool   `json:"autorun"    yaml:"autorun"`
	LastErr string `json:"last_error" yaml:"last_error"`
}

func printShortInfo(format string, info []pipelineShortInfo) (string, error) {
	switch format {
	case "plain":
		b := new(bytes.Buffer)
		w := tabwriter.NewWriter(b, 1, 1, 1, ' ', 0)
		fmt.Fprintf(w, "%v\t%v\t%v\t%v\n", "id", "state", "autorun", "last_error")
		fmt.Fprintf(w, "%v\t%v\t%v\t%v\n", "--", "-----", "-------", "----------")

		for _, pipe := range info {
			fmt.Fprintf(w, "%v\t%v\t%v\t%v\n", pipe.Id, pipe.State, pipe.Autorun, emptyStringAsNil(pipe.LastErr))
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
	*config.Pipeline `yaml:",inline"`
	Runtime          pipelineRuntimeInfo `json:"runtime"  yaml:"runtime"  toml:"runtime"`
}

type pipelineRuntimeInfo struct {
	State   string `json:"state"      yaml:"state"      toml:"state"`
	LastErr string `json:"last_error" yaml:"last_error" toml:"last_error"`
}

func printFullInfo(format string, pipeline *config.Pipeline, state string, lastErr error) (string, error) {
	info := pipelineFullInfo{
		Pipeline: pipeline,
		Runtime: pipelineRuntimeInfo{
			State:   state,
			LastErr: errAsString(lastErr),
		},
	}

	switch format {
	case "toml":
		result, err := toml.Marshal(info)
		return string(result), err
	case "yaml":
		result, err := yaml.Marshal(info)
		return string(result), err
	case "json":
		result, err := json.Marshal(info)
		return string(result), err
	default:
		return "", fmt.Errorf("unknown format: %v", format)
	}
}

func errAsString(err error) string {
	if err != nil {
		return err.Error()
	}
	return ""
}

func emptyStringAsNil(s string) string {
	if len(s) == 0 {
		return "<nil>"
	}
	return s
}
