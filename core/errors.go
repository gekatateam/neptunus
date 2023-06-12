package core

import "encoding/json"

type Errors []error

func (me Errors) MarshalJSON() ([]byte, error) {
	data := []byte("[")
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
