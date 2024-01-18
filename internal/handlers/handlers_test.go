package handlers_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/impr0ver/metrics-service/internal/handlers"
	"github.com/impr0ver/metrics-service/internal/logger"
	"github.com/impr0ver/metrics-service/internal/storage"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMetricsHandlerPostJSON(t *testing.T) {
	type Metrics struct {
		ID    string   `json:"id"`
		MType string   `json:"type"`
		Delta *int64   `json:"delta,omitempty"` //countValue
		Value *float64 `json:"value,omitempty"` //gaugeValue
	}

	type want struct {
		metric     Metrics
		httpStatus int
	}

	var gaugeVal float64 = 1700000.1111
	var countVal int64 = 55
	var countVal2 int64 = 5
	var countValRes int64 = 60 //55 + 5

	tests := []struct {
		name  string
		value Metrics
		want  want
	}{
		{"test gauge #1",
			Metrics{ID: "Sys", MType: "gauge", Value: (*float64)(&gaugeVal)},
			want{Metrics{ID: "Sys", MType: "gauge", Value: &gaugeVal, Delta: nil}, http.StatusOK}},
		{"test counter #2",
			Metrics{ID: "MyCount", MType: "counter", Delta: (*int64)(&countVal)},
			want{Metrics{ID: "MyCount", MType: "counter", Delta: &countVal}, http.StatusOK}},
		{"test add counter test #3",
			Metrics{ID: "MyCount", MType: "counter", Delta: (*int64)(&countVal2)},
			want{Metrics{ID: "MyCount", MType: "counter", Delta: &countValRes}, http.StatusOK}},
		{"test counter #4",
			Metrics{ID: "NewCounter", MType: "counter", Delta: (*int64)(&countVal)},
			want{Metrics{ID: "NewCounter", MType: "counter", Value: nil, Delta: &countVal}, http.StatusOK}},
	}
	memstorage := storage.MemoryStorage{Gauges: make(map[string]storage.Gauge),
		Counters: make(map[string]storage.Counter)}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var sLogger = logger.NewLogger()
			r := handlers.ChiRouter(&memstorage, sLogger)

			mbytes, _ := json.Marshal(tt.value)
			bodyReader := strings.NewReader(string(mbytes))
			request := httptest.NewRequest(http.MethodPost, "/update/", bodyReader)
			request.Header.Set("Content-Type", "application/json; charset=UTF-8")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, request)

			res := w.Result()

			//check status code
			if res.StatusCode != tt.want.httpStatus {
				t.Errorf("expected status code %d, got %d", tt.want.httpStatus, res.StatusCode)
			}
			var metric Metrics

			err := json.NewDecoder(res.Body).Decode(&metric)
			res.Body.Close()

			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, tt.want.metric, metric)
		})
	}
}

func TestMetricsHandlerGetJSON(t *testing.T) {
	type Metrics struct {
		ID    string   `json:"id"`
		MType string   `json:"type"`
		Delta *int64   `json:"delta,omitempty"` //countValue
		Value *float64 `json:"value,omitempty"` //gaugeValue
	}
	type want struct {
		metric     Metrics
		httpStatus int
	}

	var gaugeVal float64 = 234.432
	var countVal int64 = 555

	tests := []struct {
		name  string
		value Metrics
		want  want
	}{
		{"simple gauge test #1",
			Metrics{ID: "Sys", MType: "gauge"},
			want{Metrics{ID: "Sys", MType: "gauge", Value: (*float64)(&gaugeVal), Delta: nil}, http.StatusOK}},
		{"simple counter test #2",
			Metrics{ID: "MyCounter", MType: "counter"},
			want{Metrics{ID: "MyCounter", MType: "counter", Delta: (*int64)(&countVal)}, http.StatusOK}},
	}
	memstorage := storage.MemoryStorage{Gauges: make(map[string]storage.Gauge),
		Counters: make(map[string]storage.Counter)}
	memstorage.UpdateGauge("Sys", 234.432)
	memstorage.AddNewCounter("MyCounter", 555)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var sLogger = logger.NewLogger()
			r := handlers.ChiRouter(&memstorage, sLogger)
			mbytes, _ := json.Marshal(tt.value)
			bodyReader := strings.NewReader(string(mbytes))
			request := httptest.NewRequest(http.MethodPost, "/value/", bodyReader)
			request.Header.Set("Content-Type", "application/json; charset=UTF-8")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, request)

			res := w.Result()

			if res.StatusCode != tt.want.httpStatus {
				t.Errorf("Expected status code %d, got %d", tt.want.httpStatus, res.StatusCode)
			}
			var metric Metrics
			err := json.NewDecoder(res.Body).Decode(&metric)
			res.Body.Close()

			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, tt.want.metric, metric)
		})
	}
}

func TestMetricsHandlerGet(t *testing.T) {
	memstorage := storage.MemoryStorage{Gauges: make(map[string]storage.Gauge),
		Counters: make(map[string]storage.Counter)}

	//set some metrics in our storage for unit-test
	memstorage.AddNewCounter("TstCounter", storage.Counter(345))
	memstorage.AddNewCounter("TstCounter", storage.Counter(200))
	memstorage.UpdateGauge("Sys", storage.Gauge(12345.1))
	memstorage.UpdateGauge("Alloc", storage.Gauge(1764408.9))
	memstorage.UpdateGauge("MCacheInuse", storage.Gauge(9600.000123))
	memstorage.UpdateGauge("RandomValue", storage.Gauge(0.28))
	memstorage.UpdateGauge("GCSys", storage.Gauge(1764408))

	foundSys, err := memstorage.GetGaugeByKey("Sys")
	if err != nil {
		fmt.Println("err:", err)
	}
	foundCounter, err := memstorage.GetCounterByKey("tstcounter")
	if err != nil {
		fmt.Println("err:", err)
	}

	value := fmt.Sprintf("Sys = %f, tstcounter = %d", foundSys, foundCounter)
	fmt.Println(value)

	var sLogger = logger.NewLogger()
	ts := httptest.NewServer(handlers.ChiRouter(&memstorage, sLogger))
	defer ts.Close()

	var testTable = []struct {
		name   string
		url    string
		want   string
		status int
	}{
		{"positive test #1", "/value/gauge/Sys", "12345.1", http.StatusOK},
		{"positive test #2", "/value/counter/TstCounter", "545", http.StatusOK}, //345 + 200
		{"positive test #3", "/value/gauge/Alloc", "1764408.9", http.StatusOK},
		{"positive test #4", "/value/gauge/MCacheInuse", "9600.000123", http.StatusOK},
		{"positive test #5", "/value/gauge/RandomValue", "0.28", http.StatusOK},
		{"positive test #6", "/value/gauge/GCSys", "1764408", http.StatusOK},
	}
	for _, v := range testTable {
		fmt.Printf("fullURL: %s%s", ts.URL, v.url)
		code, get := testRequest(t, ts, "GET", v.url)
		assert.Equal(t, v.status, code, v.name)
		assert.Equal(t, v.want, get, v.name)
	}
}

func TestMetricsHandlerGetAll(t *testing.T) {
	memstorage := storage.MemoryStorage{Gauges: make(map[string]storage.Gauge),
		Counters: make(map[string]storage.Counter)}

	//set some metrics in our storage for unit-test
	memstorage.AddNewCounter("MyTstCounter", storage.Counter(100))
	memstorage.AddNewCounter("MyTstCounter", storage.Counter(566)) //must bee "666" = 100 + 566
	memstorage.UpdateGauge("PauseTotalNs", storage.Gauge(12345.1))
	memstorage.UpdateGauge("MSpanSys", storage.Gauge(1764408.9))
	memstorage.UpdateGauge("MCacheInuse", storage.Gauge(9600.000123))
	memstorage.UpdateGauge("RandomValue", storage.Gauge(0.99))
	memstorage.UpdateGauge("NextGC", storage.Gauge(1764408))

	var sLogger = logger.NewLogger()
	ts := httptest.NewServer(handlers.ChiRouter(&memstorage, sLogger))
	defer ts.Close()

	var testTable = []struct {
		name   string
		url    string
		want   string
		status int
	}{
		{"test #1", "/", "\n<html>\n<table>\n  <h2>Metrics storage:</h2>\n  <thead>\n    <tr>\n      <th>Metric name</th>\n      <th>Metric value</th>\n    </tr>\n  </thead>\n  <tbody>\n  \n    <tr>\n      <td><b>MCacheInuse</b></td>\n\t  <td>9600.000123</td>\n    </tr>\n    \n    <tr>\n      <td><b>MSpanSys</b></td>\n\t  <td>1764408.900000</td>\n    </tr>\n    \n    <tr>\n      <td><b>MyTstCounter</b></td>\n\t  <td>666</td>\n    </tr>\n    \n    <tr>\n      <td><b>NextGC</b></td>\n\t  <td>1764408.000000</td>\n    </tr>\n    \n    <tr>\n      <td><b>PauseTotalNs</b></td>\n\t  <td>12345.100000</td>\n    </tr>\n    \n    <tr>\n      <td><b>RandomValue</b></td>\n\t  <td>0.990000</td>\n    </tr>\n    \n  </tbody>\n</table>\n</html>",
			http.StatusOK},
	}
	for _, v := range testTable {
		fmt.Printf("fullURL: %s%s", ts.URL, v.url)
		code, get := testRequest(t, ts, "GET", v.url)
		assert.Equal(t, v.status, code, v.name)
		assert.Equal(t, v.want, get, v.name)
	}
}

func TestMetricsHandlerPost(t *testing.T) {
	memstorage := storage.MemoryStorage{Gauges: make(map[string]storage.Gauge),
		Counters: make(map[string]storage.Counter)}

	var sLogger = logger.NewLogger()
	ts := httptest.NewServer(handlers.ChiRouter(&memstorage, sLogger)) //ts := httptest.NewServer(handlers.MetricsHandlerPost(&memstorage)) !!!
	defer ts.Close()

	var testTable = []struct {
		name   string
		url    string
		want   string
		status int
	}{
		{"positive test #1", "/update/gauge/OtherSys/5557552.000000", "Registered successfully!", http.StatusOK},
		{"positive test #2", "/update/gauge/Mallocs/12345", "Registered successfully!", http.StatusOK},
		{"positive test #3", "/update/counter/PollCount/4", "Registered successfully!", http.StatusOK},
		{"positive test #4", "/update/gauge/OtherSys/5557552.000000", "Registered successfully!", http.StatusOK},
		{"negative test #5", "/update/gauge/HeapReleased/wow", "Bad request!", http.StatusBadRequest},
		{"negative test #6", "/update/gauge/HelloAll!/foo", "Bad request!", http.StatusBadRequest},
		{"positive test #7", "/update/gauge/Mallocs/12345", "Registered successfully!", http.StatusOK},
		{"positive test #8", "/update/gauge/Alloc/110072.000000", "Registered successfully!", http.StatusOK},
		{"positive test #9", "/update/gauge/NextGC/234.01", "Registered successfully!", http.StatusOK},
		{"negative test #10", "/update/gauge/Sys/", "404 page not found\n", http.StatusNotFound},  //этот тест обрабатывает сам роутер "chi"
		{"negative test #11", "/update/gauge/666.6", "404 page not found\n", http.StatusNotFound}, //этот тест обрабатывает сам роутер "chi"
		{"negative test #12", "/update/gauge/Alloc/110072.000000/this/is/interesting/test", "404 page not found\n", http.StatusNotFound},
		{"positive test #13", "/update/counter/PollCount/4", "Registered successfully!", http.StatusOK},
		{"negative test #14", "/update/counter/PollCount/321.0076", "Bad request!", http.StatusBadRequest},
		{"positive test #15", "/update/counter/testcounter/150", "Registered successfully!", http.StatusOK},
		{"positive test #16", "/update/counter/Mycounter/0", "Registered successfully!", http.StatusOK},
		{"negative test #17", "/update/counter/PollCount/15/this/is/interesting/test", "404 page not found\n", http.StatusNotFound}, //этот тест обрабатывает сам роутер "chi"
	}
	for _, v := range testTable {
		fmt.Printf("fullURL: %s%s\n", ts.URL, v.url)
		code, get := testRequest(t, ts, "POST", v.url)
		assert.Equal(t, v.status, code)
		assert.Equal(t, v.want, get, v.name)
	}
}

func testRequest(t *testing.T, ts *httptest.Server, method, path string) (int, string) {
	req, err := http.NewRequest(method, ts.URL+path, nil)
	require.NoError(t, err)

	req.Header.Add("Content-type", "text/plain")
	resp, err := ts.Client().Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	return resp.StatusCode, string(respBody)
}
