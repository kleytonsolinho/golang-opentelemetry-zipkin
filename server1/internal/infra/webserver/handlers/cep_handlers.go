package handlers

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
)

type CepRequest struct {
	Cep string `json:"cep"`
}

type TransfromTemperatureResponse struct {
	TempC float64 `json:"temp_C"`
	TempF float64 `json:"temp_F"`
	TempK float64 `json:"temp_K"`
}

func PostCepHandler(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode("error reading body")
		return
	}

	var cepBody CepRequest
	err = json.Unmarshal(body, &cepBody)
	if err != nil {
		log.Println("Erro ao fazer o Unmarshal do JSON:", err)
	}

	if len(cepBody.Cep) != 8 || cepBody.Cep == "00000000" {
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

func getCepAndTemperature(cepParams string) (*TransfromTemperatureResponse, error) {
	req, err := http.NewRequest("GET", "http://server2:8081/cep/"+cepParams+"", nil)
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

	var resultTemp TransfromTemperatureResponse
	err = json.Unmarshal(body, &resultTemp)
	if err != nil {
		log.Println("Erro ao fazer o Unmarshal do JSON:", err)
		return nil, err
	}

	return &resultTemp, nil
}
