package core

import (
	"encoding/json"
	"fmt"
	"io"
	"reflect"
)

type Errors []error

func (me Errors) MarshalJSON() ([]byte, error) {
	data := []byte{'['}
	for i, err := range me {
		if i != 0 {
			data = append(data, ',')
		}

		j, err := json.Marshal(err.Error())
		if err != nil {
			return nil, err
		}

		data = append(data, j...)
	}
	data = append(data, ']')

	return data, nil
}

func (me Errors) Slice() []string {
	s := []string{}
	for _, err := range me {
		s = append(s, err.Error())
	}
	return s
}

// FullCloseError wraps the error returned by Close method of a plugin
// with plugin type, name and alias information.
// It panics if the provided plugin is not a Keykeeper, Input, Output, Processor or Filter.
func FullCloseError(plugin io.Closer) error {
	err := plugin.Close()
	if err == nil {
		return nil
	}

	switch p := plugin.(type) {
	case Keykeeper:
		b := reflect.ValueOf(p).Elem().FieldByName(KindKeykeeper).Interface().(*BaseKeykeeper)
		return fmt.Errorf("keykeeper %s %s: %w", b.Plugin, b.Alias, err)
	case Lookup:
		b := reflect.ValueOf(p).Elem().FieldByName(KindLookup).Interface().(*BaseLookup)
		return fmt.Errorf("lookup %s %s: %w", b.Plugin, b.Alias, err)
	case Input:
		b := reflect.ValueOf(p).Elem().FieldByName(KindInput).Interface().(*BaseInput)
		return fmt.Errorf("input %s %s: %w", b.Plugin, b.Alias, err)
	case Output:
		b := reflect.ValueOf(p).Elem().FieldByName(KindOutput).Interface().(*BaseOutput)
		return fmt.Errorf("output %s %s: %w", b.Plugin, b.Alias, err)
	case Processor: // filter also may hide here
		b := reflect.ValueOf(p).Elem().FieldByName(KindProcessor)
		if b.IsValid() {
			base := b.Interface().(*BaseProcessor)
			return fmt.Errorf("processor %s %s: %w", base.Plugin, base.Alias, err)
		}

		b = reflect.ValueOf(p).Elem().FieldByName(KindFilter)
		if b.IsValid() {
			base := b.Interface().(*BaseFilter)
			return fmt.Errorf("filter %s %s: %w", base.Plugin, base.Alias, err)
		}
	}
	panic(fmt.Errorf("unsupported plugin type: %T", plugin))
}
