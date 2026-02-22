package api

import (
	"io"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/gekatateam/neptunus/config"
	"github.com/gekatateam/neptunus/pipeline"
	"github.com/gekatateam/neptunus/pipeline/model"
	xerrors "github.com/gekatateam/neptunus/pkg/errors"
)

type restApi struct {
	s   pipeline.Service
	log *slog.Logger
}

func Rest(s pipeline.Service, log *slog.Logger) *restApi {
	return &restApi{s: s, log: log}
}

func (a *restApi) Router() *chi.Mux {
	router := chi.NewRouter()

	router.Get("/", a.List().ServeHTTP)
	router.Get("/{id}", a.Get().ServeHTTP)
	router.Get("/{id}/state", a.State().ServeHTTP)
	router.Post("/", a.Add().ServeHTTP)
	router.Put("/{id}", a.Update().ServeHTTP)
	router.Delete("/{id}", a.Delete().ServeHTTP)
	router.Post("/{id}/start", a.Start().ServeHTTP)
	router.Post("/{id}/stop", a.Stop().ServeHTTP)

	return router
}

// POST /pipelines/{id}/start
func (a *restApi) Start() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")

		err := a.s.Start(id)
		switch {
		case err == nil:
			w.WriteHeader(http.StatusOK)
			w.Write(model.OkToJson("starting", nil))
		case xerrors.AsType[*pipeline.ValidationError](err):
			w.WriteHeader(http.StatusInternalServerError)
			w.Write(model.ErrToJson(err.Error()))
		case xerrors.AsType[*pipeline.NotFoundError](err):
			w.WriteHeader(http.StatusNotFound)
			w.Write(model.ErrToJson(err.Error()))
		case xerrors.AsType[*pipeline.ConflictError](err):
			w.WriteHeader(http.StatusConflict)
			w.Write(model.ErrToJson(err.Error()))
		case xerrors.AsType[*pipeline.IOError](err):
			w.WriteHeader(http.StatusInternalServerError)
			w.Write(model.ErrToJson(err.Error()))
		default:
			a.log.Error("internal error occurred",
				"url", r.URL.Path,
				"error", err,
			)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write(model.ErrToJson(err.Error()))
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
			w.Write(model.OkToJson("stopping", nil))
		case xerrors.AsType[*pipeline.ConflictError](err):
			w.WriteHeader(http.StatusConflict)
			w.Write(model.ErrToJson(err.Error()))
		case xerrors.AsType[*pipeline.NotFoundError](err):
			w.WriteHeader(http.StatusNotFound)
			w.Write(model.ErrToJson(err.Error()))
		default:
			a.log.Error("internal error occurred",
				"url", r.URL.Path,
				"error", err,
			)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write(model.ErrToJson(err.Error()))
		}
	})
}

// GET /pipelines/{id}/state
func (a *restApi) State() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")

		status, lastErr, err := a.s.State(id)
		switch {
		case err == nil:
			w.WriteHeader(http.StatusOK)
			w.Write(model.OkToJson(status, lastErr))
		case xerrors.AsType[*pipeline.NotFoundError](err):
			w.WriteHeader(http.StatusNotFound)
			w.Write(model.ErrToJson(err.Error()))
		default:
			a.log.Error("internal error occurred",
				"url", r.URL.Path,
				"error", err,
			)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write(model.ErrToJson(err.Error()))
		}
	})
}

// GET /pipelines/
func (a *restApi) List() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pipes, err := a.s.List()
		switch {
		case err == nil:
		case xerrors.AsType[*pipeline.ValidationError](err):
			w.WriteHeader(http.StatusInternalServerError)
			w.Write(model.ErrToJson(err.Error()))
			return
		case xerrors.AsType[*pipeline.IOError](err):
			w.WriteHeader(http.StatusInternalServerError)
			w.Write(model.ErrToJson(err.Error()))
			return
		default:
			a.log.Error("internal error occurred",
				"url", r.URL.Path,
				"error", err,
			)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write(model.ErrToJson(err.Error()))
			return
		}

		for _, p := range pipes {
			state, lastErr, err := a.s.State(p.Settings.Id)
			if err != nil {
				p.Runtime = &config.PipeRuntime{
					LastError: err.Error(),
				}
				continue
			}

			p.Runtime = &config.PipeRuntime{
				State:     state,
				LastError: errAsString(lastErr),
			}
		}

		data, _ := config.MarshalPipeline(pipes, ".json")
		w.WriteHeader(http.StatusOK)
		w.Write(data)
	})
}

// GET /pipelines/{id}
func (a *restApi) Get() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")

		pipe, err := a.s.Get(id)
		switch {
		case err == nil:
			data, _ := config.MarshalPipeline(*pipe, ".json")
			w.WriteHeader(http.StatusOK)
			w.Write(data)
		case xerrors.AsType[*pipeline.NotFoundError](err):
			w.WriteHeader(http.StatusNotFound)
			w.Write(model.ErrToJson(err.Error()))
		case xerrors.AsType[*pipeline.ValidationError](err):
			w.WriteHeader(http.StatusInternalServerError)
			w.Write(model.ErrToJson(err.Error()))
		case xerrors.AsType[*pipeline.IOError](err):
			w.WriteHeader(http.StatusInternalServerError)
			w.Write(model.ErrToJson(err.Error()))
		default:
			a.log.Error("internal error occurred",
				"url", r.URL.Path,
				"error", err,
			)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write(model.ErrToJson(err.Error()))
		}
	})
}

// POST /pipelines/
func (a *restApi) Add() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, _ := io.ReadAll(r.Body)

		pipe := new(config.Pipeline)
		if err := config.UnmarshalPipeline(data, pipe, ".json"); err != nil {
			a.log.Error("internal error occurred",
				"url", r.URL.Path,
				"error", err,
			)
			w.WriteHeader(http.StatusBadRequest)
			w.Write(model.ErrToJson(err.Error()))
			return
		}

		err := a.s.Add(pipe)
		switch {
		case err == nil:
			w.WriteHeader(http.StatusCreated)
			w.Write(model.OkToJson("added", nil))
		case xerrors.AsType[*pipeline.ConflictError](err):
			w.WriteHeader(http.StatusConflict)
			w.Write(model.ErrToJson(err.Error()))
		case xerrors.AsType[*pipeline.ValidationError](err):
			w.WriteHeader(http.StatusBadRequest)
			w.Write(model.ErrToJson(err.Error()))
		case xerrors.AsType[*pipeline.IOError](err):
			w.WriteHeader(http.StatusInternalServerError)
			w.Write(model.ErrToJson(err.Error()))
		default:
			a.log.Error("internal error occurred",
				"url", r.URL.Path,
				"error", err,
			)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write(model.ErrToJson(err.Error()))
		}
	})
}

// POST /pipelines/{id}
func (a *restApi) Update() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, _ := io.ReadAll(r.Body)

		pipe := new(config.Pipeline)
		if err := config.UnmarshalPipeline(data, pipe, ".json"); err != nil {
			a.log.Error("internal error occurred",
				"url", r.URL.Path,
				"error", err,
			)
			w.WriteHeader(http.StatusBadRequest)
			w.Write(model.ErrToJson(err.Error()))
			return
		}

		pipe.Settings.Id = chi.URLParam(r, "id")
		err := a.s.Update(pipe)
		switch {
		case err == nil:
			w.WriteHeader(http.StatusOK)
			w.Write(model.OkToJson("updated", nil))
		case xerrors.AsType[*pipeline.NotFoundError](err):
			w.WriteHeader(http.StatusNotFound)
			w.Write(model.ErrToJson(err.Error()))
		case xerrors.AsType[*pipeline.ConflictError](err):
			w.WriteHeader(http.StatusConflict)
			w.Write(model.ErrToJson(err.Error()))
		case xerrors.AsType[*pipeline.ValidationError](err):
			w.WriteHeader(http.StatusBadRequest)
			w.Write(model.ErrToJson(err.Error()))
		case xerrors.AsType[*pipeline.IOError](err):
			w.WriteHeader(http.StatusInternalServerError)
			w.Write(model.ErrToJson(err.Error()))
		default:
			a.log.Error("internal error occurred",
				"url", r.URL.Path,
				"error", err,
			)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write(model.ErrToJson(err.Error()))
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
			w.Write(model.OkToJson("deleted", nil))
		case xerrors.AsType[*pipeline.NotFoundError](err):
			w.WriteHeader(http.StatusNotFound)
			w.Write(model.ErrToJson(err.Error()))
		case xerrors.AsType[*pipeline.ConflictError](err):
			w.WriteHeader(http.StatusConflict)
			w.Write(model.ErrToJson(err.Error()))
		case xerrors.AsType[*pipeline.IOError](err):
			w.WriteHeader(http.StatusInternalServerError)
			w.Write(model.ErrToJson(err.Error()))
		default:
			a.log.Error("internal error occurred",
				"url", r.URL.Path,
				"error", err,
			)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write(model.ErrToJson(err.Error()))
		}
	})
}
