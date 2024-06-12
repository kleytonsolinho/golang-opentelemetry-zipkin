package main

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/kleytonsolinho/golang-opentelemetry-zipkin/server2/internal/infra/webserver/handlers"
)

func main() {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/cep/{cep}", handlers.GetCepHandler)

	http.ListenAndServe(":8081", r)
}
