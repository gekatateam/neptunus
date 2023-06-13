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
	"github.com/gekatateam/neptunus/pipeline/api"
)

var _ pipeline.Service = &restGateway{}

type restGateway struct {
	addr string
	c    *http.Client
	t    time.Duration
	ctx  context.Context
}

func Rest(addr string) *restGateway {
	return &restGateway{
		addr: addr,
		c: &http.Client{
			Timeout: 10 * time.Second,
		},
		t:   10 * time.Second,
		ctx: context.Background(),
	}
}

func (g *restGateway) Start(id string) error {
	ctx, cancel := context.WithTimeout(g.ctx, g.t)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("%v/api/v1/pipeline/%v/start", g.addr, id), nil)
	if err != nil {
		return err
	}

	res, err := g.c.Do(req)
	if err != nil {
		return err
	}

	defer res.Body.Close()
	switch res.StatusCode {
	case http.StatusOK:
		return nil
	case http.StatusNotFound:
		return &pipeline.NotFoundError{Err: unpackApiError(res.Body)}
	case http.StatusConflict:
		return &pipeline.ConflictError{Err: unpackApiError(res.Body)}
	default:
		return unpackApiError(res.Body)
	}
}

func (g *restGateway) Stop(id string) error {
	ctx, cancel := context.WithTimeout(g.ctx, g.t)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("%v/api/v1/pipelines/%v/stop", g.addr, id), nil)
	if err != nil {
		return err
	}

	res, err := g.c.Do(req)
	if err != nil {
		return err
	}

	defer res.Body.Close()
	switch res.StatusCode {
	case http.StatusOK:
		return nil
	case http.StatusNotFound:
		return &pipeline.NotFoundError{Err: unpackApiError(res.Body)}
	case http.StatusConflict:
		return &pipeline.ConflictError{Err: unpackApiError(res.Body)}
	default:
		return unpackApiError(res.Body)
	}
}

func (g *restGateway) State(id string) (string, error, error) {
	ctx, cancel := context.WithTimeout(g.ctx, g.t)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%v/api/v1/pipelines/%v/state", g.addr, id), nil)
	if err != nil {
		return "", nil, err
	}

	res, err := g.c.Do(req)
	if err != nil {
		return "", nil, err
	}

	defer res.Body.Close()
	switch res.StatusCode {
	case http.StatusOK:
		rawBody, _ := io.ReadAll(res.Body)
		structBody := &api.OkResponse{}
		json.Unmarshal(rawBody, structBody)
		if len(structBody.Error) > 0 {
			return structBody.Status, errors.New(structBody.Error), nil
		}
		return structBody.Status, nil, nil
	case http.StatusNotFound:
		return "", nil, &pipeline.NotFoundError{Err: unpackApiError(res.Body)}
	default:
		return "", nil, unpackApiError(res.Body)
	}
}

func (g *restGateway) List() ([]*config.Pipeline, error) {
	var pipes = []*config.Pipeline{}

	ctx, cancel := context.WithTimeout(g.ctx, g.t)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%v/api/v1/pipelines", g.addr), nil)
	if err != nil {
		return pipes, err
	}

	res, err := g.c.Do(req)
	if err != nil {
		return pipes, err
	}

	defer res.Body.Close()
	switch res.StatusCode {
	case http.StatusOK:
		rawBody, _ := io.ReadAll(res.Body)
		json.Unmarshal(rawBody, &pipes)
		return pipes, nil
	default:
		return pipes, unpackApiError(res.Body)
	}
}

func (g *restGateway) Get(id string) (*config.Pipeline, error) {
	var pipe = &config.Pipeline{}

	ctx, cancel := context.WithTimeout(g.ctx, g.t)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%v/api/v1/pipelines/%v", g.addr, id), nil)
	if err != nil {
		return pipe, err
	}

	res, err := g.c.Do(req)
	if err != nil {
		return pipe, err
	}

	defer res.Body.Close()
	switch res.StatusCode {
	case http.StatusOK:
		rawBody, _ := io.ReadAll(res.Body)
		json.Unmarshal(rawBody, pipe)
		return pipe, nil
	case http.StatusNotFound:
		return pipe, &pipeline.NotFoundError{Err: unpackApiError(res.Body)}
	default:
		return pipe, unpackApiError(res.Body)
	}
}

func (g *restGateway) Add(pipe *config.Pipeline) error {
	ctx, cancel := context.WithTimeout(g.ctx, g.t)
	defer cancel()

	pipeRaw, err := config.MarshalPipeline(pipe, "json")
	if err != nil {
		return nil
	}
	buf := bytes.NewBuffer(pipeRaw)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("%v/api/v1/pipelines", g.addr), buf)
	if err != nil {
		return err
	}

	res, err := g.c.Do(req)
	if err != nil {
		return err
	}

	defer res.Body.Close()
	switch res.StatusCode {
	case http.StatusOK:
		return nil
	case http.StatusConflict:
		return &pipeline.ConflictError{Err: unpackApiError(res.Body)}
	case http.StatusBadRequest:
		return &pipeline.ValidationError{Err: unpackApiError(res.Body)}
	default:
		return unpackApiError(res.Body)
	}
}

func (g *restGateway) Update(pipe *config.Pipeline) error {
	ctx, cancel := context.WithTimeout(g.ctx, g.t)
	defer cancel()

	pipeRaw, err := config.MarshalPipeline(pipe, "json")
	if err != nil {
		return nil
	}
	buf := bytes.NewBuffer(pipeRaw)

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, fmt.Sprintf("%v/api/v1/pipelines/%v", g.addr, pipe.Settings.Id), buf)
	if err != nil {
		return err
	}

	res, err := g.c.Do(req)
	if err != nil {
		return err
	}

	defer res.Body.Close()
	switch res.StatusCode {
	case http.StatusOK:
		return nil
	case http.StatusNotFound:
		return &pipeline.NotFoundError{Err: unpackApiError(res.Body)}
	case http.StatusConflict:
		return &pipeline.ConflictError{Err: unpackApiError(res.Body)}
	case http.StatusBadRequest:
		return &pipeline.ValidationError{Err: unpackApiError(res.Body)}
	default:
		return unpackApiError(res.Body)
	}
}

func (g *restGateway) Delete(id string) error {
	ctx, cancel := context.WithTimeout(g.ctx, g.t)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, fmt.Sprintf("%v/api/v1/pipelines/%v", g.addr, id), nil)
	if err != nil {
		return err
	}

	res, err := g.c.Do(req)
	if err != nil {
		return err
	}

	defer res.Body.Close()
	switch res.StatusCode {
	case http.StatusOK:
		return nil
	case http.StatusNotFound:
		return &pipeline.NotFoundError{Err: unpackApiError(res.Body)}
	case http.StatusConflict:
		return &pipeline.ConflictError{Err: unpackApiError(res.Body)}
	default:
		return unpackApiError(res.Body)
	}
}

func unpackApiError(resBody io.ReadCloser) error {
	rawBody, _ := io.ReadAll(resBody)
	structBody := &api.ErrResponse{}
	json.Unmarshal(rawBody, structBody)
	return errors.New(structBody.Error)
}
