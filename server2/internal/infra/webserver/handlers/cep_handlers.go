package handlers

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/url"

	"github.com/go-chi/chi/v5"
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

func GetCepHandler(w http.ResponseWriter, r *http.Request) {
	cepParams := chi.URLParam(r, "cep")
	if len(cepParams) != 8 || cepParams == "00000000" {
		w.WriteHeader(http.StatusUnprocessableEntity)
		json.NewEncoder(w).Encode("invalid zipcode")
		return
	}

	cep, err := getCepViaCEP(cepParams)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode("error getting zipcode")
		return
	}
	temp, err := getTemperature(cep.Localidade)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode("error getting temperature")
		return
	}

	tempFandK := getTemperatureFandK(temp.Current.TempC)

	log.Println("Response CEP:", cep)
	log.Println("Response Temperature:", temp)
	log.Println("Response Temperature F and K:", tempFandK)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(tempFandK)
}

func getCepViaCEP(cepParams string) (*CepResponse, error) {
	req, err := http.NewRequest("GET", "http://viacep.com.br/ws/"+cepParams+"/json/", nil)
	if err != nil {
		log.Printf("Erro ao fazer a requisição HTTP: %v\n", err)
		return nil, err
	}

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
