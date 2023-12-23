package handlers_test

import (
	"io"
	"metrics-service/internal/handlers"
	"metrics-service/internal/storage"
	"net/http"
	"net/http/httptest"
	"testing"
)

//4 Gauge metrics
func TestMetricsHandlerGauge(t *testing.T) {
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
		{name: "negative test #1", value: "/PollCount/4", want: want{code: http.StatusOK, response: "Registered successfully!", contentType: "text/plain"}},
		{name: "positive test #2", value: "/OtherSys/5557552.000000", want: want{code: http.StatusOK, response: "Registered successfully!", contentType: "text/plain"}},
		{name: "negative test #3", value: "/HeapReleased/wow", want: want{code: http.StatusBadRequest, response: "Bad request!", contentType: "text/plain"}},
		{name: "negative test #4", value: "/HelloAll!/foo", want: want{code: http.StatusBadRequest, response: "Bad request!", contentType: "text/plain"}},
		{name: "positive test #5", value: "/Mallocs/12345", want: want{code: http.StatusOK, response: "Registered successfully!", contentType: "text/plain"}},
		{name: "positive test #6", value: "/Alloc/110072.000000", want: want{code: http.StatusOK, response: "Registered successfully!", contentType: "text/plain"}},
		{name: "positive test #7", value: "/NextGC/234.01", want: want{code: http.StatusOK, response: "Registered successfully!", contentType: "text/plain"}},
		{name: "negative test #8", value: "/Sys/", want: want{code: http.StatusNotFound, response: "Not found!", contentType: "text/plain"}},
		{name: "negative test #9", value: "/666.6", want: want{code: http.StatusNotFound, response: "Not found!", contentType: "text/plain"}},
		{name: "negative test #10", value: "/Alloc/110072.000000/this/is/interesting/test", want: want{code: http.StatusNotFound, response: "Not found!", contentType: "text/plain"}},
		
	}

	for _, tt := range tests {
		// run every test
		t.Run(tt.name, func(t *testing.T) {
			memstorage := storage.InitMemory()
			request := httptest.NewRequest(http.MethodPost, "/update/gauge"+tt.value, nil)

			// create new Recorder
			w := httptest.NewRecorder()
			// set handler
			h := http.HandlerFunc(handlers.MetricsHandler(&memstorage))
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


//4 Count metrics
func TestMetricsHandlerCount(t *testing.T) {
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
		{name: "positive test #1", value: "/PollCount/4", want: want{code: http.StatusOK, response: "Registered successfully!", contentType: "text/plain"}},
		{name: "negative test #2", value: "/PollCount/321.0076", want: want{code: http.StatusBadRequest, response: "Bad request!", contentType: "text/plain"}},
		{name: "positive test #3", value: "/testcounter/150", want: want{code: http.StatusOK, response: "Registered successfully!", contentType: "text/plain"}},
		{name: "positive test #4", value: "/Mycounter/0", want: want{code: http.StatusOK, response: "Registered successfully!", contentType: "text/plain"}},
		{name: "negative test #5", value: "/PollCount/15/this/is/interesting/test", want: want{code: http.StatusNotFound, response: "Not found!", contentType: "text/plain"}},
	
	}

	for _, tt := range tests {
		// run every test
		t.Run(tt.name, func(t *testing.T) {
			memstorage := storage.InitMemory()
			request := httptest.NewRequest(http.MethodPost, "/update/counter"+tt.value, nil)

			// create new Recorder
			w := httptest.NewRecorder()
			// set handler
			h := http.HandlerFunc(handlers.MetricsHandler(&memstorage))
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
