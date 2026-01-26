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

// reserved plugin configuration keys
const (
	KeyPluginId     = "::plugin_id"
	KeyAlias        = "alias"
	KeyReverse      = "reverse"
	KeyType         = "type"
	KeyLogLevel     = "log_level"
	KeyLookup       = "lookup"
	KeyParser       = "parser"
	KeySerializer   = "serializer"
	KeyFilters      = "filters"
	KeyCompressor   = "compressor"
	KeyDecompressor = "decompressor"
)

type Pipeline struct {
	Settings   PipeSettings `toml:"settings"          yaml:"settings"          json:"settings"`
	Vars       PipeVars     `toml:"vars"              yaml:"vars"              json:"vars"`
	Inputs     []PluginSet  `toml:"inputs"            yaml:"inputs"            json:"inputs"`
	Processors []PluginSet  `toml:"processors"        yaml:"processors"        json:"processors"`
	Outputs    []PluginSet  `toml:"outputs"           yaml:"outputs"           json:"outputs"`
	Keykeepers []PluginSet  `toml:"keykeepers"        yaml:"keykeepers"        json:"keykeepers"`
	Lookups    []PluginSet  `toml:"lookups"           yaml:"lookups"           json:"lookups"`
	Runtime    *PipeRuntime `toml:"runtime,omitempty" yaml:"runtime,omitempty" json:"runtime,omitempty"`
}

type PipeSettings struct {
	Id          string `toml:"id"          yaml:"id"          json:"id"`
	Lines       int    `toml:"lines"       yaml:"lines"       json:"lines"`
	Run         bool   `toml:"run"         yaml:"run"         json:"run"`
	Buffer      int    `toml:"buffer"      yaml:"buffer"      json:"buffer"`
	Consistency string `toml:"consistency" yaml:"consistency" json:"consistency"`
	LogLevel    string `toml:"log_level"   yaml:"log_level"   json:"log_level"`
}

type PipeRuntime struct {
	State     string `toml:"state"      yaml:"state"      json:"state"`
	LastError string `toml:"last_error" yaml:"last_error" json:"last_error"`
}

type PipeVars map[string]any

type PluginSet map[string]Plugin

type Plugin map[string]any

func (p Plugin) Id() uint64 {
	if rawId, ok := p[KeyPluginId]; ok {
		if id, ok := rawId.(uint64); ok {
			return id
		}
	}

	id := rand.Uint64()
	p[KeyPluginId] = id
	return id
}

func (p Plugin) Alias() string {
	aliasRaw, ok := p[KeyAlias]
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
	reverseRaw, ok := p[KeyReverse]
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
	typeRaw, ok := p[KeyType]
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
	logLevelRaw, ok := p[KeyLogLevel]
	if !ok {
		return ""
	}
	logLevel, ok := logLevelRaw.(string)
	if !ok {
		return ""
	}
	return logLevel
}

func (p Plugin) Lookup() string {
	lookupRaw, ok := p[KeyLookup]
	if !ok {
		return ""
	}
	lookup, ok := lookupRaw.(string)
	if !ok {
		return ""
	}
	return lookup
}

func (p Plugin) Parser() Plugin {
	parserRaw, ok := p[KeyParser]
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
	serializerRaw, ok := p[KeySerializer]
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
	filtersRaw, ok := p[KeyFilters]
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
	compressorNameRaw, ok := p[KeyCompressor]
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
	decompressorNameRaw, ok := p[KeyDecompressor]
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

func SetPipelineDefaults(settings *PipeSettings) {
	if settings.Lines <= 0 {
		settings.Lines = 1
	}

	if settings.Buffer < 0 {
		settings.Buffer = 0
	}

	if settings.Consistency == "" {
		settings.Consistency = "soft"
	}
}

type Marshalable interface {
	PipeSettings | Pipeline | []*Pipeline | PipeVars | []PluginSet
}

func UnmarshalPipeline[T Marshalable](data []byte, dest *T, format string) error {
	switch d := any(dest).(type) {
	case *PipeSettings:
		if err := UnmarshalFormat(data, d, format); err != nil {
			return err
		}
		SetPipelineDefaults(d)
		return nil
	case *Pipeline:
		if err := UnmarshalFormat(data, d, format); err != nil {
			return err
		}
		SetPipelineDefaults(&d.Settings)
		return nil
	case *[]*Pipeline:
		if err := UnmarshalFormat(data, d, format); err != nil {
			return err
		}
		for _, p := range *d {
			SetPipelineDefaults(&p.Settings)
		}
		return nil
	case *PipeVars:
		return UnmarshalFormat(data, d, format)
	case *[]PluginSet:
		return UnmarshalFormat(data, d, format)
	default:
		return fmt.Errorf("unmarshal pipeline: unsupported type: %T", dest)
	}
}

func MarshalPipeline[T Marshalable](src T, format string) ([]byte, error) {
	switch format {
	case ".toml":
		return toml.Marshal(src)
	case ".yaml", ".yml":
		return yaml.Marshal(src)
	case ".json":
		return json.Marshal(src)
	default:
		return nil, fmt.Errorf("unknown format: %v", format)
	}
}

func UnmarshalFormat(data []byte, dest any, format string) error {
	switch format {
	case ".toml":
		return toml.Unmarshal(data, dest)
	case ".yaml", ".yml":
		return yaml.Unmarshal(data, dest)
	case ".json":
		return json.Unmarshal(data, dest,
			json.WithUnmarshalers(json.UnmarshalFromFunc(jsonUnmarshalNumberStrict)))
	default:
		return fmt.Errorf("unknown format: %v", format)
	}
}

func jsonUnmarshalNumberStrict(dec *jsontext.Decoder, val *any) error {
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
