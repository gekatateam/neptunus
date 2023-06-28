package pipeline

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/gekatateam/neptunus/config"
	"github.com/gekatateam/neptunus/core"
	"github.com/gekatateam/neptunus/logger"
	"github.com/gekatateam/neptunus/logger/logrus"
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
	log    logger.Logger

	state   state
	lastErr error
	outs    []outputSet
	procs   [][]procSet
	ins     []inputSet
}

func New(config *config.Pipeline, log logger.Logger) *Pipeline {
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
		p.log.Errorf("inputs confiruration test failed: %v", err.Error())
		goto PIPELINE_TEST_FAILED
	}
	p.log.Info("inputs confiruration has no errors")

	if err = p.configureProcessors(); err != nil {
		p.log.Errorf("processors confiruration test failed: %v", err.Error())
		goto PIPELINE_TEST_FAILED
	}
	p.log.Info("processors confiruration has no errors")

	if err = p.configureOutputs(); err != nil {
		p.log.Errorf("outputs confiruration test failed: %v", err.Error())
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
		p.log.Tracef("input plugin %v out channel: %v", i, outCh)
		go func(u unit) {
			u.Run()
			wg.Done()
		}(inputUnit)
	}

	p.log.Info("starting inputs-to-processors fusionner")
	inFusionUnit, outCh := core.NewDirectFusionSoftUnit(fusion.New("fusion::inputs", p.config.Settings.Id), inputsOutChannels, p.config.Settings.Buffer)
	wg.Add(1)
	p.log.Tracef("inputs-to-processors fusion plugin in channels: %v", inputsOutChannels)
	p.log.Tracef("inputs-to-processors fusion plugin out channel: %v", outCh)
	go func(u unit) {
		u.Run()
		wg.Done()
	}(inFusionUnit)

	if len(p.procs) > 0 {
		p.log.Infof("starting processors, scaling to %v parallel lines", p.config.Settings.Lines)
		var procsOutChannels = make([]<-chan *core.Event, 0, p.config.Settings.Lines)
		for i := 0; i < p.config.Settings.Lines; i++ {
			procInput := outCh
			for j, processor := range p.procs[i] {
				processorUnit, procOut := core.NewDirectProcessorSoftUnit(processor.p, processor.f, procInput, p.config.Settings.Buffer)
				wg.Add(1)
				p.log.Tracef("line %v, processor %v plugin in channel: %v", i, j, procInput)
				p.log.Tracef("line %v, processor %v plugin out channel: %v", i, j, procOut)
				go func(u unit) {
					u.Run()
					wg.Done()
				}(processorUnit)
				procInput = procOut
			}
			procsOutChannels = append(procsOutChannels, procInput)
			p.log.Infof("line %v started", i)
		}

		p.log.Info("starting processors-to-broadcast fusionner")
		outFusionUnit, fusionOutCh := core.NewDirectFusionSoftUnit(fusion.New("fusion::processors", p.config.Settings.Id), procsOutChannels, p.config.Settings.Buffer)
		outCh = fusionOutCh
		wg.Add(1)
		p.log.Tracef("processors-to-broadcast fusion plugin in channels: %v", procsOutChannels)
		p.log.Tracef("processors-to-broadcast fusion plugin out channel: %v", outCh)
		go func(u unit) {
			u.Run()
			wg.Done()
		}(outFusionUnit)
	}

	p.log.Info("starting broadcaster")
	bcastUnit, bcastChs := core.NewDirectBroadcastSoftUnit(broadcast.New("broadcast::processors", p.config.Settings.Id), outCh, len(p.outs), p.config.Settings.Buffer)
	wg.Add(1)
	p.log.Tracef("broadcast plugin in channel: %v", outCh)
	p.log.Tracef("broadcast plugin out channels: %v", bcastChs)
	go func(u unit) {
		u.Run()
		wg.Done()
	}(bcastUnit)

	p.log.Info("starting outputs")
	for i, output := range p.outs {
		outputUnit := core.NewDirectOutputSoftUnit(output.o, output.f, bcastChs[i], p.config.Settings.Buffer)
		wg.Add(1)
		p.log.Tracef("output plugin %v in channel: %v", i, bcastChs[i])
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

			if serializerNeedy, ok := output.(core.SerializerNeedy); ok {
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

			if parserNeedy, ok := output.(core.ParserNeedy); ok {
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

			err := output.Init(outputCfg, alias, p.config.Settings.Id, logrus.NewLogger(map[string]any{
				"pipeline": p.config.Settings.Id,
				"output":   plugin,
				"name":     alias,
			}))
			if err != nil {
				return fmt.Errorf("%v output configuration error: %v", plugin, err.Error())
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

				if serializerNeedy, ok := processor.(core.SerializerNeedy); ok {
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

				if parserNeedy, ok := processor.(core.ParserNeedy); ok {
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

				processorCfg["::line"] = i
				err := processor.Init(processorCfg, alias, p.config.Settings.Id, logrus.NewLogger(map[string]any{
					"pipeline":  p.config.Settings.Id,
					"processor": plugin,
					"name":      alias,
				}))
				delete(processorCfg, "::line")
				if err != nil {
					return fmt.Errorf("%v processor configuration error: %v", plugin, err.Error())
				}

				filters, err := p.configureFilters(processorCfg.Filters(), alias)
				if err != nil {
					return fmt.Errorf("%v output filters configuration error: %v", plugin, err.Error())
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

			if serializerNeedy, ok := input.(core.SerializerNeedy); ok {
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

			if parserNeedy, ok := input.(core.ParserNeedy); ok {
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

			err := input.Init(inputCfg, alias, p.config.Settings.Id, logrus.NewLogger(map[string]any{
				"pipeline": p.config.Settings.Id,
				"input":    plugin,
				"name":     alias,
			}))
			if err != nil {
				return fmt.Errorf("%v input configuration error: %v", plugin, err.Error())
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

		if serializerNeedy, ok := filter.(core.SerializerNeedy); ok {
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

		if parserNeedy, ok := filter.(core.ParserNeedy); ok {
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

		err := filter.Init(filterCfg, alias, p.config.Settings.Id, logrus.NewLogger(map[string]any{
			"pipeline": p.config.Settings.Id,
			"filter":   plugin,
			"name":     alias,
		}))
		if err != nil {
			return nil, fmt.Errorf("%v filter configuration error: %v", plugin, err.Error())
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

	err := parser.Init(parserCfg, alias, p.config.Settings.Id, logrus.NewLogger(map[string]any{
		"pipeline": p.config.Settings.Id,
		"parser":   plugin,
		"name":     alias,
	}))
	if err != nil {
		return nil, fmt.Errorf("%v parser configuration error: %v", plugin, err.Error())
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

	err := serializer.Init(serCfg, alias, p.config.Settings.Id, logrus.NewLogger(map[string]any{
		"pipeline":   p.config.Settings.Id,
		"serializer": plugin,
		"name":       alias,
	}))
	if err != nil {
		return nil, fmt.Errorf("%v serializer configuration error: %v", plugin, err.Error())
	}

	return serializer, nil
}
