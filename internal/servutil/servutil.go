package servutil

import (
	"metrics-service/internal/storage"
	"net/http"
	"strconv"
	"strings"
)

func PrepareURL(URL string) []string {
	trimURL := strings.Trim(URL, "/update") //  "counter/someMetric/527" //0, 1, 2
	return strings.Split(trimURL, "/")
}

func ParseURLMetrics(reqMetrics []string, memStor *storage.Memory) (int, string) {
	if len(reqMetrics) != 3 {	//if len(reqMetrics) != 3 {
		return http.StatusNotFound, "Not found!"
	}

	switch reqMetrics[0] {
	case "counter":
		counterValue, err := strconv.ParseInt(reqMetrics[2], 10, 64)
		if err != nil {
			return http.StatusBadRequest, "Bad request!"
		}

		_, found := memStor.Counters[reqMetrics[1]]
		if found {
			memStor.Counters[reqMetrics[1]] += counterValue
		} else {
			memStor.Counters[reqMetrics[1]] = counterValue
		}

	case "gauge":
		gaugeValue, err := strconv.ParseFloat(reqMetrics[2], 64)
		if err != nil {
			return http.StatusBadRequest, "Bad request!"
		}

		memStor.Gauges[reqMetrics[1]] = gaugeValue

	default:
		return http.StatusBadRequest, "Bad request!"
	}
	return http.StatusOK, "Registered successfully!"
}
