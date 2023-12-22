package main

import (
	"fmt"
	"log"
	"metrics-service/internal/storage"
	"metrics-service/internal/util"
	"net/http"
)

var memStor *storage.MemStorage

func metricsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		fmt.Println("reqURL:", r.URL) //  "/update/counter/someMetric/527"

		splitUrlMetrics := util.PrepareUrl(r.URL) //  "[counter, someMetric, 527]" //0, 1, 2

		statusCode := util.ParseUrlMetrics(splitUrlMetrics, memStor)

		w.WriteHeader(statusCode)

		//4 view result data
		//resultData := fmt.Sprintf("counter: %v\ngauge: %v\n", memStor.Counter, memStor.Gauge)
		//w.Write([]byte(resultData))
	}
}

func main() {
	mux := http.NewServeMux()

	memStor = new(storage.MemStorage)
	memStor.Counter = make(map[string]int64)
	memStor.Gauge = make(map[string]float64)

	mux.HandleFunc("/update/", metricsHandler)
	log.Println("Server is listening...")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
