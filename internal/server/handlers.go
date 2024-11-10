package server

import (
	"errors"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
	"io"
	"log/slog"
	"net/http"
	resp "shorty/internal/pkg/api/response"
	"shorty/internal/pkg/logger/slo"
	"shorty/internal/pkg/random"
	"shorty/internal/storage"
)

//go:generate mockgen -source=handlers.go -destination=mocks/handlers.go -package=mocks

type UrlProvider interface {
	SaveURL(urlToSave string, alias string) (int64, error)
	GetURL(alias string) (string, error)
	DeleteURL(alias string) error
	UpdateAlias(oldAlias string, newAlias string) error
}

type Request struct {
	URL   string `json:"url" validate:"required,url"`
	Alias string `json:"alias,omitempty"`
}

type UpdateRequest struct {
	NewAlias string `json:"new_alias" validate:"required"`
}

type Response struct {
	resp.Response
	Alias string `json:"alias,omitempty"`
}

const (
	handlersOperationSaveURL     = "handlers.url.save"
	handlersOperationRedirect    = "handlers.url.redirect"
	handlersOperationDelete      = "handlers.url.delete"
	handlersOperationUpdateAlias = "handlers.url.update"
)

const AliasLength = 5

func (ro *router) saveAliasHandler(w http.ResponseWriter, r *http.Request) {
	var req Request

	ro.log.With(
		slog.String("operation", handlersOperationSaveURL),
		slog.String("request_id", middleware.GetReqID(r.Context())), //req tracing
	)

	//parse request
	err := render.DecodeJSON(r.Body, &req)
	if errors.Is(err, io.EOF) {
		ro.log.Error("request body is empty")
		render.JSON(w, r, resp.Error("empty request"))

		return
	}

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
		alias = random.GenerateRandomString(AliasLength)
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

	ro.log.Info("url successfully saved", slog.Int64("id", id))

	render.JSON(w, r, Response{
		Response: resp.OK(),
		Alias:    alias,
	})
}

func (ro *router) redirectHandler(w http.ResponseWriter, r *http.Request) {
	ro.log.With(
		slog.String("operation", handlersOperationRedirect),
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

func (ro *router) deleteAliasHandler(w http.ResponseWriter, r *http.Request) {
	ro.log.With(
		slog.String("operation", handlersOperationDelete),
		slog.String("request_id", middleware.GetReqID(r.Context())),
	)

	alias := chi.URLParam(r, "alias")
	if alias == "" {
		ro.log.Info("alias is empty")
		render.JSON(w, r, resp.Error("invalid request"))

		return
	}

	err := ro.storage.DeleteURL(alias)
	if err != nil {
		ro.log.Error("failed to delete url", slog.String("alias", alias))
		render.JSON(w, r, resp.Error("internal error"))

		return
	}

	ro.log.Info("alias successfully deleted", slog.String("alias", alias))
	w.WriteHeader(http.StatusOK)
}

func (ro *router) updateAliasHandler(w http.ResponseWriter, r *http.Request) {
	var req UpdateRequest

	ro.log.With(
		slog.String("operation", handlersOperationUpdateAlias),
		slog.String("request_id", middleware.GetReqID(r.Context())),
	)

	oldAlias := chi.URLParam(r, "alias")
	if oldAlias == "" {
		ro.log.Info("alias is empty")
		render.JSON(w, r, resp.Error("invalid request"))

		return
	}

	err := render.DecodeJSON(r.Body, &req)
	if errors.Is(err, io.EOF) {
		ro.log.Error("request body is empty")
		render.JSON(w, r, resp.Error("empty request"))

		return
	}

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

	newAlias := req.NewAlias
	if newAlias == "" {
		ro.log.Info("alias was not updated because the field with the new alias is empty")
		render.JSON(w, r, resp.Error("invalid request: new alias is empty"))
		//TODO: check alias uniqueness

		return
	}

	if len(newAlias) < AliasLength {
		ro.log.Info("new alias is too short", slog.String("new_alias", newAlias))
		render.JSON(w, r, resp.Error("invalid request: new alias is too short"))

		return
	}

	if newAlias == oldAlias {
		ro.log.Info("new alias is the same as the old one", slog.String("new_alias", newAlias), slog.String("old_alias", oldAlias))
		render.JSON(w, r, resp.Error("new alias is the same as the old one"))

		return
	}

	err = ro.storage.UpdateAlias(oldAlias, newAlias)
	if err != nil {
		ro.log.Error("failed to update alias", slog.String("old_alias", oldAlias), slog.String("new_alias", newAlias))
		render.JSON(w, r, resp.Error("internal error"))

		return
	}

	ro.log.Info("alias successfully updated", slog.String("old_alias", oldAlias), slog.String("new_alias", newAlias))
	render.JSON(w, r, Response{
		Response: resp.OK(),
		Alias:    newAlias,
	})
}
