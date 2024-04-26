package handlers_test

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/impr0ver/metrics-service/internal/handlers"
	"github.com/impr0ver/metrics-service/internal/servconfig"
	"github.com/impr0ver/metrics-service/internal/storage"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGzipMiddleware test.
func TestGzipMiddleware(t *testing.T) {
	type metricAlias struct {
		ID    string  `json:"id"`
		MType string  `json:"type"`
		Delta int64   `json:"delta,omitempty"`
		Value float64 `json:"value,omitempty"`
	}
	type want struct {
		metric     metricAlias
		httpStatus int
	}
	tests := []struct {
		name  string
		value metricAlias
		want  want
	}{
		{"simple gauge test #1",
			metricAlias{ID: "Alloc", MType: "gauge"},
			want{metricAlias{ID: "Alloc", MType: "gauge", Value: 555.34, Delta: 0}, http.StatusOK}},
		{"simple counter test #2",
			metricAlias{ID: "MyCounter", MType: "counter"},
			want{metricAlias{ID: "MyCounter", MType: "counter", Delta: 95}, http.StatusOK}},
	}

	memstorage := storage.MemoryStorage{Gauges: make(map[string]storage.Gauge),
		Counters: make(map[string]storage.Counter)}
	memstorage.UpdateGauge(context.TODO(), "Alloc", 555.34)
	memstorage.AddNewCounter(context.TODO(), "MyCounter", 95)

	var cfg = servconfig.Config{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := handlers.ChiRouter(&memstorage, &cfg)

			jData, _ := json.Marshal(tt.value)
			var buf bytes.Buffer
			g := gzip.NewWriter(&buf)
			g.Write(jData)
			g.Close()
			request := httptest.NewRequest(http.MethodPost, "/value/", &buf)
			request.Header.Add("Content-Encoding", "gzip")
			request.Header.Add("Accept-Encoding", "gzip")
			request.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			r.ServeHTTP(w, request)

			res := w.Result()
			defer res.Body.Close()

			if res.StatusCode != tt.want.httpStatus {
				t.Errorf("Expected status code %d, got %d", tt.want.httpStatus, res.StatusCode)
			}
			gr, _ := gzip.NewReader(res.Body)
			var metric metricAlias
			err := json.NewDecoder(gr).Decode(&metric)
			gr.Close()

			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, tt.want.metric, metric)
			switch mtype := tt.want.metric.MType; mtype {
			case "counter":
				v, _ := memstorage.GetCounterByKey(context.TODO(), tt.want.metric.ID)
				assert.Equal(t, int64(v), tt.want.metric.Delta)
			case "gauge":
				v, _ := memstorage.GetGaugeByKey(context.TODO(), tt.want.metric.ID)
				assert.Equal(t, float64(v), tt.want.metric.Value)
			}
		})
	}
}

// TestMetricsHandlerPostJSON test.
func TestMetricsHandlerPostJSON(t *testing.T) {
	type Metrics struct {
		ID    string   `json:"id"`
		MType string   `json:"type"`
		Delta *int64   `json:"delta,omitempty"` // countValue
		Value *float64 `json:"value,omitempty"` // gaugeValue
	}

	type want struct {
		metric     Metrics
		httpStatus int
	}

	var gaugeVal = 1700000.1111
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
	var cfg = servconfig.Config{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := handlers.ChiRouter(&memstorage, &cfg)

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

// TestMetricsHandlerGetJSON test.
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

	var gaugeVal = 234.432
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
	var cfg = servconfig.Config{}
	memstorage.UpdateGauge(context.TODO(), "Sys", 234.432)
	memstorage.AddNewCounter(context.TODO(), "MyCounter", 555)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := handlers.ChiRouter(&memstorage, &cfg)
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

// TestMetricsHandlerGet test.
func TestMetricsHandlerGet(t *testing.T) {
	memstorage := storage.MemoryStorage{Gauges: make(map[string]storage.Gauge),
		Counters: make(map[string]storage.Counter)}
	var cfg = servconfig.Config{}

	//set some metrics in our storage for unit-test
	memstorage.AddNewCounter(context.TODO(), "TstCounter", storage.Counter(345))
	memstorage.AddNewCounter(context.TODO(), "TstCounter", storage.Counter(200))
	memstorage.UpdateGauge(context.TODO(), "Sys", storage.Gauge(12345.1))
	memstorage.UpdateGauge(context.TODO(), "Alloc", storage.Gauge(1764408.9))
	memstorage.UpdateGauge(context.TODO(), "MCacheInuse", storage.Gauge(9600.000123))
	memstorage.UpdateGauge(context.TODO(), "RandomValue", storage.Gauge(0.28))
	memstorage.UpdateGauge(context.TODO(), "GCSys", storage.Gauge(1764408))

	foundSys, err := memstorage.GetGaugeByKey(context.TODO(), "Sys")
	if err != nil {
		fmt.Println("err:", err)
	}
	foundCounter, err := memstorage.GetCounterByKey(context.TODO(), "tstcounter")
	if err != nil {
		fmt.Println("err:", err)
	}

	value := fmt.Sprintf("Sys = %f, tstcounter = %d", foundSys, foundCounter)
	fmt.Println(value)

	ts := httptest.NewServer(handlers.ChiRouter(&memstorage, &cfg))
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

		{"negative test #7", "/value/gauge/NoName", "Bad request!", http.StatusNotFound},
		{"negative test #8", "/value/counter/NoName", "Bad request!", http.StatusNotFound},
	}
	for _, v := range testTable {
		fmt.Printf("fullURL: %s%s", ts.URL, v.url)
		code, get := testRequest(t, ts, "GET", v.url)
		assert.Equal(t, v.status, code, v.name)
		assert.Equal(t, v.want, get, v.name)
	}
}

// TestMetricsHandlerGetAll test.
func TestMetricsHandlerGetAll(t *testing.T) {
	memstorage := storage.MemoryStorage{Gauges: make(map[string]storage.Gauge),
		Counters: make(map[string]storage.Counter)}
	var cfg = servconfig.Config{}

	//set some metrics in our storage for unit-test
	memstorage.AddNewCounter(context.TODO(), "MyTstCounter", storage.Counter(100))
	memstorage.AddNewCounter(context.TODO(), "MyTstCounter", storage.Counter(566)) //must bee "666" = 100 + 566
	memstorage.UpdateGauge(context.TODO(), "PauseTotalNs", storage.Gauge(12345.1))
	memstorage.UpdateGauge(context.TODO(), "MSpanSys", storage.Gauge(1764408.9))
	memstorage.UpdateGauge(context.TODO(), "MCacheInuse", storage.Gauge(9600.000123))
	memstorage.UpdateGauge(context.TODO(), "RandomValue", storage.Gauge(0.99))
	memstorage.UpdateGauge(context.TODO(), "NextGC", storage.Gauge(1764408))

	ts := httptest.NewServer(handlers.ChiRouter(&memstorage, &cfg))
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

// TestMetricsHandlerPost test.
func TestMetricsHandlerPost(t *testing.T) {
	memstorage := storage.MemoryStorage{Gauges: make(map[string]storage.Gauge),
		Counters: make(map[string]storage.Counter)}
	var cfg = servconfig.Config{}

	ts := httptest.NewServer(handlers.ChiRouter(&memstorage, &cfg)) //ts := httptest.NewServer(handlers.MetricsHandlerPost(&memstorage)) !!!
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
		{"negative test #18", "/update/noname/PollCount/15/this/is/interesting/test", "404 page not found\n", http.StatusNotFound},
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

func TestMetricsHandlerPostBatch(t *testing.T) {
	testJSON := `[{ "id": "MCacheSys", "type": "gauge", "value": 15600 },
  { "id": "StackInuse", "type": "gauge", "value": 327680 },
  { "id": "HeapInuse", "type": "gauge", "value": 811008 },
  { "id": "CPUutilization1", "type": "gauge", "value": 1.9801980198269442 },
  { "id": "StackSys", "type": "gauge", "value": 327680 },
  { "id": "GCSys", "type": "gauge", "value": 8055592 },
  { "id": "Alloc", "type": "gauge", "value": 308568 },
  { "id": "MCacheInuse", "type": "gauge", "value": 1200 }]`

	r := handlers.ChiRouter(&memstorage, &cfg)

	bodyReader := strings.NewReader(testJSON)

	request := httptest.NewRequest(http.MethodPost, "/updates/", bodyReader)
	request.Header.Set("Content-Type", "application/json; charset=UTF-8")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, request)

	res := w.Result()
	res.Body.Close()

	respBody, err := io.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err)
	}

	assert.Equal(t, res.StatusCode, 200)
	assert.Equal(t, string(respBody), "Registered successfully!")
}

func TestDataBasePing(t *testing.T) {
	var memStor storage.MemoryStoragerInterface

	testDB, err := testConnectDB(context.TODO())
	if err != nil {
		fmt.Println(err)
	}

	memStor = testDB

	r := handlers.ChiRouter(memStor, &cfg)

	request := httptest.NewRequest(http.MethodGet, "/ping", nil)
	request.Header.Set("Content-Type", "text/plain")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, request)

	res := w.Result()
	defer res.Body.Close()

	respBody, err := io.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err)
	}

	assert.Equal(t, res.StatusCode, 200)
	assert.Equal(t, string(respBody), "DB alive!")
}

// BenchmarkMetricsHandlerPostBatch test.
func BenchmarkMetricsHandlerPostBatch(b *testing.B) {
	testJSON := `[{ "id": "MCacheSys", "type": "gauge", "value": 15600 },
  { "id": "StackInuse", "type": "gauge", "value": 327680 },
  { "id": "HeapInuse", "type": "gauge", "value": 811008 },
  { "id": "CPUutilization1", "type": "gauge", "value": 1.9801980198269442 },
  { "id": "StackSys", "type": "gauge", "value": 327680 },
  { "id": "HeapIdle", "type": "gauge", "value": 3055616 },
  { "id": "BuckHashSys", "type": "gauge", "value": 4022 },
  { "id": "TotalMemory", "type": "gauge", "value": 2046296064 },
  { "id": "LastGC", "type": "gauge", "value": 0 },
  { "id": "NumForcedGC", "type": "gauge", "value": 0 },
  { "id": "HeapSys", "type": "gauge", "value": 3866624 },
  { "id": "HeapAlloc", "type": "gauge", "value": 308568 },
  { "id": "GCSys", "type": "gauge", "value": 8055592 },
  { "id": "Alloc", "type": "gauge", "value": 308568 },
  { "id": "MCacheInuse", "type": "gauge", "value": 1200 }]`

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		b.StopTimer()

		memstorage := storage.MemoryStorage{Gauges: make(map[string]storage.Gauge),
			Counters: make(map[string]storage.Counter)}
		var cfg = servconfig.Config{}

		r := handlers.ChiRouter(&memstorage, &cfg)

		bodyReader := strings.NewReader(testJSON)
		request := httptest.NewRequest(http.MethodPost, "/updates/", bodyReader)
		request.Header.Set("Content-Type", "application/json; charset=UTF-8")
		w := httptest.NewRecorder()
		b.StartTimer()
		r.ServeHTTP(w, request)
		res := w.Result()
		res.Body.Close()
	}
}

// BenchmarkMetricsHandlerPostJSON test.
func BenchmarkMetricsHandlerPostJSON(b *testing.B) {
	type Metrics struct {
		ID    string  `json:"id"`
		MType string  `json:"type"`
		Delta int64   `json:"delta,omitempty"`
		Value float64 `json:"value,omitempty"`
	}
	metric := Metrics{ID: "Alloc", MType: "gauge", Value: 1234.321}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		b.StopTimer()

		memstorage := storage.MemoryStorage{Gauges: make(map[string]storage.Gauge),
			Counters: make(map[string]storage.Counter)}
		var cfg = servconfig.Config{}

		r := handlers.ChiRouter(&memstorage, &cfg)

		mbytes, _ := json.Marshal(metric)
		bodyReader := strings.NewReader(string(mbytes))
		request := httptest.NewRequest(http.MethodPost, "/update/", bodyReader)
		request.Header.Set("Content-Type", "application/json; charset=UTF-8")
		w := httptest.NewRecorder()

		b.StartTimer()

		r.ServeHTTP(w, request)
		res := w.Result()
		res.Body.Close()
	}
}

// BenchmarkMetricsHandlerGetJSON test.
func BenchmarkMetricsHandlerGetJSON(b *testing.B) {
	ctx := context.TODO()
	type Metrics struct {
		ID    string  `json:"id"`
		MType string  `json:"type"`
		Delta int64   `json:"delta,omitempty"`
		Value float64 `json:"value,omitempty"`
	}
	metric := Metrics{ID: "Sys", MType: "gauge"}

	memstorage := storage.MemoryStorage{Gauges: make(map[string]storage.Gauge),
		Counters: make(map[string]storage.Counter)}

	var cfg = servconfig.Config{}

	memstorage.UpdateGauge(ctx, "Sys", 234.432)

	r := handlers.ChiRouter(&memstorage, &cfg)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		b.StopTimer()

		mbytes, _ := json.Marshal(metric)
		bodyReader := strings.NewReader(string(mbytes))
		request := httptest.NewRequest(http.MethodPost, "/value/", bodyReader)
		request.Header.Set("Content-Type", "application/json; charset=UTF-8")
		w := httptest.NewRecorder()
		b.StartTimer()
		r.ServeHTTP(w, request)
		res := w.Result()
		res.Body.Close()
	}
}
