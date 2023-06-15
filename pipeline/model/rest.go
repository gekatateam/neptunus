package model

import "encoding/json"

type ErrResponse struct {
	Error string `json:"error"`
}

func ErrToJson(msg string) []byte {
	s, _ := json.Marshal(ErrResponse{Error: msg})
	return s
}

type OkResponse struct {
	Status string `json:"status"`
	Error  string `json:"last_error,omitempty"`
}

func OkToJson(msg string, err error) []byte {
	le := ""
	if err != nil {
		le = err.Error()
	}
	s, _ := json.Marshal(OkResponse{Status: msg, Error: le})
	return s
}
