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
)

type unit interface {
	Run()
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
	id     string
	config *config.Pipeline
	log    logger.Logger

	outs  []outputSet
	procs []procSet
	ins   []core.Input
}

func NewPipeline(id string, config *config.Pipeline, log logger.Logger) *Pipeline {
	return &Pipeline{
		id:     id,
		config: config,
		log:    log,
		outs:   make([]outputSet, 0),
		procs:  make([]procSet, 0),
		ins:    make([]core.Input, 0),
	}
}

func (p *Pipeline) Build() error {
	if err := p.configureOutputs(); err != nil {
		return err
	}

	p.log.Debug("outputs confiruration has no errors")

	if err := p.configureProcessors(); err != nil {
		return err
	}

	p.log.Debug("processors confiruration has no errors")

	if err := p.configureInputs(); err != nil {
		return err
	}

	p.log.Debug("inputs confiruration has no errors")

	return nil
}

func (p *Pipeline) Run(ctx context.Context) {
	p.log.Info("starting pipeline")
	wg := &sync.WaitGroup{}

	p.log.Info("starting outputs")
	var outputsChannels = make([]chan<- *core.Event, 0, len(p.outs))
	for _, output := range p.outs {
		unit, outCh := core.NewOutputSoftUnit(output.o, output.f)
		outputsChannels = append(outputsChannels, outCh)
		wg.Add(1)
		go func() {
			unit.Run()
			wg.Done()
		}()
	}

	p.log.Info("starting broadcaster")
	bcastUnit, bcastCh := core.NewBroadcastSoftUnit(outputsChannels...)
	wg.Add(1)
	go func() {
		bcastUnit.Run()
		wg.Done()
	}()

	p.log.Info("starting processors")
	for _, processor := range p.procs {
		unit, procCh := core.NewProcessorSoftUnit(processor.p, processor.f, bcastCh)
		wg.Add(1)
		go func() {
			unit.Run()
			wg.Done()
		}()
		bcastCh = procCh
	}

	p.log.Info("starting fusionner")
	fusionUnit, inputsCh := core.NewFusionSoftUnit(bcastCh, len(p.ins))
	wg.Add(1)
	go func() {
		fusionUnit.Run()
		wg.Done()
	}()

	p.log.Info("starting inputs")
	var inputsStopChannels = make([]chan<- struct{}, 0, len(p.ins))
	for i, input := range p.ins {
		unit, stopCh := core.NewInputSoftUnit(input, inputsCh[i])
		inputsStopChannels = append(inputsStopChannels, stopCh)
		wg.Add(1)
		go func() {
			unit.Run()
			wg.Done()
		}()
	}

	wg.Add(1)
	go func() {
		<-ctx.Done()
		p.log.Info("stop signal received, stopping pipeline")
		for _, stop := range inputsStopChannels {
			stop <- struct{}{}
		}
		wg.Done()
	}()
	p.log.Info("pipeline started")
	wg.Wait()
	p.log.Info("pipeline stopped")
}

func (p *Pipeline) configureOutputs() error {
	if len(p.config.Outputs) == 0 {
		return errors.New("at leats one output required")
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
				"pipeline": p.id,
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
				"pipeline":  p.id,
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
				"pipeline": p.id,
				"input":    plugin,
				"name":     alias,
			}))
			if err != nil {
				return fmt.Errorf("%v input configuration error: %v", plugin, err.Error())
			}

			p.ins = append(p.ins, input)
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
			"pipeline": p.id,
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
