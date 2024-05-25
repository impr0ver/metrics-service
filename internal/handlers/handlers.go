// Handlers package contains endpoint handlers for accessing and reporting various metrics.
// The package implements its own function for compressing and decompressing http-requests and http-responses.
// Metrics are received in the JSON-form (struct internals/storage/Metrics).
package handlers

import (
	"bytes"
	"context"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"net"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/impr0ver/metrics-service/internal/crypt"
	"github.com/impr0ver/metrics-service/internal/gzip"
	"github.com/impr0ver/metrics-service/internal/logger"
	proto "github.com/impr0ver/metrics-service/internal/rpc"
	"github.com/impr0ver/metrics-service/internal/servconfig"
	"github.com/impr0ver/metrics-service/internal/storage"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	status "google.golang.org/grpc/status"
)

type RPC struct {
	proto.UnimplementedMetricsExhangeServer
	servconfig.Config
	Ms storage.MemoryStoragerInterface
}

const (
	mType   = "mtype"
	mName   = "mname"
	mValue  = "mvalue"
	counter = "counter"
	gauge   = "gauge"
)

var (
	signKey           string                         // secret key from servconfig
	defaultCtxTimeout = servconfig.DefaultCtxTimeout // default context timeout from servconfig
)

func (r RPC) Update(ctx context.Context, m *proto.Metrics) (*proto.MetricsUpdateResponse, error) {
	res := proto.MetricsUpdateResponse{}

	switch m.Mtype {
	case proto.Metrics_GAUGE:
		err := r.Ms.UpdateGauge(ctx, m.Id, storage.Gauge(m.Value))
		if err != nil {
			return &res, status.Errorf(codes.Internal, "internal error %v", err)
		}
	case proto.Metrics_COUNTER:
		err := r.Ms.AddNewCounter(ctx, m.Id, storage.Counter(m.Delta))
		if err != nil {
			return &res, status.Errorf(codes.Internal, "internal error %v", err)
		}
		actual, err := r.Ms.GetCounterByKey(ctx, m.Id)
		if err != nil {
			return &res, status.Errorf(codes.Internal, "internal error %v", err)
		}
		m.Delta = (int64)(actual)
	default:
		return &res, status.Errorf(codes.InvalidArgument, "unknown metric type")
	}

	res.Metric = m
	return &res, nil
}

func (r RPC) Updates(ctx context.Context, m *proto.MetricsArray) (*proto.MetricsUpdatesResponse, error) {
	res := proto.MetricsUpdatesResponse{}
	metricsSlice := make([]storage.Metrics, 0, 32)
	var storageMetrics storage.Metrics

	for _, metric := range m.Metrics {
		storageMetrics.Delta = &metric.Delta
		storageMetrics.Value = &metric.Value

		if metric.Mtype == proto.Metrics_GAUGE {
			storageMetrics.MType = "gauge"
		} else {
			storageMetrics.MType = "counter"
		}
		storageMetrics.ID = metric.Id
		metricsSlice = append(metricsSlice, storageMetrics)
	}

	err := r.Ms.AddNewMetricsAsBatch(ctx, metricsSlice)
	if err != nil {
		return &res, status.Errorf(codes.Internal, "internal error %v", err)
	}

	return &res, nil
}

func (r RPC) CryptUpdates(ctx context.Context, cm *proto.CryptMetrics) (*proto.MetricsUpdatesResponse, error) {
	res := proto.MetricsUpdatesResponse{}
	metrics := proto.MetricsArray{}
	metricsSlice := make([]storage.Metrics, 0, 32)
	var storageMetrics storage.Metrics

	if err := json.Unmarshal(cm.Plainbuff, &metrics); err != nil {
		return nil, status.Errorf(codes.Internal, "can not unmarshal send data: %v", err)
	}

	for _, metric := range metrics.Metrics {
		storageMetrics.Delta = &metric.Delta
		storageMetrics.Value = &metric.Value

		if metric.Mtype == proto.Metrics_GAUGE {
			storageMetrics.MType = "gauge"
		} else {
			storageMetrics.MType = "counter"
		}
		storageMetrics.ID = metric.Id
		metricsSlice = append(metricsSlice, storageMetrics)
	}

	err := r.Ms.AddNewMetricsAsBatch(ctx, metricsSlice)
	if err != nil {
		return &res, status.Errorf(codes.Internal, "internal error %v", err)
	}

	return &res, nil
}

func (r RPC) GetValue(ctx context.Context, m *proto.Metrics) (*proto.Metrics, error) {
	var metric proto.Metrics
	metric.Id = m.Id

	switch m.Mtype {
	case proto.Metrics_GAUGE:
		v, err := r.Ms.GetGaugeByKey(ctx, m.Id)
		if err != nil {
			return nil, status.Errorf(codes.NotFound, "not found, err: %v", err)
		}
		metric.Value = (float64)(v)
		metric.Mtype = proto.Metrics_GAUGE
	case proto.Metrics_COUNTER:
		v, err := r.Ms.GetCounterByKey(ctx, m.Id)
		if err != nil {
			return nil, status.Errorf(codes.NotFound, "not found, err: %v", err)
		}
		metric.Delta = (int64)(v)
		metric.Mtype = proto.Metrics_COUNTER
	default:
		return nil, status.Errorf(codes.InvalidArgument, "unknown metric type")
	}

	return &metric, nil
}

// MetricsHandlerPost endpoint handler "/update/{mtype}/{mname}/{mvalue}" metric update.
// Type can take two values: "gauge" or "counter".
func MetricsHandlerPost(memStor storage.MemoryStoragerInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		metricType := chi.URLParam(r, mType)
		metricName := chi.URLParam(r, mName)
		metricValue := chi.URLParam(r, mValue)

		fmt.Println("reqMetrics", metricType, metricName, metricValue)

		ctx, cancel := context.WithTimeout(r.Context(), defaultCtxTimeout)
		defer cancel()

		switch metricType {
		case counter:
			counterValue, err := strconv.ParseInt(metricValue, 10, 64)
			if err != nil {
				w.Header().Set("Content-Type", "text/plain")
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte("Bad request!"))
				return
			}

			memStor.AddNewCounter(ctx, metricName, storage.Counter(counterValue))

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

			memStor.UpdateGauge(ctx, metricName, storage.Gauge(gaugeValue))

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

// MetricsHandlerGet endpoint handler "/value/{mtype}/{mname}".
// Returns the current value of the requested metric.
func MetricsHandlerGet(memStor storage.MemoryStoragerInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		metricType := chi.URLParam(r, mType)
		metricName := chi.URLParam(r, mName)

		ctx, cancel := context.WithTimeout(r.Context(), defaultCtxTimeout)
		defer cancel()

		switch metricType {
		case counter:
			foundValue, err := memStor.GetCounterByKey(ctx, metricName)
			if err != nil {
				w.Header().Set("Content-Type", "text/plain")
				w.WriteHeader(http.StatusNotFound)
				w.Write([]byte("Bad request!"))
				return
			}
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(fmt.Sprintf("%d", int(foundValue))))
		case gauge:
			foundValue, err := memStor.GetGaugeByKey(ctx, metricName)
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

// MetricsHandlerGetAll endpoint handler "/", get all metrics in browser.
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

		ctx, cancel := context.WithTimeout(r.Context(), defaultCtxTimeout)
		defer cancel()

		foundCounters, err := memStor.GetAllCounters(ctx)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte{})
			return
		}

		foundGauges, err := memStor.GetAllGauges(ctx)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte{})
			return
		}

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

// MetricsHandlerPostJSON endpoint handler "/update/", metric update.
// Accepts JSON of storage.Metrics.
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

		ctx, cancel := context.WithTimeout(r.Context(), defaultCtxTimeout)
		defer cancel()

		switch metric.MType {
		case counter:
			if metric.Delta == nil {
				writeError(errors.New("bad metric value"), http.StatusBadRequest, w)
				return
			}
			memStor.AddNewCounter(ctx, metric.ID, storage.Counter(*metric.Delta))
			realVal, err := memStor.GetCounterByKey(ctx, metric.ID)
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
			memStor.UpdateGauge(ctx, metric.ID, storage.Gauge(*metric.Value))
			realVal, err := memStor.GetGaugeByKey(ctx, metric.ID)
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

// MetricsHandlerGetJSON endpoint handler "/value/".
// Returns the value of the metric whose name was specified in the input JSON of the storage.Metrics structure.
func MetricsHandlerGetJSON(memStor storage.MemoryStoragerInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		w.Header().Set("Content-Type", "application/json")

		var metric storage.Metrics
		err := json.NewDecoder(r.Body).Decode(&metric)
		if err != nil {
			writeError(err, http.StatusBadRequest, w)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), defaultCtxTimeout)
		defer cancel()

		switch metric.MType {
		case counter:
			realValue, err := memStor.GetCounterByKey(ctx, metric.ID)
			if err != nil {
				writeError(err, http.StatusNotFound, w)
				return
			}

			metric.Delta = (*int64)(&realValue)
			metric.Value = nil

		case gauge:
			realValue, err := memStor.GetGaugeByKey(ctx, metric.ID)
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

// MetricsHandlerPostBatch endpoint handler "/updates/", metrics update.
// Accepts JSON slice of storage.Metrics.
func MetricsHandlerPostBatch(memStor storage.MemoryStoragerInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var allMetrics []storage.Metrics

		decData := json.NewDecoder(r.Body)

		_, err := decData.Token()
		if err != nil {
			writeError(err, http.StatusBadRequest, w)
			return
		}
		for decData.More() {
			var metric storage.Metrics
			err := decData.Decode(&metric)
			if err != nil {
				writeError(err, http.StatusBadRequest, w)
				return
			}
			allMetrics = append(allMetrics, metric)
		}

		ctx, cancel := context.WithTimeout(r.Context(), defaultCtxTimeout)
		defer cancel()

		err = memStor.AddNewMetricsAsBatch(ctx, allMetrics)
		if err != nil {
			writeError(err, http.StatusInternalServerError, w)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Registered successfully!"))
	}
}

// DataBasePing endpoint handler "/ping", checks for a connection to the database.
func DataBasePing(memStor storage.MemoryStoragerInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), defaultCtxTimeout)
		defer cancel()

		if err := memStor.DBPing(ctx); err != nil {
			writeError(err, http.StatusInternalServerError, w)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("DB alive!"))
	}
}

// gzipMiddleware compress and decompress data on middleware.
func gzipMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ow := w

		// some checks for server response
		contentType := r.Header.Get("Content-Type")
		supportsType := strings.Contains(contentType, "text/html") || strings.Contains(contentType, "application/json")
		acceptEncoding := r.Header.Get("Accept-Encoding")
		supportsGzip := strings.Contains(acceptEncoding, "gzip")
		if supportsGzip && supportsType {
			cw := gzip.NewCompressWriter(w) // compress http.ResponseWriter
			ow = cw
			defer cw.Close()
		}

		// check client data in gzip format
		contentEncoding := r.Header.Get("Content-Encoding")
		sendsGzip := strings.Contains(contentEncoding, "gzip")
		if sendsGzip {
			cr, err := gzip.NewCompressReader(r.Body) // decompress r.Body
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

// verifyDataMiddleware check hash from request.
func VerifyDataMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sLogger := logger.NewLogger()

		if signKey != "" {
			reqHash := r.Header.Get("HashSHA256")
			if reqHash != "" {
				bodyBytes, err := io.ReadAll(r.Body)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				rBodyCopy := io.NopCloser(bytes.NewBuffer(bodyBytes)) //use this, because r.Body empty after io.ReadAll (can only read it once)!
				r.Body = rBodyCopy

				resultHash, _ := crypt.SignDataWithSHA256(bodyBytes, signKey)
				w.Header().Add("HashSHA256", resultHash)

				if !crypt.CheckHashSHA256(resultHash, reqHash) {
					sLogger.Infoln("signature is incorrect")
					w.WriteHeader(http.StatusBadRequest)
					return
				}
			}
		}
		next.ServeHTTP(w, r)
	})
}

func DecriptDataMiddleware(privateKey *rsa.PrivateKey) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sLogger := logger.NewLogger()

			if privateKey != nil {
				ct := r.Header.Get("Content-type")
				if ct != "application/octet-stream" {
					next.ServeHTTP(w, r)
				}
				ciphertext, err := io.ReadAll(r.Body)
				if err != nil {
					sLogger.Error("decriptMiddleware: ReadAll error, %v", err)
					return
				}
				jsonbytes, err := crypt.DecryptPKCS1v15(privateKey, ciphertext)
				if err != nil {
					sLogger.Error("decriptMiddleware: DecryptMsg error, %v", err)
					return
				}
				r.Header.Set("Content-type", "application/json; charset=utf-8")
				r.Body.Close()
				r.Body = io.NopCloser(bytes.NewBuffer(jsonbytes))
			}
			next.ServeHTTP(w, r)
		})
	}
}

func CheckIPMiddleware(trustedSubnet string) func(next http.Handler) http.Handler {
	sLogger := logger.NewLogger()
	_, subnet, err := net.ParseCIDR(trustedSubnet)
	if err != nil {
		sLogger.Panic(err)
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			realIP := r.Header.Get("X-Real-IP")

			if realIP == "" {
				ip, _, err := net.SplitHostPort(r.RemoteAddr)
				realIP = ip
				if err != nil {
					sLogger.Errorf("checkIPMiddleware: error, %v", err)
					return
				}
			}

			netIP := net.ParseIP(realIP)

			if !subnet.Contains(netIP) {
				sLogger.Infof("Forbidden! Trusted subnet \"%s\" is not contains IP %s", subnet.String(), netIP.String())
				w.WriteHeader(http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// ChiRouter initializing and setting up the router.
func ChiRouter(memStor storage.MemoryStoragerInterface, cfg *servconfig.Config) *chi.Mux {
	r := chi.NewRouter()

	signKey = cfg.Key

	r.Use(middleware.Logger) // replace my custom logger on chi middleware logger

	// Middleware sequence:
	// 1. Check remote IP for access;
	// 2. Check verify sending data;
	// 3. Decrypt data if "PrivateKey" is set (RSA with PKCS1v15);
	// 4. Gzip/ungzip data.
	r.Use(CheckIPMiddleware(cfg.TrustedSubnet), VerifyDataMiddleware, DecriptDataMiddleware(cfg.PrivateKey), gzipMiddleware)

	r.Mount("/debug", middleware.Profiler()) // add pprof via chi

	// Handlers.
	r.Post("/update/{mtype}/{mname}/{mvalue}", MetricsHandlerPost(memStor))
	r.Get("/value/{mtype}/{mname}", MetricsHandlerGet(memStor))
	r.Get("/", MetricsHandlerGetAll(memStor))
	r.Post("/value/", MetricsHandlerGetJSON(memStor))
	r.Post("/update/", MetricsHandlerPostJSON(memStor))
	r.Get("/ping", DataBasePing(memStor))
	r.Post("/updates/", MetricsHandlerPostBatch(memStor))

	return r
}

func LoggingInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	var sLogger = logger.NewLogger()
	sLogger.Infof("FullMethod: %s, Received request: %v", info.FullMethod, req)
	resp, err := handler(ctx, req)
	return resp, err
}

func VerifyDataInterceptor(c servconfig.Config) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		var hash string

		if c.Key != "" {
			if md, ok := metadata.FromIncomingContext(ctx); ok {
				values := md.Get("hashsha256")
				if len(values) > 0 {
					hash = values[0]
				}
			}

			reqStr := fmt.Sprint(req)
			resultHash, _ := crypt.SignDataWithSHA256([]byte(reqStr), c.Key)

			if !crypt.CheckHashSHA256(resultHash, hash) {
				return nil, status.Error(codes.Internal, "signature is incorrect")
			}
		}
		return handler(ctx, req)
	}
}

func DecryptDataInterceptor(c servconfig.Config) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {

		if c.PrivateKey != nil {
			//type assertion
			if cryptMetrics, ok := req.(*proto.CryptMetrics); ok {
				cryptMetrics.Plainbuff, err = crypt.DecryptPKCS1v15(c.PrivateKey, cryptMetrics.Cryptbuff)
				if err != nil {
					return nil, status.Errorf(codes.Internal, "can not decrypt send data: %v", err)
				}
				return handler(ctx, req)
			}
		}
		return handler(ctx, req)
	}
}
