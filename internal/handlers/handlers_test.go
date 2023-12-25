package handlers_test

import (
	"fmt"
	"io"
	"metrics-service/internal/handlers"
	"metrics-service/internal/storage"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

	ts := httptest.NewServer(handlers.ChiRouter(&memstorage)) //ts := httptest.NewServer(handlers.MetricsHandlerGet(&memstorage)) !!!
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
		resp, get := testRequest(t, ts, "GET", v.url)
		assert.Equal(t, v.status, resp.StatusCode, v.name)
		assert.Equal(t, v.want, get, v.name)
	}
}

func TestMetricsHandlerGetAll(t *testing.T) {
	memstorage := storage.MemoryStorage{Gauges: make(map[string]storage.Gauge),
		Counters: make(map[string]storage.Counter)}

	//set some metrics in our storage for unit-test
	memstorage.AddNewCounter("MyTstCounter", storage.Counter(100))
	memstorage.AddNewCounter("MyTstCounter", storage.Counter(566))
	memstorage.UpdateGauge("PauseTotalNs", storage.Gauge(12345.1))
	memstorage.UpdateGauge("MSpanSys", storage.Gauge(1764408.9))
	memstorage.UpdateGauge("MCacheInuse", storage.Gauge(9600.000123))
	memstorage.UpdateGauge("RandomValue", storage.Gauge(0.99))
	memstorage.UpdateGauge("NextGC", storage.Gauge(1764408))

	ts := httptest.NewServer(handlers.ChiRouter(&memstorage)) //ts := httptest.NewServer(handlers.MetricsHandlerGet(&memstorage)) !!!
	defer ts.Close()

	var testTable = []struct {
		name   string
		url    string
		want   string
		status int
	}{
		{"positive test #1", "/", "\n<html>\n<table>\n  <h2>Metrics storage:</h2>\n  <thead>\n    <tr>\n      <th>Metric name</th>\n      <th>Metric value</th>\n    </tr>\n  </thead>\n  <tbody>\n  \n    <tr>\n      <td><b>MCacheInuse</b></td>\n\t  <td>9600.000123</td>\n    </tr>\n    \n    <tr>\n      <td><b>MSpanSys</b></td>\n\t  <td>1764408.900000</td>\n    </tr>\n    \n    <tr>\n      <td><b>MyTstCounter</b></td>\n\t  <td>666</td>\n    </tr>\n    \n    <tr>\n      <td><b>NextGC</b></td>\n\t  <td>1764408.000000</td>\n    </tr>\n    \n    <tr>\n      <td><b>PauseTotalNs</b></td>\n\t  <td>12345.100000</td>\n    </tr>\n    \n    <tr>\n      <td><b>RandomValue</b></td>\n\t  <td>0.990000</td>\n    </tr>\n    \n  </tbody>\n</table>\n</html>",
			http.StatusOK},
	}
	for _, v := range testTable {
		fmt.Printf("fullURL: %s%s", ts.URL, v.url)
		resp, get := testRequest(t, ts, "GET", v.url)
		assert.Equal(t, v.status, resp.StatusCode, v.name)
		assert.Equal(t, v.want, get, v.name)
	}
}


func TestMetricsHandlerPost(t *testing.T) {
	memstorage := storage.MemoryStorage{Gauges: make(map[string]storage.Gauge),
		Counters: make(map[string]storage.Counter)}

	ts := httptest.NewServer(handlers.ChiRouter(&memstorage)) //ts := httptest.NewServer(handlers.MetricsHandlerPost(&memstorage)) !!!
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
		resp, get := testRequest(t, ts, "POST", v.url)
		assert.Equal(t, v.status, resp.StatusCode)
		assert.Equal(t, v.want, get, v.name)
	}
}

func testRequest(t *testing.T, ts *httptest.Server, method, path string) (*http.Response, string) {
	req, err := http.NewRequest(method, ts.URL+path, nil)
	require.NoError(t, err)

	req.Header.Add("Content-type", "text/plain")
	resp, err := ts.Client().Do(req)
	require.NoError(t, err)
	defer resp.Body.Close() //FOR YANDEX AUTOTEST!!! Body is closed!!!

	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	return resp, string(respBody)
}
