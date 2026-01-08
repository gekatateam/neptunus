package config

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"math/rand"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/goccy/go-yaml"
)

type Pipeline struct {
	Settings   PipeSettings   `toml:"settings"   yaml:"settings"   json:"settings"`
	Vars       map[string]any `toml:"vars"       yaml:"vars"       json:"vars"`
	Inputs     []PluginSet    `toml:"inputs"     yaml:"inputs"     json:"inputs"`
	Processors []PluginSet    `toml:"processors" yaml:"processors" json:"processors"`
	Outputs    []PluginSet    `toml:"outputs"    yaml:"outputs"    json:"outputs"`
	Keykeepers []PluginSet    `toml:"keykeepers" yaml:"keykeepers" json:"keykeepers"`
}

type PipeSettings struct {
	Id          string `toml:"id"          yaml:"id"          json:"id"`
	Lines       int    `toml:"lines"       yaml:"lines"       json:"lines"`
	Run         bool   `toml:"run"         yaml:"run"         json:"run"`
	Buffer      int    `toml:"buffer"      yaml:"buffer"      json:"buffer"`
	Consistency string `toml:"consistency" yaml:"consistency" json:"consistency"`
	LogLevel    string `toml:"log_level"   yaml:"log_level"   json:"log_level"`
}

type PluginSet map[string]Plugin

type Plugin map[string]any

func (p Plugin) Id() uint64 {
	if rawId, ok := p["::plugin_id"]; ok {
		if id, ok := rawId.(uint64); ok {
			return id
		}
	}

	id := rand.Uint64()
	p["::plugin_id"] = id
	return id
}

func (p Plugin) Alias() string {
	aliasRaw, ok := p["alias"]
	if !ok {
		return ""
	}
	alias, ok := aliasRaw.(string)
	if !ok {
		return ""
	}
	return alias
}

func (p Plugin) Reverse() bool {
	reverseRaw, ok := p["reverse"]
	if !ok {
		return false
	}
	reverse, ok := reverseRaw.(bool)
	if !ok {
		return false
	}
	return reverse
}

func (p Plugin) Type() string {
	typeRaw, ok := p["type"]
	if !ok {
		return ""
	}
	typeStr, ok := typeRaw.(string)
	if !ok {
		return ""
	}
	return typeStr
}

func (p Plugin) LogLevel() string {
	logLevelRaw, ok := p["log_level"]
	if !ok {
		return ""
	}
	logLevel, ok := logLevelRaw.(string)
	if !ok {
		return ""
	}
	return logLevel
}

func (p Plugin) Parser() Plugin {
	parserRaw, ok := p["parser"]
	if !ok {
		return nil
	}

	parser, ok := parserRaw.(map[string]any)
	if !ok {
		return nil
	}

	return parser
}

func (p Plugin) Serializer() Plugin {
	serializerRaw, ok := p["serializer"]
	if !ok {
		return nil
	}

	serializer, ok := serializerRaw.(map[string]any)
	if !ok {
		return nil
	}

	return serializer
}

func (p Plugin) Filters() PluginSet {
	filtersRaw, ok := p["filters"]
	if !ok {
		return nil
	}
	filtersSet, ok := filtersRaw.(map[string]any)
	if !ok {
		return nil
	}
	var filters = make(PluginSet, len(filtersSet))
	for key, value := range filtersSet {
		filterCfg, ok := value.(map[string]any)
		if !ok {
			filters[key] = Plugin{}
			continue
		}
		filters[key] = Plugin(filterCfg)
	}
	return filters
}

func (p Plugin) Compressor() (Plugin, string) {
	compressorNameRaw, ok := p["compressor"]
	if !ok {
		return nil, ""
	}

	compressorName, ok := compressorNameRaw.(string)
	if !ok {
		return nil, ""
	}

	compressor := make(Plugin, len(p))
	for k, v := range p {
		if strings.HasPrefix(k, compressorName+"_") {
			compressor[k] = v
		}
	}

	return compressor, compressorName
}

func (p Plugin) Decompressor() (Plugin, string) {
	decompressorNameRaw, ok := p["decompressor"]
	if !ok {
		return nil, ""
	}

	decompressorName, ok := decompressorNameRaw.(string)
	if !ok {
		return nil, ""
	}

	decompressor := make(Plugin, len(p))
	for k, v := range p {
		if strings.HasPrefix(k, decompressorName+"_") {
			decompressor[k] = v
		}
	}

	return decompressor, decompressorName
}

func UnmarshalPipeline(data []byte, format string) (*Pipeline, error) {
	pipeline := Pipeline{
		Vars: make(map[string]any),
	}

	switch format {
	case ".toml":
		if err := toml.Unmarshal(data, &pipeline); err != nil {
			return &pipeline, err
		}
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, &pipeline); err != nil {
			return &pipeline, err
		}
	case ".json":
		if err := json.Unmarshal(data, &pipeline,
			json.WithUnmarshalers(json.UnmarshalFromFunc(JsonStrictNumberUnmarshal))); err != nil {
			return &pipeline, err
		}
	default:
		return &pipeline, fmt.Errorf("unknown pipeline extension: %v", format)
	}

	return SetPipelineDefaults(&pipeline), nil
}

func MarshalPipeline(pipe *Pipeline, format string) ([]byte, error) {
	var content = []byte{}
	var err error

	switch format {
	case ".toml":
		if content, err = toml.Marshal(pipe); err != nil {
			return nil, err
		}
	case ".yaml", ".yml":
		if content, err = yaml.Marshal(pipe); err != nil {
			return nil, err
		}
	case ".json":
		if content, err = json.Marshal(pipe); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unknown pipeline extension: %v", format)
	}

	return content, nil
}

func SetPipelineDefaults(pipe *Pipeline) *Pipeline {
	if pipe.Settings.Lines <= 0 {
		pipe.Settings.Lines = 1
	}

	if pipe.Settings.Buffer < 0 {
		pipe.Settings.Buffer = 0
	}

	if pipe.Settings.Consistency == "" {
		pipe.Settings.Consistency = "soft"
	}

	return pipe
}

func JsonStrictNumberUnmarshal(dec *jsontext.Decoder, val *any) error {
	if dec.PeekKind() == '0' {
		v, err := dec.ReadValue()
		if err != nil {
			return err
		}

		if i, err := strconv.ParseInt(string(v), 10, 64); err == nil {
			*val = i
			return nil
		}

		if u, err := strconv.ParseUint(string(v), 10, 64); err == nil {
			*val = u
			return nil
		}

		if f, err := strconv.ParseFloat(string(v), 64); err == nil {
			*val = f
			return nil
		}

		return fmt.Errorf("cannot parse number: %s; int, uint, float failed", string(v))
	}

	return json.SkipFunc
}

func JsonUnmarshalStrict[T any](data []byte, vars T) error {
	return json.Unmarshal(data, vars,
		json.WithUnmarshalers(json.UnmarshalFromFunc(JsonStrictNumberUnmarshal)))
}
