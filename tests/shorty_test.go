package tests

import (
	"github.com/brianvoe/gofakeit/v7"
	"github.com/gavv/httpexpect/v2"
	"net/http"
	"net/url"
	"shorty/internal/pkg/random"
	"shorty/internal/server"
	"testing"
)

const (
	host = "localhost:8082"
)

func TestShorty_HappyPath(t *testing.T) {
	u := url.URL{
		Scheme: "http",
		Host:   host,
	}
	alias := random.GenerateRandomString(server.AliasLength)

	e := httpexpect.Default(t, u.String())
	response := e.POST("/v1/url").
		WithBasicAuth("myuser", "mypass").
		WithJSON(server.Request{
			URL:   gofakeit.URL(),
			Alias: alias,
		}).
		Expect().
		Status(http.StatusOK).
		JSON().Object()

	response.Value("alias").IsEqual(alias)
	response.Value("status").IsEqual("ok")
}
