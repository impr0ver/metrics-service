package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/impr0ver/metrics-service/internal/gzip"
	"github.com/impr0ver/metrics-service/internal/logger"
	"github.com/impr0ver/metrics-service/internal/storage"

	"github.com/go-chi/chi/v5"
)

const (
	mType   = "mtype"
	mName   = "mname"
	mValue  = "mvalue"
	counter = "counter"
	gauge   = "gauge"
)

func MetricsHandlerPost(memStor storage.MemoryStoragerInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		metricType := chi.URLParam(r, mType)
		metricName := chi.URLParam(r, mName)
		metricValue := chi.URLParam(r, mValue)

		fmt.Println("reqMetrics", metricType, metricName, metricValue)

		switch metricType {
		case counter:
			counterValue, err := strconv.ParseInt(metricValue, 10, 64)
			if err != nil {
				w.Header().Set("Content-Type", "text/plain")
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte("Bad request!"))
				return
			}

			memStor.AddNewCounter(metricName, storage.Counter(counterValue))

			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Registered successfully!"))

		case gauge:
			gaugeValue, err := strconv.ParseFloat(metricValue, 64)
			if err != nil {
				w.Header().Set("Content-Type", "text/plain")
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte("Bad request!"))
				return
			}

			memStor.UpdateGauge(metricName, storage.Gauge(gaugeValue))

			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Registered successfully!"))

		default:
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request!"))
		}
	}
}

func MetricsHandlerGet(memStor storage.MemoryStoragerInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		metricType := chi.URLParam(r, mType)
		metricName := chi.URLParam(r, mName)

		switch metricType {
		case counter:
			foundValue, err := memStor.GetCounterByKey(metricName)
			if err != nil {
				w.Header().Set("Content-Type", "text/plain")
				w.WriteHeader(http.StatusNotFound)
				w.Write([]byte("Bad request!"))
				return
			}
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(fmt.Sprintf("%d", int(foundValue))))
		case gauge:
			foundValue, err := memStor.GetGaugeByKey(metricName)
			if err != nil {
				w.Header().Set("Content-Type", "text/plain")
				w.WriteHeader(http.StatusNotFound)
				w.Write([]byte("Bad request!"))
				return
			}
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(strconv.FormatFloat(float64(foundValue), 'f', -1, 64)))
		default:
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad request!"))
		}
	}
}

func MetricsHandlerGetAll(memStor storage.MemoryStoragerInterface) http.HandlerFunc {

	const tmplHTML = `
<html>
<table>
  <h2>Metrics storage:</h2>
  <thead>
    <tr>
      <th>Metric name</th>
      <th>Metric value</th>
    </tr>
  </thead>
  <tbody>
  {{range .AllMetrics}}
    <tr>
      <td><b>{{.Name}}</b></td>
	  <td>{{.Value}}</td>
    </tr>
    {{end}}
  </tbody>
</table>
</html>`

	tmpl, err := template.New("tmplHTML").Parse(tmplHTML)
	if err != nil {
		panic(err)
	}

	return func(w http.ResponseWriter, r *http.Request) {

		var pContent storage.Pagecontent
		var allMetrics []storage.Metric

		foundCounters := memStor.GetAllCounters()
		foundGauges := memStor.GetAllGauges()

		for name, value := range foundGauges { //range Gauge storage
			allMetrics = append(allMetrics, storage.Metric{Name: name, Value: fmt.Sprintf("%f", value)})
		}
		for name, value := range foundCounters { //range Counter storage
			allMetrics = append(allMetrics, storage.Metric{Name: name, Value: fmt.Sprintf("%d", value)})
		}

		sort.Slice(allMetrics, func(i, j int) bool { //need for unit test for Equal test
			return allMetrics[i].Name < allMetrics[j].Name
		})
		pContent.AllMetrics = allMetrics

		gz := gzip.CompressTextHTML(w)
		defer gz.Close()
		w.Header().Set("Content-Type", "text/html")
		w.Header().Add("Content-Encoding", "gzip")

		tmpl.Execute(gz, pContent)
	}
}

func writeError(err error, httpStatus int, w http.ResponseWriter) bool {
	errMessage := struct {
		Error string `json:"error"`
	}{Error: err.Error()}

	msgbytes, err := json.Marshal(errMessage)
	if err != nil {
		sLogger := logger.NewLogger()
		sLogger.Errorf("error marshal %v", err)
		return false
	}

	w.WriteHeader(httpStatus)
	w.Write(msgbytes)
	return true
}

func MetricsHandlerPostJSON(memStor storage.MemoryStoragerInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		w.Header().Set("Content-Type", "application/json")

		var metric storage.Metrics
		var sLogger = logger.NewLogger()
		err := json.NewDecoder(r.Body).Decode(&metric)
		if err != nil {
			writeError(err, http.StatusBadRequest, w)
			return
		}

		if metric.Delta != nil {
			sLogger.Infoln("reqMetrics", metric.MType, metric.ID, *metric.Delta)
		}
		if metric.Value != nil {
			sLogger.Infoln("reqMetrics", metric.MType, metric.ID, *metric.Value)
		}

		switch metric.MType {
		case counter:
			if metric.Delta == nil {
				writeError(errors.New("bad metric value"), http.StatusBadRequest, w)
				return
			}
			memStor.AddNewCounter(metric.ID, storage.Counter(*metric.Delta))
			realVal, err := memStor.GetCounterByKey(metric.ID)
			if err != nil {
				writeError(err, http.StatusNotFound, w)
				return
			}

			metric.Delta = (*int64)(&realVal)

		case gauge:
			if metric.Value == nil {
				writeError(errors.New("bad metric value"), http.StatusBadRequest, w)
				return
			}
			memStor.UpdateGauge(metric.ID, storage.Gauge(*metric.Value))
			realVal, err := memStor.GetGaugeByKey(metric.ID)
			if err != nil {
				writeError(err, http.StatusNotFound, w)
				return
			}

			metric.Value = (*float64)(&realVal)

		default:
			writeError(errors.New("unsupported metric type"), http.StatusBadRequest, w)
			return
		}

		answer, err := json.Marshal(metric)
		if err != nil {
			writeError(err, http.StatusInternalServerError, w)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write(answer)
	}
}

func MetricsHandlerGetJSON(memStor storage.MemoryStoragerInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		w.Header().Set("Content-Type", "application/json")

		var metric storage.Metrics
		err := json.NewDecoder(r.Body).Decode(&metric)
		if err != nil {
			writeError(err, http.StatusBadRequest, w)
			return
		}

		switch metric.MType {
		case counter:
			realValue, err := memStor.GetCounterByKey(metric.ID)
			if err != nil {
				writeError(err, http.StatusNotFound, w)
				return
			}

			metric.Delta = (*int64)(&realValue)
			metric.Value = nil

		case gauge:
			realValue, err := memStor.GetGaugeByKey(metric.ID)
			if err != nil {
				writeError(err, http.StatusNotFound, w)
				return
			}

			metric.Value = (*float64)(&realValue)
			metric.Delta = nil

		default:
			writeError(errors.New("unsupported metric type"), http.StatusBadRequest, w)
			return
		}

		answer, err := json.Marshal(metric)
		if err != nil {
			writeError(err, http.StatusInternalServerError, w)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write(answer)
	}
}

func logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		start := time.Now()

		lw := logger.NewResponceWriterWithLogging(w)
		sLogger := logger.NewLogger()

		next.ServeHTTP(lw, r) // servicing the original request
		duration := time.Since(start)

		//send request information to zap
		sLogger.Infoln(
			"\033[93m"+"uri", r.RequestURI+"\033[0m",
			"\033[96m"+"method", r.Method+"\033[0m",
			"\033[32m"+"duration", duration.String()+"\033[0m",
			"\033[36m"+"status", lw.ResponseData.Status,
			"size", lw.ResponseData.Size,
			"\033[0m",
		)
	})
}

func gzipMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ow := w

		//Server responce
		//some checks
		contentType := r.Header.Get("Content-Type")
		supportsType := strings.Contains(contentType, "text/html") || strings.Contains(contentType, "application/json")
		acceptEncoding := r.Header.Get("Accept-Encoding")
		supportsGzip := strings.Contains(acceptEncoding, "gzip")
		if supportsGzip && supportsType {
			// compress http.ResponseWriter
			cw := gzip.NewCompressWriter(w)
			ow = cw
			defer cw.Close()
		}

		//Client request
		//check client data in gzip format
		contentEncoding := r.Header.Get("Content-Encoding")
		sendsGzip := strings.Contains(contentEncoding, "gzip")
		if sendsGzip {
			//decompress r.Body
			cr, err := gzip.NewCompressReader(r.Body)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			r.Body = cr
			defer cr.Close()
		}
		next.ServeHTTP(ow, r)
	})
}

func ChiRouter(memStor storage.MemoryStoragerInterface) *chi.Mux {
	r := chi.NewRouter()

	//this chi function do all handmade work in stock! //r.Use(middleware.Compress(5))
	r.Use(gzipMiddleware)

	//this chi function do all handmade work in stock! //r.Use(middleware.Logger)
	r.With(logging).Post("/update/{mtype}/{mname}/{mvalue}", MetricsHandlerPost(memStor))
	r.With(logging).Get("/value/{mtype}/{mname}", MetricsHandlerGet(memStor))
	r.With(logging).Get("/", MetricsHandlerGetAll(memStor))
	r.With(logging).Post("/value/", MetricsHandlerGetJSON(memStor))
	r.With(logging).Post("/update/", MetricsHandlerPostJSON(memStor))

	return r
}
