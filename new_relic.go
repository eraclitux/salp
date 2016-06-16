package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/eraclitux/stracer"
)

const newRelicURI = "https://api.newrelic.com/v2/applications/%s.json"

type NewRelicData struct {
	App NewRelicAppData `json:"application"`
}

type NewRelicAppData struct {
	LastReported time.Time           `json:"last_reported_at"`
	Health       string              `json:"health_status"`
	Summary      NewRelicSummaryData `json:"application_summary"`
}

type NewRelicSummaryData struct {
	Throughput   float64 `json:"throughput"`
	ResponseTime float64 `json:"response_time"`
	ErrorRate    float64 `json:"error_rate"`
	Apdex        float64 `json:"apdex_score"`
}

// FetchNewRelic polls NewRelic api
// and updates generalStatus.
func FetchNewRelic(token, appID string) {
	for {
		stracer.Traceln("calling new relic apis...")
		client := &http.Client{}
		u := fmt.Sprintf(newRelicURI, appID)
		req, _ := http.NewRequest("GET", u, nil)
		req.Header.Add("X-Api-Key", token)
		resp, err := client.Do(req)
		if err != nil {
			ErrorLogger.Println("getting data from NewRelic:", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			ErrorLogger.Println("bad status from NewRelic:", resp.StatusCode)
			return
		}
		var data NewRelicData
		err = json.NewDecoder(resp.Body).Decode(&data)
		if err != nil {
			ErrorLogger.Println("decoding data from NewRelic:", err)
		}
		stracer.PrettyStruct("nr data:", data)
		generalStatus.updateNewRelicData(data.App)
		time.Sleep(10 * time.Minute)
	}
}
