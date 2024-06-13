package server

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"
	"unicode"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

type TemplateData struct {
	OTELTracer trace.Tracer
}

type Webserver struct {
	TemplateData *TemplateData
}

func NewServer(templateData *TemplateData) *Webserver {
	return &Webserver{TemplateData: templateData}
}

type CepRequest struct {
	Cep string `json:"cep"`
}

type Response struct {
	City  string  `json:"city"`
	TempC float64 `json:"temp_C"`
	TempF float64 `json:"temp_F"`
	TempK float64 `json:"temp_K"`
}

func (we *Webserver) CreateServer() *chi.Mux {
	router := chi.NewRouter()

	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)
	router.Use(middleware.Timeout(60 * time.Second))

	router.Handle("/metrics", promhttp.Handler())
	router.Post("/cep", we.HandleRequest)

	return router
}

func (we *Webserver) HandleRequest(w http.ResponseWriter, r *http.Request) {
	carrier := propagation.HeaderCarrier(r.Header)
	ctx := r.Context()
	ctx = otel.GetTextMapPropagator().Extract(ctx, carrier)
	_, span := we.TemplateData.OTELTracer.Start(ctx, "HandleRequest POST CEP")
	defer span.End()

	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode("error reading body")
		return
	}

	var cepBody CepRequest
	err = json.Unmarshal(body, &cepBody)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		log.Println("Erro ao fazer o Unmarshal do JSON:", err)
		return
	}

	cepBody.Cep = sanitizeString(cepBody.Cep)

	if !validateCep(cepBody.Cep) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		json.NewEncoder(w).Encode("invalid zipcode")
		return
	}

	cepAndTemperature, err := getCepAndTemperature(cepBody.Cep)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode("error getting zipcode")
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(cepAndTemperature)
}

func validateCep(cep string) bool {
	if len(cep) != 8 {
		return false
	}

	for _, char := range cep {
		if !unicode.IsDigit(char) {
			return false
		}
	}

	if cep == "00000000" {
		return false
	}

	return true
}

func sanitizeString(str string) string {
	var sanitized []rune
	for _, char := range str {
		if unicode.IsDigit(char) {
			sanitized = append(sanitized, char)
		}
	}
	return string(sanitized)
}

func getCepAndTemperature(cepParams string) (*Response, error) {
	req, err := http.NewRequest("GET", "http://server2:8081/cep/"+cepParams+"", nil)
	if err != nil {
		log.Printf("Erro ao fazer a requisição HTTP: %v\n", err)
		return nil, err
	}

	otel.GetTextMapPropagator().Inject(req.Context(), propagation.HeaderCarrier(req.Header))
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("Erro ao fazer a requisição HTTP: %v\n", err)
		return nil, err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		log.Printf("Erro ao ler o corpo da resposta: %v\n", err)
		return nil, err
	}

	var resultTemp Response
	err = json.Unmarshal(body, &resultTemp)
	if err != nil {
		log.Println("Erro ao fazer o Unmarshal do JSON:", err)
		return nil, err
	}

	return &resultTemp, nil
}
