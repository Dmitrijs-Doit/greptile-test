package common

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

type HistoricalResponse struct {
	Date  string             `json:"date"`
	Base  string             `json:"base"`
	Rates map[string]float64 `json:"rates"`
}

type TimeseriesResponse struct {
	StartDate string                        `json:"start_date"`
	EndDate   string                        `json:"end_date"`
	Base      string                        `json:"base"`
	Rates     map[string]map[string]float64 `json:"rates"`
}

const accessKey = "8e0c84521bc40a719783d155ff416733"
const supportedCurrencies = "USD,ILS,EUR,GBP,AUD,CAD,DKK,NOK,SEK,BRL,SGD,MXN,CHF,MYR,TWD,EGP,ZAR,JPY,IDR"

func HistoricalRates(date time.Time) (*HistoricalResponse, error) {
	client := http.DefaultClient
	url := fmt.Sprintf("http://data.fixer.io/api/%s", date.Format("2006-01-02"))
	req, _ := http.NewRequest("GET", url, nil)

	q := req.URL.Query()
	q.Add("access_key", accessKey)
	q.Add("base", "USD")
	q.Add("symbols", supportedCurrencies)
	req.URL.RawQuery = q.Encode()

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	body, err := io.ReadAll(resp.Body)

	defer resp.Body.Close()

	result := &HistoricalResponse{}
	if err := json.Unmarshal(body, result); err != nil {
		log.Println(string(body))
		return nil, err
	}

	return result, err
}

func CurrencyTimeseries(startDate, endDate string) (*TimeseriesResponse, error) {
	client := http.DefaultClient
	url := "http://data.fixer.io/api/timeseries"
	req, _ := http.NewRequest("GET", url, nil)

	q := req.URL.Query()
	q.Add("access_key", accessKey)
	q.Add("start_date", startDate)
	q.Add("end_date", endDate)
	q.Add("base", "USD")
	q.Add("symbols", supportedCurrencies)
	req.URL.RawQuery = q.Encode()

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	body, err := io.ReadAll(resp.Body)

	defer resp.Body.Close()

	result := &TimeseriesResponse{}
	if err := json.Unmarshal(body, result); err != nil {
		log.Println(string(body))
		return nil, err
	}

	return result, err
}
