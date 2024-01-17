package handlers

import (
	"fmt"
	"html/template"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/impr0ver/metrics-service/internal/logger"
	"github.com/impr0ver/metrics-service/internal/storage"
	"go.uber.org/zap"

	"github.com/go-chi/chi/v5"
)

const (
	mType   = "mtype"
	mName   = "mname"
	mValue  = "mvalue"
	counter = "counter"
	gauge   = "gauge"
)

func MetricsHandlerPost(memStor *storage.MemoryStorage) http.HandlerFunc {
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
			_, found := memStor.Counters[metricName]
			if found {
				memStor.Counters[metricName] += storage.Counter(counterValue)
			} else {
				memStor.Counters[metricName] = storage.Counter(counterValue)
			}
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
			memStor.Gauges[metricName] = storage.Gauge(gaugeValue)
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

func MetricsHandlerGet(memStor *storage.MemoryStorage) http.HandlerFunc {
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

func MetricsHandlerGetAll(memStor *storage.MemoryStorage) http.HandlerFunc {

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

		tmpl.Execute(w, pContent)
	}
}

func ChiRouter(memStor *storage.MemoryStorage, sLogger *zap.SugaredLogger) *chi.Mux {
	r := chi.NewRouter()

	//r.Use(middleware.Logger) //but this method does all the work

	logging := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			start := time.Now()

			lw := logger.NewResponceWriterWithLogging(w)

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

	r.With(logging).Post("/update/{mtype}/{mname}/{mvalue}", MetricsHandlerPost(memStor))
	r.With(logging).Get("/value/{mtype}/{mname}", MetricsHandlerGet(memStor))
	r.With(logging).Get("/", MetricsHandlerGetAll(memStor))

	return r
}
