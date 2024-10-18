package server

import (
	"errors"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
	"log/slog"
	"net/http"
	resp "shorty/internal/pkg/api/response"
	"shorty/internal/pkg/logger/slo"
	"shorty/internal/pkg/random"
	"shorty/internal/storage"
)

type RouterApp interface {
	SaveURL(urlToSave string, alias string) (int64, error)
	GetURL(alias string) (string, error)
	DeleteURL(alias string) error
}

type Request struct {
	URL   string `json:"url" validate:"required,url"`
	Alias string `json:"alias,omitempty"`
}

type Response struct {
	resp.Response
	Alias string `json:"alias,omitempty"`
}

const aliasLength = 5

func (ro *router) saveAliasHandler(w http.ResponseWriter, r *http.Request) {
	const operation = "handlers.url.save"
	var req Request

	ro.log.With(
		slog.String("operation", operation),
		slog.String("request_id", middleware.GetReqID(r.Context())), //req tracing
	)

	//parse request
	err := render.DecodeJSON(r.Body, &req)
	if err != nil {
		ro.log.Error("failed to decode request body", slo.Err(err))

		render.JSON(w, r, resp.Error("failed to decode request"))

		return
	}

	ro.log.Info("request body decoded successfully", slog.Any("request", req))

	err = validator.New().Struct(req)
	if err != nil {
		validateErr := err.(validator.ValidationErrors)
		ro.log.Error("invalid request", slo.Err(err))

		//human-readable error text for a client
		render.JSON(w, r, resp.ValidationError(validateErr))

		return
	}

	alias := req.Alias
	if alias == "" {
		alias = random.GenerateRandomString(aliasLength)
		//TODO: check alias uniqueness
	}

	id, err := ro.storage.SaveURL(req.URL, alias)
	if errors.Is(err, storage.ErrURLAlreadyExists) {
		ro.log.Info("url already exists", slog.String("url", req.URL))
		render.JSON(w, r, resp.Error("url already exists"))

		return
	}

	if err != nil {
		ro.log.Error("failed to save url", slo.Err(err))
		render.JSON(w, r, resp.Error("failed to save url"))

		return
	}

	ro.log.Info("url saved successfully", slog.Int64("id", id))

	render.JSON(w, r, Response{
		Response: resp.OK(),
		Alias:    alias,
	})
}

func (ro *router) redirectHandler(w http.ResponseWriter, r *http.Request) {
	const operation = "handlers.url.redirect"

	ro.log.With(
		slog.String("operation", operation),
		slog.String("request_id", middleware.GetReqID(r.Context())),
	)

	alias := chi.URLParam(r, "alias")
	if alias == "" {
		ro.log.Info("alias is empty")
		render.JSON(w, r, resp.Error("invalid request"))

		return
	}

	result, err := ro.storage.GetURL(alias)
	if errors.Is(err, storage.ErrURLNotFound) {
		ro.log.Info("url not found", "alias", alias)
		render.JSON(w, r, resp.Error("url not found for given alias"))

		return
	}

	if err != nil {
		ro.log.Error("failed to get url by given alias", slog.String("alias", alias))
		render.JSON(w, r, resp.Error("internal error")) //omit details for a client

		return
	}

	ro.log.Info("got url", slog.String("url", result))
	http.Redirect(w, r, result, http.StatusFound)
}

func (ro *router) deleteAliasHandler(w http.ResponseWriter, r *http.Request) {}
