package pipeline

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"regexp"
	"sync"

	"github.com/gekatateam/neptunus/config"
	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/core/unit"
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/pkg/mapstructure"
	"github.com/gekatateam/neptunus/plugins"

	"github.com/gekatateam/neptunus/plugins/core/broadcast"
	"github.com/gekatateam/neptunus/plugins/core/fusion"
	"github.com/gekatateam/neptunus/plugins/core/self"

	_ "github.com/gekatateam/neptunus/plugins/filters"
	_ "github.com/gekatateam/neptunus/plugins/inputs"
	_ "github.com/gekatateam/neptunus/plugins/keykeepers"
	_ "github.com/gekatateam/neptunus/plugins/outputs"
	_ "github.com/gekatateam/neptunus/plugins/parsers"
	_ "github.com/gekatateam/neptunus/plugins/processors"
	_ "github.com/gekatateam/neptunus/plugins/serializers"
)

type state string

var (
	StateCreated  state = "created"
	StateBuilding state = "building"
	StateStarting state = "starting"
	StateRunning  state = "running"
	StateStopping state = "stopping"
	StateStopped  state = "stopped"

	StateCode = map[state]int{
		StateCreated:  1,
		StateBuilding: 2,
		StateStarting: 3,
		StateRunning:  4,
		StateStopping: 5,
		StateStopped:  6,
	}
)

var keyConfigPattern = regexp.MustCompile(`^@{(.+):(.+)}$`)

// at this moment it is not possible to combine sets into a generic type
// like:
//
//	type pluginSet[P core.Input | core.Processor | core.Output] struct {
//		p P
//		f []core.Filter
//	}
//
// cause https://github.com/golang/go/issues/49054
type inputSet struct {
	i core.Input
	f []core.Filter
}

type outputSet struct {
	o core.Output
	f []core.Filter
}

type procSet struct {
	p core.Processor
	f []core.Filter
}

// pipeline run a set of plugins
type Pipeline struct {
	config *config.Pipeline
	log    *slog.Logger

	state   state
	lastErr error
	aliases map[string]struct{}

	keepers map[string]core.Keykeeper
	outs    []outputSet
	procs   [][]procSet
	ins     []inputSet

	chansStatsFuncs []metrics.ChanStatsFunc
}

func New(config *config.Pipeline, log *slog.Logger) *Pipeline {
	return &Pipeline{
		config:  config,
		log:     log,
		state:   StateCreated,
		aliases: make(map[string]struct{}),
		keepers: make(map[string]core.Keykeeper),
		outs:    make([]outputSet, 0, len(config.Outputs)),
		procs:   make([][]procSet, 0, config.Settings.Lines),
		ins:     make([]inputSet, 0, len(config.Inputs)),
	}
}

func (p *Pipeline) ChansStats() []metrics.ChanStats {
	chansStats := []metrics.ChanStats{}
	for _, f := range p.chansStatsFuncs {
		chansStats = append(chansStats, f())
	}

	return chansStats
}

func (p *Pipeline) State() state {
	return p.state
}

func (p *Pipeline) LastError() error {
	return p.lastErr
}

func (p *Pipeline) Config() *config.Pipeline {
	return p.config
}

// Pipeline Close() MUST be called only if pipeline build failed.
// After successfull Run(), each plugin will be closed dy it's unit.
// As usual, there is a few exclusions:
//   - each plugin becomes available for closing only after successful Init()
//   - as a consequence, plugins must check network connections, run init queries, etc.
//     inside Init(), and must free all resources, if Init() failed
func (p *Pipeline) Close() error {
	for _, k := range p.keepers {
		k.Close()
	}

	for _, set := range p.ins {
		set.i.Close()
		for _, f := range set.f {
			f.Close()
		}
	}

	for i := range p.procs {
		for _, set := range p.procs[i] {
			for _, f := range set.f {
				f.Close()
			}
			set.p.Close()
		}
	}

	for _, set := range p.outs {
		for _, f := range set.f {
			f.Close()
		}
		set.o.Close()
	}

	p.aliases = make(map[string]struct{})
	p.keepers = make(map[string]core.Keykeeper)
	p.outs = make([]outputSet, 0, len(p.config.Outputs))
	p.procs = make([][]procSet, 0, p.config.Settings.Lines)
	p.ins = make([]inputSet, 0, len(p.config.Inputs))

	return nil
}

func (p *Pipeline) Test() (err error) {
	if err = p.configureKeykeepers(); err != nil {
		p.log.Error("keykeepers confiruration test failed",
			"error", err.Error(),
		)
		goto PIPELINE_TEST_FAILED
	}
	p.log.Info("keykeepers confiruration has no errors")

	if err = p.configureInputs(); err != nil {
		p.log.Error("inputs confiruration test failed",
			"error", err.Error(),
		)
		goto PIPELINE_TEST_FAILED
	}
	p.log.Info("inputs confiruration has no errors")

	if err = p.configureProcessors(); err != nil {
		p.log.Error("inputs confiruration test failed",
			"error", err.Error(),
		)
		goto PIPELINE_TEST_FAILED
	}
	p.log.Info("processors confiruration has no errors")

	if err = p.configureOutputs(); err != nil {
		p.log.Error("inputs confiruration test failed",
			"error", err.Error(),
		)
		goto PIPELINE_TEST_FAILED
	}
	p.log.Info("outputs confiruration has no errors")

	p.log.Info("pipeline tested successfully")
	return nil

PIPELINE_TEST_FAILED:
	return errors.New("pipeline test failed")
}

func (p *Pipeline) Build() (err error) {
	p.state = StateBuilding
	defer func() {
		p.lastErr = err
		if p.lastErr != nil {
			p.state = StateStopped
		}
	}()

	if err = p.configureKeykeepers(); err != nil {
		return
	}
	p.log.Debug("keykeepers confiruration has no errors")

	if err = p.configureInputs(); err != nil {
		return
	}
	p.log.Debug("inputs confiruration has no errors")

	if err = p.configureProcessors(); err != nil {
		return
	}
	p.log.Debug("processors confiruration has no errors")

	if err = p.configureOutputs(); err != nil {
		return
	}
	p.log.Debug("outputs confiruration has no errors")

	return
}

func (p *Pipeline) Run(ctx context.Context) {
	p.state = StateStarting
	p.log.Info("starting pipeline")

	wg := &sync.WaitGroup{}

	p.log.Info("starting inputs")
	var inputsStopChannels = make([]chan struct{}, 0, len(p.ins))
	var inputsOutChannels = make([]<-chan *core.Event, 0, len(p.ins))
	for i, input := range p.ins {
		inputsStopChannels = append(inputsStopChannels, make(chan struct{}))
		inputUnit, outCh, chansStats := unit.NewInput(&p.config.Settings, p.log, input.i, input.f, inputsStopChannels[i], p.config.Settings.Buffer)
		p.chansStatsFuncs = append(p.chansStatsFuncs, chansStats...)
		inputsOutChannels = append(inputsOutChannels, outCh)
		wg.Add(1)
		go func() {
			inputUnit.Run()
			wg.Done()
		}()
	}

	p.log.Info("starting inputs-to-processors fusionner")
	inFusionUnit, outCh, chansStats := unit.NewFusion(&p.config.Settings, p.log, fusion.New(&core.BaseCore{
		Alias:    "fusion::inputs",
		Plugin:   "fusion",
		Pipeline: p.config.Settings.Id,
		Log: p.log.With(slog.Group("output",
			"plugin", "fusion",
			"name", "fusion::inputs",
		)),
		Obs: metrics.ObserveCoreSummary,
	}), inputsOutChannels, p.config.Settings.Buffer)
	p.chansStatsFuncs = append(p.chansStatsFuncs, chansStats...)
	wg.Add(1)
	go func() {
		inFusionUnit.Run()
		wg.Done()
	}()

	if len(p.procs) > 0 {
		p.log.Info(fmt.Sprintf("starting processors, scaling to %v parallel lines", p.config.Settings.Lines))
		var procsOutChannels = make([]<-chan *core.Event, 0, p.config.Settings.Lines)
		for i := 0; i < p.config.Settings.Lines; i++ {
			procInput := outCh
			for _, processor := range p.procs[i] {
				processorUnit, procOut, chansStats := unit.NewProcessor(&p.config.Settings, p.log, processor.p, processor.f, procInput, p.config.Settings.Buffer)
				p.chansStatsFuncs = append(p.chansStatsFuncs, chansStats...)
				wg.Add(1)
				go func() {
					processorUnit.Run()
					wg.Done()
				}()
				procInput = procOut
			}
			procsOutChannels = append(procsOutChannels, procInput)
			p.log.Info(fmt.Sprintf("line %v started", i))
		}

		p.log.Info("starting processors-to-broadcast fusionner")
		outFusionUnit, fusionOutCh, chansStats := unit.NewFusion(&p.config.Settings, p.log, fusion.New(&core.BaseCore{
			Alias:    "fusion::processors",
			Plugin:   "fusion",
			Pipeline: p.config.Settings.Id,
			Log: p.log.With(slog.Group("output",
				"plugin", "fusion",
				"name", "fusion::processors",
			)),
			Obs: metrics.ObserveCoreSummary,
		}), procsOutChannels, p.config.Settings.Buffer)
		p.chansStatsFuncs = append(p.chansStatsFuncs, chansStats...)
		outCh = fusionOutCh
		wg.Add(1)
		go func() {
			outFusionUnit.Run()
			wg.Done()
		}()
	}

	p.log.Info("starting broadcaster")
	bcastUnit, bcastChs, chansStats := unit.NewBroadcast(&p.config.Settings, p.log, broadcast.New(&core.BaseCore{
		Alias:    "broadcast::processors",
		Plugin:   "fusion",
		Pipeline: p.config.Settings.Id,
		Log: p.log.With(slog.Group("output",
			"plugin", "fusion",
			"name", "broadcast::processors",
		)),
		Obs: metrics.ObserveCoreSummary,
	}), outCh, len(p.outs), p.config.Settings.Buffer)
	p.chansStatsFuncs = append(p.chansStatsFuncs, chansStats...)
	wg.Add(1)
	go func() {
		bcastUnit.Run()
		wg.Done()
	}()

	p.log.Info("starting outputs")
	for i, output := range p.outs {
		outputUnit, chansStats := unit.NewOutput(&p.config.Settings, p.log, output.o, output.f, bcastChs[i], p.config.Settings.Buffer)
		p.chansStatsFuncs = append(p.chansStatsFuncs, chansStats...)
		wg.Add(1)
		go func() {
			outputUnit.Run()
			wg.Done()
		}()
	}

	p.state = StateRunning
	p.log.Info("pipeline started")

	<-ctx.Done()

	p.state = StateStopping
	p.log.Info("stop signal received, stopping pipeline")
	for _, stop := range inputsStopChannels {
		stop <- struct{}{}
	}
	wg.Wait()

	for _, k := range p.keepers {
		k.Close()
	}

	p.log.Info("pipeline stopped")
	p.state = StateStopped
}

func (p *Pipeline) configureKeykeepers() error {
	for index, keykeepers := range p.config.Keykeepers {
		for plugin, keeperCfg := range keykeepers {
			keeperFunc, ok := plugins.GetKeykeeper(plugin)
			if !ok {
				return fmt.Errorf("unknown keykeeper plugin in pipeline configuration: %v", plugin)
			}
			keykeeper := keeperFunc()

			var alias = fmt.Sprintf("keykeeper:%v:%v", plugin, index)
			if len(keeperCfg.Alias()) > 0 {
				alias = keeperCfg.Alias()
			}

			if _, ok := p.aliases[alias]; ok {
				return fmt.Errorf("duplicate alias detected in pipeline configuration: %v, from %v keykeeper", alias, plugin)
			}
			p.aliases[alias] = struct{}{}

			if self, ok := keykeeper.(*self.Self); ok {
				self.SetConfig(p.config)
			}

			baseField := reflect.ValueOf(keykeeper).Elem().FieldByName(core.KindKeykeeper)
			if baseField.IsValid() && baseField.CanSet() {
				baseField.Set(reflect.ValueOf(&core.BaseKeykeeper{
					Alias:    alias,
					Plugin:   plugin,
					Pipeline: p.config.Settings.Id,
					Log: p.log.With(slog.Group("keykeeper",
						"plugin", plugin,
						"name", alias,
					)),
				}))
			} else {
				return fmt.Errorf("%v keykeeper plugin does not contains BaseKeykeeper", plugin)
			}

			if err := mapstructure.Decode(keeperCfg, keykeeper, p.decodeHook()); err != nil {
				return fmt.Errorf("%v keykeeper configuration mapping error: %v", plugin, err.Error())
			}

			if err := keykeeper.Init(); err != nil {
				return fmt.Errorf("%v keykeeper initialization error: %v", plugin, err.Error())
			}

			p.keepers[alias] = keykeeper
		}
	}
	return nil
}

func (p *Pipeline) configureOutputs() error {
	if len(p.config.Outputs) == 0 {
		return errors.New("at least one output required")
	}

	for index, outputs := range p.config.Outputs {
		for plugin, outputCfg := range outputs {
			outputFunc, ok := plugins.GetOutput(plugin)
			if !ok {
				return fmt.Errorf("unknown output plugin in pipeline configuration: %v", plugin)
			}
			output := outputFunc()

			var alias = fmt.Sprintf("output:%v:%v", plugin, index)
			if len(outputCfg.Alias()) > 0 {
				alias = outputCfg.Alias()
			}

			if _, ok := p.aliases[alias]; ok {
				return fmt.Errorf("duplicate alias detected in pipeline configuration: %v, from %v output", alias, plugin)
			}
			p.aliases[alias] = struct{}{}

			if serializerNeedy, ok := output.(core.SetSerializer); ok {
				cfgSerializer := outputCfg.Serializer()
				if cfgSerializer == nil {
					return fmt.Errorf("%v output requires serializer, but no serializer configuration provided", plugin)
				}

				serializer, err := p.configureSerializer(cfgSerializer, alias)
				if err != nil {
					return fmt.Errorf("%v output serializer configuration error: %v", plugin, err.Error())
				}
				serializerNeedy.SetSerializer(serializer)
			}

			if parserNeedy, ok := output.(core.SetParser); ok {
				cfgParser := outputCfg.Parser()
				if cfgParser == nil {
					return fmt.Errorf("%v output requires parser, but no parser configuration provided", plugin)
				}

				parser, err := p.configureParser(cfgParser, alias)
				if err != nil {
					return fmt.Errorf("%v output parser configuration error: %v", plugin, err.Error())
				}
				parserNeedy.SetParser(parser)
			}

			if idNeedy, ok := output.(core.SetId); ok {
				idNeedy.SetId(outputCfg.Id())
			}

			baseField := reflect.ValueOf(output).Elem().FieldByName(core.KindOutput)
			if baseField.IsValid() && baseField.CanSet() {
				baseField.Set(reflect.ValueOf(&core.BaseOutput{
					Alias:    alias,
					Plugin:   plugin,
					Pipeline: p.config.Settings.Id,
					Log: p.log.With(slog.Group("output",
						"plugin", plugin,
						"name", alias,
					)),
					Obs: metrics.ObserveOutputSummary,
				}))
			} else {
				return fmt.Errorf("%v output plugin does not contains BaseOutput", plugin)
			}

			if err := mapstructure.Decode(outputCfg, output, p.decodeHook()); err != nil {
				return fmt.Errorf("%v output configuration mapping error: %v", plugin, err.Error())
			}

			if err := output.Init(); err != nil {
				return fmt.Errorf("%v output initialization error: %v", plugin, err.Error())
			}

			filters, err := p.configureFilters(outputCfg.Filters(), alias)
			if err != nil {
				return fmt.Errorf("%v output filters configuration error: %v", plugin, err.Error())
			}

			p.outs = append(p.outs, outputSet{output, filters})
		}
	}
	return nil
}

func (p *Pipeline) configureProcessors() error {
	// because Go does not provide safe way to copy objects
	// we create so much duplicate of processors sets
	// as lines configured
	for i := 0; i < p.config.Settings.Lines; i++ {
		var sets = make([]procSet, 0, len(p.config.Processors))
		for index, processors := range p.config.Processors {
			for plugin, processorCfg := range processors {
				processorFunc, ok := plugins.GetProcessor(plugin)
				if !ok {
					return fmt.Errorf("unknown processor plugin in pipeline configuration: %v", plugin)
				}
				processor := processorFunc()

				var alias = fmt.Sprintf("processor:%v:%v:%v", plugin, index, i)
				if len(processorCfg.Alias()) > 0 {
					alias = fmt.Sprintf("%v:%v", processorCfg.Alias(), i)
				}

				if _, ok := p.aliases[alias]; ok {
					return fmt.Errorf("duplicate alias detected in pipeline configuration: %v, from %v processor", alias, plugin)
				}
				p.aliases[alias] = struct{}{}

				if serializerNeedy, ok := processor.(core.SetSerializer); ok {
					cfgSerializer := processorCfg.Serializer()
					if cfgSerializer == nil {
						return fmt.Errorf("%v processor requires serializer, but no serializer configuration provided", plugin)
					}

					serializer, err := p.configureSerializer(cfgSerializer, alias)
					if err != nil {
						return fmt.Errorf("%v processor serializer configuration error: %v", plugin, err.Error())
					}
					serializerNeedy.SetSerializer(serializer)
				}

				if parserNeedy, ok := processor.(core.SetParser); ok {
					cfgParser := processorCfg.Parser()
					if cfgParser == nil {
						return fmt.Errorf("%v processor requires parser, but no parser configuration provided", plugin)
					}

					parser, err := p.configureParser(cfgParser, alias)
					if err != nil {
						return fmt.Errorf("%v processor parser configuration error: %v", plugin, err.Error())
					}
					parserNeedy.SetParser(parser)
				}

				if idNeedy, ok := processor.(core.SetId); ok {
					idNeedy.SetId(processorCfg.Id())
				}

				if lineNeedy, ok := processor.(core.SetLine); ok {
					lineNeedy.SetLine(i)
				}

				baseField := reflect.ValueOf(processor).Elem().FieldByName(core.KindProcessor)
				if baseField.IsValid() && baseField.CanSet() {
					baseField.Set(reflect.ValueOf(&core.BaseProcessor{
						Alias:    alias,
						Plugin:   plugin,
						Pipeline: p.config.Settings.Id,
						Log: p.log.With(slog.Group("processor",
							"plugin", plugin,
							"name", alias,
						)),
						Obs: metrics.ObserveProcessorSummary,
					}))
				} else {
					return fmt.Errorf("%v processor plugin does not contains BaseProcessor", plugin)
				}

				if err := mapstructure.Decode(processorCfg, processor, p.decodeHook()); err != nil {
					return fmt.Errorf("%v processor configuration mapping error: %v", plugin, err.Error())
				}

				if err := processor.Init(); err != nil {
					return fmt.Errorf("%v processor initialization error: %v", plugin, err.Error())
				}

				filters, err := p.configureFilters(processorCfg.Filters(), alias)
				if err != nil {
					return fmt.Errorf("%v processor filters configuration error: %v", plugin, err.Error())
				}

				sets = append(sets, procSet{processor, filters})
			}
		}
		p.procs = append(p.procs, sets)
	}
	return nil
}

func (p *Pipeline) configureInputs() error {
	if len(p.config.Inputs) == 0 {
		return errors.New("at least one input required")
	}

	for index, inputs := range p.config.Inputs {
		for plugin, inputCfg := range inputs {
			inputFunc, ok := plugins.GetInput(plugin)
			if !ok {
				return fmt.Errorf("unknown input plugin in pipeline configuration: %v", plugin)
			}
			input := inputFunc()

			var alias = fmt.Sprintf("input:%v:%v", plugin, index)
			if len(inputCfg.Alias()) > 0 {
				alias = inputCfg.Alias()
			}

			if _, ok := p.aliases[alias]; ok {
				return fmt.Errorf("duplicate alias detected in pipeline configuration: %v, from %v input", alias, plugin)
			}
			p.aliases[alias] = struct{}{}

			if serializerNeedy, ok := input.(core.SetSerializer); ok {
				cfgSerializer := inputCfg.Serializer()
				if cfgSerializer == nil {
					return fmt.Errorf("%v input requires serializer, but no serializer configuration provided", plugin)
				}

				serializer, err := p.configureSerializer(cfgSerializer, alias)
				if err != nil {
					return fmt.Errorf("%v input serializer configuration error: %v", plugin, err.Error())
				}
				serializerNeedy.SetSerializer(serializer)
			}

			if parserNeedy, ok := input.(core.SetParser); ok {
				cfgParser := inputCfg.Parser()
				if cfgParser == nil {
					return fmt.Errorf("%v input requires parser, but no parser configuration provided", plugin)
				}

				parser, err := p.configureParser(cfgParser, alias)
				if err != nil {
					return fmt.Errorf("%v input parser configuration error: %v", plugin, err.Error())
				}
				parserNeedy.SetParser(parser)
			}

			if idNeedy, ok := input.(core.SetId); ok {
				idNeedy.SetId(inputCfg.Id())
			}

			baseField := reflect.ValueOf(input).Elem().FieldByName(core.KindInput)
			if baseField.IsValid() && baseField.CanSet() {
				baseField.Set(reflect.ValueOf(&core.BaseInput{
					Alias:    alias,
					Plugin:   plugin,
					Pipeline: p.config.Settings.Id,
					Log: p.log.With(slog.Group("input",
						"plugin", plugin,
						"name", alias,
					)),
					Obs: metrics.ObserveInputSummary,
				}))
			} else {
				return fmt.Errorf("%v input plugin does not contains BaseInput", plugin)
			}

			if err := mapstructure.Decode(inputCfg, input, p.decodeHook()); err != nil {
				return fmt.Errorf("%v input configuration mapping error: %v", plugin, err.Error())
			}

			if err := input.Init(); err != nil {
				return fmt.Errorf("%v input initialization error: %v", plugin, err.Error())
			}

			filters, err := p.configureFilters(inputCfg.Filters(), alias)
			if err != nil {
				return fmt.Errorf("%v input filters configuration error: %v", plugin, err.Error())
			}

			p.ins = append(p.ins, inputSet{input, filters})
		}
	}
	return nil
}

func (p *Pipeline) configureFilters(filtersSet config.PluginSet, parentName string) ([]core.Filter, error) {
	var filters []core.Filter
	for plugin, filterCfg := range filtersSet {
		filterFunc, ok := plugins.GetFilter(plugin)
		if !ok {
			return nil, fmt.Errorf("unknown filter plugin in pipeline configuration: %v", plugin)
		}
		filter := filterFunc()

		var alias = fmt.Sprintf("filter:%v::%v", plugin, parentName)
		if len(filterCfg.Alias()) > 0 {
			alias = filterCfg.Alias()
		}

		if _, ok := p.aliases[alias]; ok {
			return nil, fmt.Errorf("duplicate alias detected in pipeline configuration: %v, from %v filter", alias, plugin)
		}
		p.aliases[alias] = struct{}{}

		if serializerNeedy, ok := filter.(core.SetSerializer); ok {
			cfgSerializer := filterCfg.Serializer()
			if cfgSerializer == nil {
				return nil, fmt.Errorf("%v filter requires serializer, but no serializer configuration provided", plugin)
			}

			serializer, err := p.configureSerializer(cfgSerializer, alias)
			if err != nil {
				return nil, fmt.Errorf("%v filter serializer configuration error: %v", plugin, err.Error())
			}
			serializerNeedy.SetSerializer(serializer)
		}

		if parserNeedy, ok := filter.(core.SetParser); ok {
			cfgParser := filterCfg.Parser()
			if cfgParser == nil {
				return nil, fmt.Errorf("%v filter requires parser, but no parser configuration provided", plugin)
			}

			parser, err := p.configureParser(cfgParser, alias)
			if err != nil {
				return nil, fmt.Errorf("%v filter parser configuration error: %v", plugin, err.Error())
			}
			parserNeedy.SetParser(parser)
		}

		if idNeedy, ok := filter.(core.SetId); ok {
			idNeedy.SetId(filterCfg.Id())
		}

		baseField := reflect.ValueOf(filter).Elem().FieldByName(core.KindFilter)
		if baseField.IsValid() && baseField.CanSet() {
			baseField.Set(reflect.ValueOf(&core.BaseFilter{
				Alias:    alias,
				Plugin:   plugin,
				Pipeline: p.config.Settings.Id,
				Reverse:  filterCfg.Reverse(),
				Log: p.log.With(slog.Group("filter",
					"plugin", plugin,
					"name", alias,
				)),
				Obs: metrics.ObserveFilterSummary,
			}))
		} else {
			return nil, fmt.Errorf("%v filter plugin does not contains BaseInput", plugin)
		}

		if err := mapstructure.Decode(filterCfg, filter, p.decodeHook()); err != nil {
			return nil, fmt.Errorf("%v filter configuration mapping error: %v", plugin, err.Error())
		}

		if err := filter.Init(); err != nil {
			return nil, fmt.Errorf("%v filter initialization error: %v", plugin, err.Error())
		}

		filters = append(filters, filter)
	}
	return filters, nil
}

func (p *Pipeline) configureParser(parserCfg config.Plugin, parentName string) (core.Parser, error) {
	plugin := parserCfg.Type()
	parserFunc, ok := plugins.GetParser(plugin)
	if !ok {
		return nil, fmt.Errorf("unknown parser plugin in pipeline configuration: %v", plugin)
	}
	parser := parserFunc()

	var alias = fmt.Sprintf("parser:%v::%v", plugin, parentName)
	if len(parserCfg.Alias()) > 0 {
		alias = parserCfg.Alias()
	}

	if _, ok := p.aliases[alias]; ok {
		return nil, fmt.Errorf("duplicate alias detected in pipeline configuration: %v, from %v parser", alias, plugin)
	}
	p.aliases[alias] = struct{}{}

	if idNeedy, ok := parser.(core.SetId); ok {
		idNeedy.SetId(parserCfg.Id())
	}

	baseField := reflect.ValueOf(parser).Elem().FieldByName(core.KindParser)
	if baseField.IsValid() && baseField.CanSet() {
		baseField.Set(reflect.ValueOf(&core.BaseParser{
			Alias:    alias,
			Plugin:   plugin,
			Pipeline: p.config.Settings.Id,
			Log: p.log.With(slog.Group("parser",
				"plugin", plugin,
				"name", alias,
			)),
			Obs: metrics.ObserveParserSummary,
		}))
	} else {
		return nil, fmt.Errorf("%v parser plugin does not contains BaseParser", plugin)
	}

	if err := mapstructure.Decode(parserCfg, parser, p.decodeHook()); err != nil {
		return nil, fmt.Errorf("%v parser configuration mapping error: %v", plugin, err.Error())
	}

	if err := parser.Init(); err != nil {
		return nil, fmt.Errorf("%v parser initialization error: %v", plugin, err.Error())
	}

	return parser, nil
}

func (p *Pipeline) configureSerializer(serCfg config.Plugin, parentName string) (core.Serializer, error) {
	plugin := serCfg.Type()
	serFunc, ok := plugins.GetSerializer(plugin)
	if !ok {
		return nil, fmt.Errorf("unknown serializer plugin in pipeline configuration: %v", plugin)
	}
	serializer := serFunc()

	var alias = fmt.Sprintf("serializer:%v::%v", plugin, parentName)
	if len(serCfg.Alias()) > 0 {
		alias = serCfg.Alias()
	}

	if _, ok := p.aliases[alias]; ok {
		return nil, fmt.Errorf("duplicate alias detected in pipeline configuration: %v, from %v serializer", alias, plugin)
	}
	p.aliases[alias] = struct{}{}

	if idNeedy, ok := serializer.(core.SetId); ok {
		idNeedy.SetId(serCfg.Id())
	}

	baseField := reflect.ValueOf(serializer).Elem().FieldByName(core.KindSerializer)
	if baseField.IsValid() && baseField.CanSet() {
		baseField.Set(reflect.ValueOf(&core.BaseSerializer{
			Alias:    alias,
			Plugin:   plugin,
			Pipeline: p.config.Settings.Id,
			Log: p.log.With(slog.Group("serializer",
				"plugin", plugin,
				"name", alias,
			)),
			Obs: metrics.ObserveSerializerSummary,
		}))
	} else {
		return nil, fmt.Errorf("%v serializer plugin does not contains BaseSerializer", plugin)
	}

	if err := mapstructure.Decode(serCfg, serializer, p.decodeHook()); err != nil {
		return nil, fmt.Errorf("%v serializer configuration mapping error: %v", plugin, err.Error())
	}

	if err := serializer.Init(); err != nil {
		return nil, fmt.Errorf("%v serializer initialization error: %v", plugin, err.Error())
	}

	return serializer, nil
}

func (p *Pipeline) decodeHook() func(f reflect.Type, _ reflect.Type, data any) (any, error) {
	return func(
		f reflect.Type,
		_ reflect.Type,
		data any) (any, error) {
		if f.Kind() != reflect.String {
			return data, nil
		}

		if match := keyConfigPattern.FindStringSubmatch(data.(string)); len(match) == 3 {
			k, ok := p.keepers[match[1]]
			if !ok {
				return nil, fmt.Errorf("keykeeper not specified in configuration or still not initialized: %v", match[1])
			}

			val, err := k.Get(match[2])
			if err != nil {
				return nil, fmt.Errorf("error reading key %v using %v keykeeper: %w", match[2], match[1], err)
			}

			return val, nil
		}

		return data, nil
	}
}
