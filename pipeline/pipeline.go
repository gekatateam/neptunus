package pipeline

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/gekatateam/pipeline/config"
	"github.com/gekatateam/pipeline/core"
	"github.com/gekatateam/pipeline/logger"
	"github.com/gekatateam/pipeline/logger/logrus"
	"github.com/gekatateam/pipeline/plugins"

	_ "github.com/gekatateam/pipeline/plugins/filters"
	_ "github.com/gekatateam/pipeline/plugins/inputs"
	_ "github.com/gekatateam/pipeline/plugins/outputs"
	_ "github.com/gekatateam/pipeline/plugins/processors"
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
	o core.Input
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
	log    logger.Logger

	state state
	outs  []outputSet
	procs []procSet
	ins   []inputSet
}

func New(config *config.Pipeline, log logger.Logger) *Pipeline {
	return &Pipeline{
		config: config,
		log:    log,
		state:  StateCreated,
		outs:   make([]outputSet, 0, len(config.Outputs)),
		procs:  make([]procSet, 0, len(config.Processors)),
		ins:    make([]inputSet, 0, len(config.Inputs)),
	}
}

func (p *Pipeline) State() state {
	return p.state
}

func (p *Pipeline) Config() *config.Pipeline {
	return p.config
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
		return err
	}
	p.log.Debug("inputs confiruration has no errors")

	if err := p.configureProcessors(); err != nil {
		return err
	}
	p.log.Debug("processors confiruration has no errors")

	if err := p.configureOutputs(); err != nil {
		return err
	}
	p.log.Debug("outputs confiruration has no errors")

	return nil
}

func (p *Pipeline) Run(ctx context.Context) {
	p.log.Info("starting pipeline")
	p.state = StateStarting
	wg := &sync.WaitGroup{}

	p.log.Info("starting inputs")
	var inputsStopChannels = make([]chan struct{}, 0, len(p.ins)) // <- pre-ready channels
	var inputsOutChannels = make([]<-chan *core.Event, 0, len(p.ins))
	for i, input := range p.ins {
		inputsStopChannels = append(inputsStopChannels, make(chan struct{}))
		inputUnit, outCh := core.NewDirectInputSoftUnit(input.o, input.f, inputsStopChannels[i])
		inputsOutChannels = append(inputsOutChannels, outCh)
		wg.Add(1)
		go func() {
			inputUnit.Run()
			wg.Done()
		}()
	}

	p.log.Info("starting inputs-to-processors fusionner")
	fusionUnit, outCh := core.NewDirectFusionSoftUnit(inputsOutChannels...)
	wg.Add(1)
	go func() {
		fusionUnit.Run()
		wg.Done()
	}()

	if len(p.procs) > 0 {
		p.log.Infof("starting processors, scaling to %v parallel lines", p.config.Settings.Lines)
		var procsOutChannels = make([]<-chan *core.Event, 0, p.config.Settings.Lines)
		for i := 0; i < p.config.Settings.Lines; i++ {
			procInput := outCh
			for _, processor := range p.procs {
				processorUnit, procOut := core.NewDirectProcessorSoftUnit(processor.p, processor.f, procInput)
				wg.Add(1)
				go func() {
					processorUnit.Run()
					wg.Done()
				}()
				procInput = procOut
			}
			procsOutChannels = append(procsOutChannels, procInput)
		}

		p.log.Info("starting processors-to-broadcast fusionner")
		fusionUnit, outCh = core.NewDirectFusionSoftUnit(procsOutChannels...)
		wg.Add(1)
		go func() {
			fusionUnit.Run()
			wg.Done()
		}()
	}

	p.log.Info("starting broadcaster")
	bcastUnit, bcastChs := core.NewDirectBroadcastSoftUnit(outCh, len(p.outs))
	wg.Add(1)
	go func() {
		bcastUnit.Run()
		wg.Done()
	}()

	p.log.Info("starting outputs")
	for i, output := range p.outs {
		outputUnit := core.NewDirectOutputSoftUnit(output.o, output.f, bcastChs[i])
		wg.Add(1)
		go func() {
			outputUnit.Run()
			wg.Done()
		}()
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

			var alias = fmt.Sprintf("%v-%v", plugin, index)
			if len(outputCfg.Alias()) > 0 {
				alias = outputCfg.Alias()
			}

			output, err := outputFunc(outputCfg, alias, logrus.NewLogger(map[string]any{
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
	for index, processors := range p.config.Processors {
		for plugin, processorCfg := range processors {
			processorFunc, ok := plugins.GetProcessor(plugin)
			if !ok {
				return fmt.Errorf("unknown processor plugin in pipeline configuration: %v", plugin)
			}

			var alias = fmt.Sprintf("%v-%v", plugin, index)
			if len(processorCfg.Alias()) > 0 {
				alias = processorCfg.Alias()
			}

			processor, err := processorFunc(processorCfg, alias, logrus.NewLogger(map[string]any{
				"pipeline":  p.config.Settings.Id,
				"processor": plugin,
				"name":      alias,
			}))
			if err != nil {
				return fmt.Errorf("%v processor configuration error: %v", plugin, err.Error())
			}

			filters, err := p.configureFilters(processorCfg.Filters(), alias)
			if err != nil {
				return fmt.Errorf("%v output filters configuration error: %v", plugin, err.Error())
			}

			p.procs = append(p.procs, procSet{processor, filters})
		}
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

			var alias = fmt.Sprintf("%v-%v", plugin, index)
			if len(inputCfg.Alias()) > 0 {
				alias = inputCfg.Alias()
			}

			input, err := inputFunc(inputCfg, alias, logrus.NewLogger(map[string]any{
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

		var alias = fmt.Sprintf("%v-%v", parentName, plugin)
		if len(filterCfg.Alias()) > 0 {
			alias = filterCfg.Alias()
		}

		filter, err := filterFunc(filterCfg, alias, logrus.NewLogger(map[string]any{
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
