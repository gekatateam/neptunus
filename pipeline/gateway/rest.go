package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gekatateam/neptunus/config"
	"github.com/gekatateam/neptunus/pipeline"
	"github.com/gekatateam/neptunus/pipeline/model"
)

var _ pipeline.Service = (*restGateway)(nil)

type restGateway struct {
	addr string
	c    *http.Client
	t    time.Duration
}

func Rest(addr, path string, timeout time.Duration) *restGateway {
	return &restGateway{
		addr: fmt.Sprintf("%v/%v", addr, path),
		c: &http.Client{
			Timeout: timeout,
		},
		t: timeout,
	}
}

func (g *restGateway) Start(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), g.t)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("%v/%v/start", g.addr, id), nil)
	if err != nil {
		return &pipeline.IOError{Err: err}
	}

	res, err := g.c.Do(req)
	if err != nil {
		return &pipeline.IOError{Err: err}
	}

	defer res.Body.Close()
	switch res.StatusCode {
	case http.StatusOK:
		io.Copy(io.Discard, res.Body)
		return nil
	case http.StatusNotFound:
		return &pipeline.NotFoundError{Err: unpackApiError(res.Body)}
	case http.StatusConflict:
		return &pipeline.ConflictError{Err: unpackApiError(res.Body)}
	default:
		return &pipeline.IOError{Err: unpackApiError(res.Body)}
	}
}

func (g *restGateway) Stop(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), g.t)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("%v/%v/stop", g.addr, id), nil)
	if err != nil {
		return &pipeline.IOError{Err: err}
	}

	res, err := g.c.Do(req)
	if err != nil {
		return &pipeline.IOError{Err: err}
	}

	defer res.Body.Close()
	switch res.StatusCode {
	case http.StatusOK:
		io.Copy(io.Discard, res.Body)
		return nil
	case http.StatusNotFound:
		return &pipeline.NotFoundError{Err: unpackApiError(res.Body)}
	case http.StatusConflict:
		return &pipeline.ConflictError{Err: unpackApiError(res.Body)}
	default:
		return &pipeline.IOError{Err: unpackApiError(res.Body)}
	}
}

func (g *restGateway) State(id string) (string, error, error) {
	ctx, cancel := context.WithTimeout(context.Background(), g.t)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%v/%v/state", g.addr, id), nil)
	if err != nil {
		return "", nil, &pipeline.IOError{Err: err}
	}

	res, err := g.c.Do(req)
	if err != nil {
		return "", nil, &pipeline.IOError{Err: err}
	}

	defer res.Body.Close()
	switch res.StatusCode {
	case http.StatusOK:
		rawBody, err := io.ReadAll(res.Body)
		if err != nil {
			return "", nil, &pipeline.IOError{Err: err}
		}

		structBody := &model.OkResponse{}
		json.Unmarshal(rawBody, structBody)
		if len(structBody.Error) > 0 {
			return structBody.Status, errors.New(structBody.Error), nil
		}
		return structBody.Status, nil, nil
	case http.StatusNotFound:
		return "", nil, &pipeline.NotFoundError{Err: unpackApiError(res.Body)}
	default:
		return "", nil, &pipeline.IOError{Err: err}
	}
}

func (g *restGateway) List() ([]*config.Pipeline, error) {
	var pipes = []*config.Pipeline{}

	ctx, cancel := context.WithTimeout(context.Background(), g.t)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%v/", g.addr), nil)
	if err != nil {
		return nil, &pipeline.IOError{Err: err}
	}

	res, err := g.c.Do(req)
	if err != nil {
		return nil, &pipeline.IOError{Err: err}
	}

	defer res.Body.Close()
	switch res.StatusCode {
	case http.StatusOK:
		rawBody, err := io.ReadAll(res.Body)
		if err != nil {
			return nil, &pipeline.IOError{Err: err}
		}

		// io error here because unmarshalling fail means data couldn't be read properly
		if err := config.UnmarshalPipeline(rawBody, &pipes, ".json"); err != nil {
			return nil, &pipeline.IOError{Err: err}
		}
		return pipes, nil
	default:
		return pipes, &pipeline.IOError{Err: unpackApiError(res.Body)}
	}
}

func (g *restGateway) Get(id string) (*config.Pipeline, error) {
	var pipe = &config.Pipeline{}

	ctx, cancel := context.WithTimeout(context.Background(), g.t)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%v/%v", g.addr, id), nil)
	if err != nil {
		return nil, &pipeline.IOError{Err: err}
	}

	res, err := g.c.Do(req)
	if err != nil {
		return nil, &pipeline.IOError{Err: err}
	}

	defer res.Body.Close()
	switch res.StatusCode {
	case http.StatusOK:
		rawBody, err := io.ReadAll(res.Body)
		if err != nil {
			return pipe, &pipeline.IOError{Err: err}
		}

		// io error here because unmarshalling fail means data couldn't be read properly
		if err := config.UnmarshalPipeline(rawBody, pipe, ".json"); err != nil {
			return nil, &pipeline.IOError{Err: err}
		}
		return pipe, nil
	case http.StatusNotFound:
		return pipe, &pipeline.NotFoundError{Err: unpackApiError(res.Body)}
	default:
		return pipe, &pipeline.IOError{Err: unpackApiError(res.Body)}
	}
}

func (g *restGateway) Add(pipe *config.Pipeline) error {
	ctx, cancel := context.WithTimeout(context.Background(), g.t)
	defer cancel()

	pipeRaw, err := config.MarshalPipeline(*pipe, ".json")
	if err != nil {
		return &pipeline.ValidationError{Err: err}
	}
	buf := bytes.NewBuffer(pipeRaw)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("%v/", g.addr), buf)
	if err != nil {
		return &pipeline.IOError{Err: err}
	}

	res, err := g.c.Do(req)
	if err != nil {
		return &pipeline.IOError{Err: err}
	}

	defer res.Body.Close()
	switch res.StatusCode {
	case http.StatusCreated:
		io.Copy(io.Discard, res.Body)
		return nil
	case http.StatusConflict:
		return &pipeline.ConflictError{Err: unpackApiError(res.Body)}
	case http.StatusBadRequest:
		return &pipeline.ValidationError{Err: unpackApiError(res.Body)}
	default:
		return &pipeline.IOError{Err: unpackApiError(res.Body)}
	}
}

func (g *restGateway) Update(pipe *config.Pipeline) error {
	ctx, cancel := context.WithTimeout(context.Background(), g.t)
	defer cancel()

	pipeRaw, err := config.MarshalPipeline(*pipe, ".json")
	if err != nil {
		return &pipeline.ValidationError{Err: err}
	}
	buf := bytes.NewBuffer(pipeRaw)

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, fmt.Sprintf("%v/%v", g.addr, pipe.Settings.Id), buf)
	if err != nil {
		return &pipeline.IOError{Err: err}
	}

	res, err := g.c.Do(req)
	if err != nil {
		return &pipeline.IOError{Err: err}
	}

	defer res.Body.Close()
	switch res.StatusCode {
	case http.StatusOK:
		io.Copy(io.Discard, res.Body)
		return nil
	case http.StatusNotFound:
		return &pipeline.NotFoundError{Err: unpackApiError(res.Body)}
	case http.StatusConflict:
		return &pipeline.ConflictError{Err: unpackApiError(res.Body)}
	case http.StatusBadRequest:
		return &pipeline.ValidationError{Err: unpackApiError(res.Body)}
	default:
		return &pipeline.IOError{Err: unpackApiError(res.Body)}
	}
}

func (g *restGateway) Delete(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), g.t)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, fmt.Sprintf("%v/%v", g.addr, id), nil)
	if err != nil {
		return &pipeline.IOError{Err: err}
	}

	res, err := g.c.Do(req)
	if err != nil {
		return &pipeline.IOError{Err: err}
	}

	defer res.Body.Close()
	switch res.StatusCode {
	case http.StatusOK:
		io.Copy(io.Discard, res.Body)
		return nil
	case http.StatusNotFound:
		return &pipeline.NotFoundError{Err: unpackApiError(res.Body)}
	case http.StatusConflict:
		return &pipeline.ConflictError{Err: unpackApiError(res.Body)}
	default:
		return &pipeline.IOError{Err: unpackApiError(res.Body)}
	}
}

func unpackApiError(resBody io.ReadCloser) error {
	rawBody, err := io.ReadAll(resBody)
	if err != nil {
		return err
	}

	structBody := &model.ErrResponse{}
	if err := json.Unmarshal(rawBody, structBody); err != nil {
		return err
	}

	return errors.New(structBody.Error)
}
