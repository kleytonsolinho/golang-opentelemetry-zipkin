package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	opentelemetry "github.com/kleytonsolinho/golang-opentelemetry-zipkin/.open-telemetry"
	server "github.com/kleytonsolinho/golang-opentelemetry-zipkin/server2/internal/web"
	"go.opentelemetry.io/otel"
)

func main() {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	shutdown, err := opentelemetry.InitProvider("server2", "otel-collector:4317")
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := shutdown(ctx); err != nil {
			log.Fatal("failed to shutdown TracerProvider: %w", err)
		}
	}()

	tracer := otel.Tracer("microservice-tracer")

	templateData := &server.TemplateData{
		OTELTracer: tracer,
	}

	server := server.NewServer(templateData)
	router := server.CreateServer()

	go func() {
		log.Println("Starting server on port 8081")
		if err := http.ListenAndServe(":8081", router); err != nil {
			log.Fatal(err)
		}
	}()

	select {
	case <-sigCh:
		log.Println("Shutting down gracefully, CTRL+C pressed...")
	case <-ctx.Done():
		log.Println("Shutting down due to other reason...")
	}

	_, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
}
