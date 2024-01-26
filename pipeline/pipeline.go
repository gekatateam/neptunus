package pipeline

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"sync"

	"github.com/gekatateam/neptunus/config"
	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/metrics"
	"github.com/gekatateam/neptunus/pkg/mapstructure"
	"github.com/gekatateam/neptunus/plugins"

	"github.com/gekatateam/neptunus/plugins/core/broadcast"
	"github.com/gekatateam/neptunus/plugins/core/fusion"

	_ "github.com/gekatateam/neptunus/plugins/filters"
	_ "github.com/gekatateam/neptunus/plugins/inputs"
	_ "github.com/gekatateam/neptunus/plugins/outputs"
	_ "github.com/gekatateam/neptunus/plugins/parsers"
	_ "github.com/gekatateam/neptunus/plugins/processors"
	_ "github.com/gekatateam/neptunus/plugins/serializers"
)

type state string

const (
	StateCreated  state = "created"
	StateStarting state = "starting"
	StateStopping state = "stopping"
	StateRunning  state = "running"
	StateStopped  state = "stopped"
)

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

type unit interface {
	Run()
}

// pipeline run a set of plugins
type Pipeline struct {
	config *config.Pipeline
	log    *slog.Logger

	state   state
	lastErr error
	outs    []outputSet
	procs   [][]procSet
	ins     []inputSet
}

func New(config *config.Pipeline, log *slog.Logger) *Pipeline {
	return &Pipeline{
		config: config,
		log:    log,
		state:  StateCreated,
		outs:   make([]outputSet, 0, len(config.Outputs)),
		procs:  make([][]procSet, 0, config.Settings.Lines),
		ins:    make([]inputSet, 0, len(config.Inputs)),
	}
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

func (p *Pipeline) Close() error {
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
	return nil
}

func (p *Pipeline) Test() error {
	var err error
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

func (p *Pipeline) Build() error {
	if err := p.configureInputs(); err != nil {
		p.lastErr = err
		return err
	}
	p.log.Debug("inputs confiruration has no errors")

	if err := p.configureProcessors(); err != nil {
		p.lastErr = err
		return err
	}
	p.log.Debug("processors confiruration has no errors")

	if err := p.configureOutputs(); err != nil {
		p.lastErr = err
		return err
	}
	p.log.Debug("outputs confiruration has no errors")

	p.lastErr = nil
	return nil
}

func (p *Pipeline) Run(ctx context.Context) {
	p.log.Info("starting pipeline")
	p.state = StateStarting
	wg := &sync.WaitGroup{}

	p.log.Info("starting inputs")
	var inputsStopChannels = make([]chan struct{}, 0, len(p.ins))
	var inputsOutChannels = make([]<-chan *core.Event, 0, len(p.ins))
	for i, input := range p.ins {
		inputsStopChannels = append(inputsStopChannels, make(chan struct{}))
		inputUnit, outCh := core.NewDirectInputSoftUnit(input.i, input.f, inputsStopChannels[i], p.config.Settings.Buffer)
		inputsOutChannels = append(inputsOutChannels, outCh)
		wg.Add(1)
		go func(u unit) {
			u.Run()
			wg.Done()
		}(inputUnit)
	}

	p.log.Info("starting inputs-to-processors fusionner")
	inFusionUnit, outCh := core.NewDirectFusionSoftUnit(fusion.New(&core.BaseCore{
		Alias:    "fusion::inputs",
		Plugin:   "fusion",
		Pipeline: p.config.Settings.Id,
		Log: p.log.With(slog.Group("output",
			"plugin", "fusion",
			"name", "fusion::inputs",
		)),
		Obs: metrics.ObserveCoreSummary,
	}), inputsOutChannels, p.config.Settings.Buffer)
	wg.Add(1)
	go func(u unit) {
		u.Run()
		wg.Done()
	}(inFusionUnit)

	if len(p.procs) > 0 {
		p.log.Info(fmt.Sprintf("starting processors, scaling to %v parallel lines", p.config.Settings.Lines))
		var procsOutChannels = make([]<-chan *core.Event, 0, p.config.Settings.Lines)
		for i := 0; i < p.config.Settings.Lines; i++ {
			procInput := outCh
			for _, processor := range p.procs[i] {
				processorUnit, procOut := core.NewDirectProcessorSoftUnit(processor.p, processor.f, procInput, p.config.Settings.Buffer)
				wg.Add(1)
				go func(u unit) {
					u.Run()
					wg.Done()
				}(processorUnit)
				procInput = procOut
			}
			procsOutChannels = append(procsOutChannels, procInput)
			p.log.Info(fmt.Sprintf("line %v started", i))
		}

		p.log.Info("starting processors-to-broadcast fusionner")
		outFusionUnit, fusionOutCh := core.NewDirectFusionSoftUnit(fusion.New(&core.BaseCore{
			Alias:    "fusion::processors",
			Plugin:   "fusion",
			Pipeline: p.config.Settings.Id,
			Log: p.log.With(slog.Group("output",
				"plugin", "fusion",
				"name", "fusion::processors",
			)),
			Obs: metrics.ObserveCoreSummary,
		}), procsOutChannels, p.config.Settings.Buffer)
		outCh = fusionOutCh
		wg.Add(1)
		go func(u unit) {
			u.Run()
			wg.Done()
		}(outFusionUnit)
	}

	p.log.Info("starting broadcaster")
	bcastUnit, bcastChs := core.NewDirectBroadcastSoftUnit(broadcast.New(&core.BaseCore{
		Alias:    "broadcast::processors",
		Plugin:   "fusion",
		Pipeline: p.config.Settings.Id,
		Log: p.log.With(slog.Group("output",
			"plugin", "fusion",
			"name", "broadcast::processors",
		)),
		Obs: metrics.ObserveCoreSummary,
	}), outCh, len(p.outs), p.config.Settings.Buffer)
	wg.Add(1)
	go func(u unit) {
		u.Run()
		wg.Done()
	}(bcastUnit)

	p.log.Info("starting outputs")
	for i, output := range p.outs {
		outputUnit := core.NewDirectOutputSoftUnit(output.o, output.f, bcastChs[i], p.config.Settings.Buffer)
		wg.Add(1)
		go func(u unit) {
			u.Run()
			wg.Done()
		}(outputUnit)
	}

	p.log.Info("pipeline started")
	p.state = StateRunning

	<-ctx.Done()
	p.log.Info("stop signal received, stopping pipeline")
	p.state = StateStopping
	for _, stop := range inputsStopChannels {
		stop <- struct{}{}
	}
	wg.Wait()

	p.log.Info("pipeline stopped")
	p.state = StateStopped
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

			baseField := reflect.ValueOf(output.Self()).Elem().FieldByName("BaseOutput")
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

			if err := mapstructure.Decode(outputCfg, output.Self()); err != nil {
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

				processorCfg["::line"] = i

				baseField := reflect.ValueOf(processor.Self()).Elem().FieldByName("BaseProcessor")
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

				if err := mapstructure.Decode(processorCfg, processor.Self()); err != nil {
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
		return errors.New("at leats one input required")
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

			baseField := reflect.ValueOf(input.Self()).Elem().FieldByName("BaseInput")
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

			if err := mapstructure.Decode(inputCfg, input.Self()); err != nil {
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
			alias = fmt.Sprintf("%v::%v", filterCfg.Alias(), parentName)
		}

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

		baseField := reflect.ValueOf(filter.Self()).Elem().FieldByName("BaseFilter")
		if baseField.IsValid() && baseField.CanSet() {
			baseField.Set(reflect.ValueOf(&core.BaseFilter{
				Alias:    alias,
				Plugin:   plugin,
				Pipeline: p.config.Settings.Id,
				Log: p.log.With(slog.Group("filter",
					"plugin", plugin,
					"name", alias,
				)),
				Obs: metrics.ObserveFilterSummary,
			}))
		} else {
			return nil, fmt.Errorf("%v filter plugin does not contains BaseInput", plugin)
		}

		if err := mapstructure.Decode(filterCfg, filter.Self()); err != nil {
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
		alias = fmt.Sprintf("%v::%v", parserCfg.Alias(), parentName)
	}

	if idNeedy, ok := parser.(core.SetId); ok {
		idNeedy.SetId(parserCfg.Id())
	}

	baseField := reflect.ValueOf(parser.Self()).Elem().FieldByName("BaseParser")
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

	if err := mapstructure.Decode(parserCfg, parser.Self()); err != nil {
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
		alias = fmt.Sprintf("%v::%v", serCfg.Alias(), parentName)
	}

	if idNeedy, ok := serializer.(core.SetId); ok {
		idNeedy.SetId(serCfg.Id())
	}

	baseField := reflect.ValueOf(serializer.Self()).Elem().FieldByName("BaseSerializer")
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

	if err := mapstructure.Decode(serCfg, serializer.Self()); err != nil {
		return nil, fmt.Errorf("%v serializer configuration mapping error: %v", plugin, err.Error())
	}

	if err := serializer.Init(); err != nil {
		return nil, fmt.Errorf("%v serializer initialization error: %v", plugin, err.Error())
	}

	return serializer, nil
}
