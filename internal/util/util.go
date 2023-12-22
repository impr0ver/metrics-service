package util

import (
	"metrics-service/internal/storage"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

func Sleep(suffix string) {
	t, _ := time.ParseDuration(suffix)

	time.Sleep(t)
}

func PrepareUrl(URL *url.URL) []string {
	trimUrl := strings.Trim(URL.String(), "/update") //  "counter/someMetric/527" //0, 1, 2
	return strings.Split(trimUrl, "/")
}

func ParseUrlMetrics(reqMetrics []string, memStor *storage.MemStorage) int {
	if len(reqMetrics) != 3 {
		return http.StatusNotFound
	}
	switch reqMetrics[0] {
	case "counter":
		counterValue, err := strconv.ParseInt(reqMetrics[2], 10, 64)
		if err != nil {
			return http.StatusBadRequest
		}

		_, found := memStor.Counter[reqMetrics[1]]
		if found {
			memStor.Counter[reqMetrics[1]] += counterValue
		} else {
			memStor.Counter[reqMetrics[1]] = counterValue
		}

	case "gauge":
		gaugeValue, err := strconv.ParseFloat(reqMetrics[2], 64)
		if err != nil {
			return http.StatusBadRequest
		}

		memStor.Gauge[reqMetrics[1]] = gaugeValue

	default:
		return http.StatusBadRequest
	}
	return http.StatusOK
}