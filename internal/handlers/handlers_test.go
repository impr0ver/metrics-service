package handlers_test

import (
	"io"
	"metrics-service/internal/handlers"
	"metrics-service/internal/storage"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ////////////////////////////
func TestMetricsHandlerPost(t *testing.T) {
	type want struct {
		code        int
		response    string
		contentType string
	}
	tests := []struct {
		name  string
		value string
		want  want
	}{
		{name: "positive test #1", value: "counter/PollCount/4", want: want{code: http.StatusOK, response: "Registered successfully!", contentType: "text/plain"}},
		{name: "positive test #2", value: "gauge/OtherSys/5557552.000000", want: want{code: http.StatusOK, response: "Registered successfully!", contentType: "text/plain"}},
		{name: "negative test #3", value: "gauge/HeapReleased/wow", want: want{code: http.StatusBadRequest, response: "Bad request!", contentType: "text/plain"}},
		{name: "negative test #4", value: "gauge/HelloAll!/foo", want: want{code: http.StatusBadRequest, response: "Bad request!", contentType: "text/plain"}},
		{name: "positive test #5", value: "gauge/Mallocs/12345", want: want{code: http.StatusOK, response: "Registered successfully!", contentType: "text/plain"}},
		{name: "positive test #6", value: "gauge/Alloc/110072.000000", want: want{code: http.StatusOK, response: "Registered successfully!", contentType: "text/plain"}},
		{name: "positive test #7", value: "gauge/NextGC/234.01", want: want{code: http.StatusOK, response: "Registered successfully!", contentType: "text/plain"}},
		{name: "negative test #8", value: "gauge/Sys/", want: want{code: http.StatusNotFound, response: "Not found!", contentType: "text/plain"}},
		{name: "negative test #9", value: "gauge/666.6", want: want{code: http.StatusNotFound, response: "Not found!", contentType: "text/plain"}},
		{name: "negative test #10", value: "gauge/Alloc/110072.000000/this/is/interesting/test", want: want{code: http.StatusNotFound, response: "Not found!", contentType: "text/plain"}},
		{name: "positive test #11", value: "counter/PollCount/4", want: want{code: http.StatusOK, response: "Registered successfully!", contentType: "text/plain"}},
		{name: "negative test #12", value: "counter/PollCount/321.0076", want: want{code: http.StatusBadRequest, response: "Bad request!", contentType: "text/plain"}},
		{name: "positive test #13", value: "counter/testcounter/150", want: want{code: http.StatusOK, response: "Registered successfully!", contentType: "text/plain"}},
		{name: "positive test #14", value: "counter/Mycounter/0", want: want{code: http.StatusOK, response: "Registered successfully!", contentType: "text/plain"}},
		{name: "negative test #15", value: "counter/PollCount/15/this/is/interesting/test", want: want{code: http.StatusNotFound, response: "Not found!", contentType: "text/plain"}},
	}

	for _, tt := range tests {
		// run every test
		t.Run(tt.name, func(t *testing.T) {
			memstorage := storage.MemoryStorage{Gauges: make(map[string]storage.Gauge), Counters: make(map[string]storage.Counter)}
			request := httptest.NewRequest(http.MethodPost, "/update/"+tt.value, nil)

			// create new Recorder
			w := httptest.NewRecorder()
			// set handler
			h := http.HandlerFunc(handlers.MetricsHandlerPost(&memstorage))
			// start server
			h.ServeHTTP(w, request)
			res := w.Result()

			// check res code
			if res.StatusCode != tt.want.code {
				t.Errorf("expected status code %d, got %d", tt.want.code, w.Code)
			}

			// get and check body
			defer res.Body.Close()
			resBody, err := io.ReadAll(res.Body)
			if err != nil {
				t.Fatal(err)
			}
			if string(resBody) != tt.want.response {
				t.Errorf("expected body %s, got %s", tt.want.response, w.Body.String())
			}

			// check res header
			if res.Header.Get("Content-Type") != tt.want.contentType {
				t.Errorf("expected Content-Type %s, got %s", tt.want.contentType, res.Header.Get("Content-Type"))
			}
		})
	}
}

func TestMetricsHandlerGet(t *testing.T) {
	type want struct {
		code  int
		value string
	}

	memstorage := storage.MemoryStorage{Gauges: make(map[string]storage.Gauge),
		Counters: make(map[string]storage.Counter)}

	memstorage.AddNewCounter("tstcounter", storage.Counter(345))
	memstorage.AddNewCounter("tstcounter", storage.Counter(200))
	memstorage.UpdateGauge("Sys", storage.Gauge(12345.12345))

	// foundSys, err := memstorage.GetGaugeByKey("Sys")
	// if err != nil {
	// 	fmt.Println("err", err)
	// }
	// fmt.Println(fmt.Sprintf("%f", foundSys))

	tests := []struct {
		name      string
		valuetype string
		valuename string
		want      want
	}{
		{name: "positive test #1", valuetype: "gauge", valuename: "Sys", want: want{code: http.StatusOK, value: "12345.123450"}},
		{name: "positive test #2", valuetype: "counter", valuename: "tstcounter", want: want{code: http.StatusOK, value: "545"}},
	}
	for _, tt := range tests {
		// запускаем каждый тест
		t.Run(tt.name, func(t *testing.T) {
			//fmt.Println("/value/" + tt.valuetype + "/" + tt.valuename)
			request := httptest.NewRequest(http.MethodGet, "/value/"+tt.valuetype+"/"+tt.valuename, nil)
			//strings.Split(request.RemoteAddr, ":")
			r := handlers.MetricsHandlerGet(&memstorage)
			// создаём новый Recorder
			w := httptest.NewRecorder()
			// определяем хендлер
			r.ServeHTTP(w, request)
			res := w.Result()

			// проверяем код ответа
			if res.StatusCode != tt.want.code {
				t.Errorf("Expected status code %d, got %d", tt.want.code, w.Code)
			}

			// получаем и проверяем тело запроса
			defer res.Body.Close()
			resBody, err := io.ReadAll(res.Body)
			if err != nil {
				t.Fatal(err)
			}
			if string(resBody) != tt.want.value {
				t.Errorf("Expected body %s, got %s", tt.want.value, w.Body.String())
			}
		})
	}
}

func testRequest(t *testing.T, ts *httptest.Server, method,
	path string) (*http.Response, string) {
	req, err := http.NewRequest(method, ts.URL+path, nil)
	require.NoError(t, err)

	resp, err := ts.Client().Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	return resp, string(respBody)
}

func TestMetricsHandlerGet2(t *testing.T) {
	memstorage := storage.MemoryStorage{Gauges: make(map[string]storage.Gauge),
		Counters: make(map[string]storage.Counter)}

	memstorage.AddNewCounter("tstcounter", storage.Counter(345))
	memstorage.AddNewCounter("tstcounter", storage.Counter(200))
	memstorage.UpdateGauge("Sys", storage.Gauge(12345.12345))

	ts := httptest.NewServer(handlers.MetricsHandlerGet(&memstorage))
	defer ts.Close()

	//fmt.Println("TS URL:", ts.URL)

	var testTable = []struct {
		url    string
		want   string
		status int
	}{

		{"/value/gauge/Sys", "12345.123450", http.StatusOK},
		{"/value/counter/tstcounter", "545", http.StatusOK},
	}
	for _, v := range testTable {
		resp, get := testRequest(t, ts, "GET", v.url)
		assert.Equal(t, v.status, resp.StatusCode)
		assert.Equal(t, v.want, get)
	}
}
