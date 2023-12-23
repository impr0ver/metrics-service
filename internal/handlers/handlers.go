package handlers

import (
	"fmt"
	"metrics-service/internal/servutil"
	"metrics-service/internal/storage"
	"net/http"
)

func MetricsHandler(memStor *storage.Memory) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			fmt.Println("reqURL:", r.URL) //  "/update/counter/someMetric/527"

			splitUrlMetrics := servutil.PrepareUrl(r.URL.String()) //  "[counter, someMetric, 527]" //0, 1, 2

			statusCode, responseMessage := servutil.ParseUrlMetrics(splitUrlMetrics, memStor)

			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(statusCode)
			w.Write([]byte(responseMessage))

			//4 view result data
			//resultData := fmt.Sprintf("counter: %v\ngauge: %v\n", memStor.Counter, memStor.Gauge)
			//w.Write([]byte(resultData))
		}
	}
}
