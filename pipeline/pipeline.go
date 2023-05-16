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
	id     string
	config *config.Pipeline
	log    logger.Logger
	scale  int

	outs  []outputSet
	procs []procSet
	ins   []inputSet
}

func New(id string, scaleProcs int, config *config.Pipeline, log logger.Logger) *Pipeline {
	return &Pipeline{
		id:     id,
		config: config,
		log:    log,
		scale:  scaleProcs,
		outs:   make([]outputSet, 0),
		procs:  make([]procSet, 0),
		ins:    make([]inputSet, 0),
	}
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

func (p *Pipeline) Test() error {
	var err error
	if err = p.configureInputs(); err != nil {
		p.log.Errorf("inputs confiruration test failed: %v", err.Error())
		err = errors.New("pipeline test failed")
		goto PIPELINE_TESTED
	}
	p.log.Info("inputs confiruration has no errors")

	if err = p.configureProcessors(); err != nil {
		p.log.Errorf("processors confiruration test failed: %v", err.Error())
		err = errors.New("pipeline test failed")
		goto PIPELINE_TESTED
	}
	p.log.Info("processors confiruration has no errors")

	if err = p.configureOutputs(); err != nil {
		p.log.Errorf("outputs confiruration test failed: %v", err.Error())
		err = errors.New("pipeline test failed")
		goto PIPELINE_TESTED
	}
	p.log.Info("outputs confiruration has no errors")
	p.log.Info("pipeline tested successfully")

	PIPELINE_TESTED:
	return err
}

// !### DEPRECATED ###
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
		unit, stopCh := core.NewInputSoftUnit(input.o, inputsCh[i])
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

func (p *Pipeline) Run2(ctx context.Context) {
	p.log.Info("starting pipeline")
	wg := &sync.WaitGroup{}

	p.log.Info("starting inputs")
	var inputsStopChannels = make([]chan struct{}, 0, len(p.ins)) // <- pre-ready channels
	var inputsOutChannels = make([]<-chan *core.Event, 0, len(p.ins))
	for i, input := range p.ins {
		inputsStopChannels = append(inputsStopChannels, make(chan struct{}))
		unit, outCh := core.NewDirectInputSoftUnit(input.o, input.f, inputsStopChannels[i])
		inputsOutChannels = append(inputsOutChannels, outCh)
		wg.Add(1)
		go func() {
			unit.Run()
			wg.Done()
		}()
	}

	wg.Add(1)
	go func() {
		p.log.Info("starting stop-watcher")
		<-ctx.Done()
		p.log.Info("stop signal received, stopping pipeline")
		for _, stop := range inputsStopChannels {
			stop <- struct{}{}
		}
		wg.Done()
	}()

	p.log.Info("starting inputs-to-processors fusionner")
	fusionUnit, outCh := core.NewDirectFusionSoftUnit(inputsOutChannels...)
	wg.Add(1)
	go func() {
		fusionUnit.Run()
		wg.Done()
	}()

	if len(p.procs) > 0 {
		p.log.Infof("starting processors, scaling to %v parallel lines", p.scale)
		var procsOutChannels = make([]<-chan *core.Event, 0, p.scale)
		for i := 0; i < p.scale; i++ {
			procInput := outCh
			for _, processor := range p.procs {
				unit, procOut := core.NewDirectProcessorSoftUnit(processor.p, processor.f, procInput)
				wg.Add(1)
				go func() {
					unit.Run()
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
		unit := core.NewDirectOutputSoftUnit(output.o, output.f, bcastChs[i])
		wg.Add(1)
		go func() {
			unit.Run()
			wg.Done()
		}()
	}

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
