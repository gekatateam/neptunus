package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/gekatateam/pipeline/config"
	"github.com/gekatateam/pipeline/logger"
	"github.com/gekatateam/pipeline/pipeline"
)

type restApi struct {
	s   pipeline.Service
	log logger.Logger
}

func Rest(s pipeline.Service, log logger.Logger) *restApi {
	return &restApi{s: s, log: log}
}

// POST /pipelines/{id}/start
func (a *restApi) Start() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")

		err := a.s.Start(id)
		switch {
		case err == nil:
			w.WriteHeader(http.StatusOK)
			w.Write(OkToJson("starting"))
		case errors.As(err, &pipeline.ValidationErr):
			w.WriteHeader(http.StatusInternalServerError)
			w.Write(ErrToJson(err.Error()))
		case errors.As(err, &pipeline.NotFoundErr):
			w.WriteHeader(http.StatusNotFound)
			w.Write(ErrToJson(err.Error()))
		case errors.As(err, &pipeline.ConflictErr):
			w.WriteHeader(http.StatusConflict)
			w.Write(ErrToJson(err.Error()))
		case errors.As(err, &pipeline.IOErr):
			w.WriteHeader(http.StatusInternalServerError)
			w.Write(ErrToJson(err.Error()))
		default:
			a.log.Errorf("internal error at %v: %v", r.URL.Path, err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			w.Write(ErrToJson(err.Error()))
		}
	})
}

// POST /pipelines/{id}/stop
func (a *restApi) Stop() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")

		err := a.s.Stop(id)
		switch {
		case err == nil:
			w.WriteHeader(http.StatusOK)
			w.Write(OkToJson("stopping"))
		case errors.As(err, &pipeline.ConflictErr):
			w.WriteHeader(http.StatusConflict)
			w.Write(ErrToJson(err.Error()))
		case errors.As(err, &pipeline.NotFoundErr):
			w.WriteHeader(http.StatusNotFound)
			w.Write(ErrToJson(err.Error()))
		default:
			a.log.Errorf("internal error at %v: %v", r.URL.Path, err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			w.Write(ErrToJson(err.Error()))
		}
	})
}

// GET /pipelines/{id}/state
func (a *restApi) State() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")

		status, err := a.s.State(id)
		switch {
		case err == nil:
			w.WriteHeader(http.StatusOK)
			w.Write(OkToJson(status))
		case errors.As(err, &pipeline.NotFoundErr):
			w.WriteHeader(http.StatusNotFound)
			w.Write(ErrToJson(err.Error()))
		default:
			a.log.Errorf("internal error at %v: %v", r.URL.Path, err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			w.Write(ErrToJson(err.Error()))
		}
	})
}

// GET /pipelines/
func (a *restApi) List() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pipes, err := a.s.List()
		switch {
		case err == nil:
			data, _ := json.Marshal(pipes)
			w.WriteHeader(http.StatusOK)
			w.Write(data)
		case errors.As(err, &pipeline.ValidationErr):
			w.WriteHeader(http.StatusInternalServerError)
			w.Write(ErrToJson(err.Error()))
		case errors.As(err, &pipeline.IOErr):
			w.WriteHeader(http.StatusInternalServerError)
			w.Write(ErrToJson(err.Error()))
		default:
			a.log.Errorf("internal error at %v: %v", r.URL.Path, err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			w.Write(ErrToJson(err.Error()))
		}
	})
}

// GET /pipelines/{id}
func (a *restApi) Get() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")

		pipe, err := a.s.Get(id)
		switch {
		case err == nil:
			data, _ := json.Marshal(pipe)
			w.WriteHeader(http.StatusOK)
			w.Write(data)
		case errors.As(err, &pipeline.NotFoundErr):
			w.WriteHeader(http.StatusNotFound)
			w.Write(ErrToJson(err.Error()))
		case errors.As(err, &pipeline.ValidationErr):
			w.WriteHeader(http.StatusInternalServerError)
			w.Write(ErrToJson(err.Error()))
		case errors.As(err, &pipeline.IOErr):
			w.WriteHeader(http.StatusInternalServerError)
			w.Write(ErrToJson(err.Error()))
		default:
			a.log.Errorf("internal error at %v: %v", r.URL.Path, err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			w.Write(ErrToJson(err.Error()))
		}
	})
}

// POST /pipelines/{id}
func (a *restApi) Add() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, _ := io.ReadAll(r.Body)

		pipe, err := config.UnmarshalPipeline(data, "json")
		switch {
		case err == nil:
		default:
			w.WriteHeader(http.StatusBadRequest)
			w.Write(ErrToJson(err.Error()))
		}

		err = a.s.Add(pipe)
		switch {
		case err == nil:
			data, _ := json.Marshal(pipe)
			w.WriteHeader(http.StatusCreated)
			w.Write(data)
		case errors.As(err, &pipeline.ConflictErr):
			w.WriteHeader(http.StatusConflict)
			w.Write(ErrToJson(err.Error()))
		case errors.As(err, &pipeline.ValidationErr):
			w.WriteHeader(http.StatusBadRequest)
			w.Write(ErrToJson(err.Error()))
		case errors.As(err, &pipeline.IOErr):
			w.WriteHeader(http.StatusInternalServerError)
			w.Write(ErrToJson(err.Error()))
		default:
			a.log.Errorf("internal error at %v: %v", r.URL.Path, err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			w.Write(ErrToJson(err.Error()))
		}
	})
}

// PUT /pipelines/{id}
func (a *restApi) Update() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, _ := io.ReadAll(r.Body)

		pipe, err := config.UnmarshalPipeline(data, "json")
		switch {
		case err == nil:
		default:
			w.WriteHeader(http.StatusBadRequest)
			w.Write(ErrToJson(err.Error()))
		}

		err = a.s.Add(pipe)
		switch {
		case err == nil:
			data, _ := json.Marshal(pipe)
			w.WriteHeader(http.StatusOK)
			w.Write(data)
		case errors.As(err, &pipeline.NotFoundErr):
			w.WriteHeader(http.StatusNotFound)
			w.Write(ErrToJson(err.Error()))
		case errors.As(err, &pipeline.ConflictErr):
			w.WriteHeader(http.StatusConflict)
			w.Write(ErrToJson(err.Error()))
		case errors.As(err, &pipeline.ValidationErr):
			w.WriteHeader(http.StatusBadRequest)
			w.Write(ErrToJson(err.Error()))
		case errors.As(err, &pipeline.IOErr):
			w.WriteHeader(http.StatusInternalServerError)
			w.Write(ErrToJson(err.Error()))
		default:
			a.log.Errorf("internal error at %v: %v", r.URL.Path, err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			w.Write(ErrToJson(err.Error()))
		}
	})
}

// DELETE /pipelines/{id}
func (a *restApi) Delete() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")

		err := a.s.Delete(id)
		switch {
		case err == nil:
			w.WriteHeader(http.StatusOK)
			w.Write(OkToJson("deleted"))
		case errors.As(err, &pipeline.NotFoundErr):
			w.WriteHeader(http.StatusNotFound)
			w.Write(ErrToJson(err.Error()))
		case errors.As(err, &pipeline.ConflictErr):
			w.WriteHeader(http.StatusConflict)
			w.Write(ErrToJson(err.Error()))
		case errors.As(err, &pipeline.IOErr):
			w.WriteHeader(http.StatusInternalServerError)
			w.Write(ErrToJson(err.Error()))
		default:
			a.log.Errorf("internal error at %v: %v", r.URL.Path, err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			w.Write(ErrToJson(err.Error()))
		}
	})
}

type ErrResponse struct {
	Error string `json:"error"`
}

func ErrToJson(msg string) []byte {
	s, _ := json.Marshal(ErrResponse{Error: msg}); 
	return s
}

type OkResponse struct {
	Status string `json:"status"`
}

func OkToJson(msg string) []byte {
	s, _ := json.Marshal(OkResponse{Status: msg})
	return s
}
