package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"text/tabwriter"

	"github.com/goccy/go-yaml"
)

type pipelineShortInfo struct {
	Id      string `json:"id"         yaml:"id"`
	Autorun bool   `json:"autorun"    yaml:"autorun"`
	State   string `json:"state"      yaml:"state"`
	LastErr string `json:"last_error" yaml:"last_error"`
}

func printShortInfo(format string, info []pipelineShortInfo) (string, error) {
	switch format {
	case "plain":
		b := new(bytes.Buffer)
		w := tabwriter.NewWriter(b, 1, 1, 1, ' ', 0)
		fmt.Fprintf(w, "%v\t%v\t%v\t%v\n", "id", "autorun", "state", "last_error")
		fmt.Fprintf(w, "%v\t%v\t%v\t%v\n", "--", "-------", "-----", "----------")

		for _, pipe := range info {
			fmt.Fprintf(w, "%v\t%v\t%v\t%v\n", pipe.Id, pipe.Autorun, pipe.State, emptyStringAsNil(pipe.LastErr))
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

func emptyStringAsNil(s string) string {
	if len(s) == 0 {
		return "<nil>"
	}
	return s
}

func errAsString(err error) string {
	if err != nil {
		return err.Error()
	}
	return ""
}
