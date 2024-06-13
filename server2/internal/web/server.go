package server

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

type CepResponse struct {
	Cep        string `json:"cep"`
	Logradouro string `json:"logradouro"`
	Bairro     string `json:"bairro"`
	Localidade string `json:"localidade"`
	Uf         string `json:"uf"`
}

type TemperatureResponse struct {
	Current struct {
		TempC float64 `json:"temp_c"`
		TempF float64 `json:"temp_f"`
	} `json:"current"`
}

type TransfromTemperatureResponse struct {
	TempC float64 `json:"temp_C"`
	TempF float64 `json:"temp_F"`
	TempK float64 `json:"temp_K"`
}

type Response struct {
	City  string  `json:"city"`
	TempC float64 `json:"temp_C"`
	TempF float64 `json:"temp_F"`
	TempK float64 `json:"temp_K"`
}

type TemplateData struct {
	OTELTracer trace.Tracer
}

type Webserver struct {
	TemplateData *TemplateData
}

func NewServer(templateData *TemplateData) *Webserver {
	return &Webserver{TemplateData: templateData}
}

func (we *Webserver) CreateServer() *chi.Mux {
	router := chi.NewRouter()

	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)
	router.Use(middleware.Timeout(60 * time.Second))

	router.Get("/cep/{cep}", we.HandleRequest)

	return router
}

func (we *Webserver) HandleRequest(w http.ResponseWriter, r *http.Request) {
	carrier := propagation.HeaderCarrier(r.Header)
	ctx := r.Context()
	ctx = otel.GetTextMapPropagator().Extract(ctx, carrier)
	_, span := we.TemplateData.OTELTracer.Start(ctx, "HandleRequest GET CEP")
	defer span.End()

	cepParams := chi.URLParam(r, "cep")

	ctx, spanCep := we.TemplateData.OTELTracer.Start(ctx, "HandleRequest getCepViaCEP")
	cep, err := getCepViaCEP(cepParams)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode("error getting zipcode")
		return
	}
	spanCep.End()

	_, spanTemperature := we.TemplateData.OTELTracer.Start(ctx, "HandleRequest getTemperature")
	temp, err := getTemperature(cep.Localidade)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode("error getting temperature")
		return
	}
	spanTemperature.End()

	tempFandK := getTemperatureFandK(temp.Current.TempC)

	log.Println("Response CEP:", cep)
	log.Println("Response Temperature:", temp)
	log.Println("Response Temperature F and K:", tempFandK)

	response := Response{
		City:  cep.Localidade,
		TempC: temp.Current.TempC,
		TempF: tempFandK.TempF,
		TempK: tempFandK.TempK,
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func getCepViaCEP(cepParams string) (*CepResponse, error) {
	req, err := http.NewRequest("GET", "http://viacep.com.br/ws/"+cepParams+"/json/", nil)
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

	var resultCep CepResponse
	err = json.Unmarshal(body, &resultCep)
	if err != nil {
		log.Println("Erro ao fazer o Unmarshal do JSON:", err)
		return nil, err
	}

	log.Printf("Response ViaCEP: %v", resultCep)

	return &resultCep, nil
}

func getTemperature(locale string) (*TemperatureResponse, error) {
	escapedLocale := url.QueryEscape(locale)

	req, err := http.NewRequest("GET", "https://api.weatherapi.com/v1/current.json?q="+escapedLocale+"&key=0893d285f33543a2a36184203240302", nil)
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

	if res.StatusCode != http.StatusOK {
		log.Printf("Erro na resposta HTTP. Código: %d\n", res.StatusCode)
		log.Println("Corpo da resposta:", string(body))
		return nil, err
	}

	var resultTemperature TemperatureResponse
	err = json.Unmarshal(body, &resultTemperature)
	if err != nil {
		log.Println("Erro ao fazer o Unmarshal do JSON:", err)
		return nil, err
	}

	log.Printf("Response Temperature C: %v\n", resultTemperature.Current.TempC)

	return &resultTemperature, nil
}

func getTemperatureFandK(tempC float64) TransfromTemperatureResponse {
	tempF := (tempC * 1.8) + 32
	tempK := tempC + 273

	log.Printf("Response Temperature F: %v\n", tempF)
	log.Printf("Response Temperature K: %v\n", tempK)

	return TransfromTemperatureResponse{
		TempC: tempC,
		TempF: tempF,
		TempK: tempK,
	}
}
