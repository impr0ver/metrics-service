package handlers_test

import (
	"compress/gzip"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/impr0ver/metrics-service/internal/handlers"
	"github.com/impr0ver/metrics-service/internal/servconfig"
	"github.com/impr0ver/metrics-service/internal/storage"
	"github.com/stretchr/testify/suite"
)

type Metrics struct {
	ID    string   `json:"id"`
	MType string   `json:"type"`
	Delta *int64   `json:"delta,omitempty"` //countValue
	Value *float64 `json:"value,omitempty"` //gaugeValue
}

var memstorage = storage.MemoryStorage{Gauges: make(map[string]storage.Gauge),
	Counters: make(map[string]storage.Counter)}

var cfg = servconfig.Config{}

func ExampleMetricsHandlerGetJSON() {
	var metric Metrics

	memstorage.UpdateGauge(context.TODO(), "Sys", 234.432)

	r := handlers.ChiRouter(&memstorage, &cfg)

	mbytes, _ := json.Marshal(Metrics{ID: "Sys", MType: "gauge"})
	bodyReader := strings.NewReader(string(mbytes))

	request := httptest.NewRequest(http.MethodPost, "/value/", bodyReader)
	request.Header.Set("Content-Type", "application/json; charset=UTF-8")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, request)

	res := w.Result()
	defer res.Body.Close()

	err := json.NewDecoder(res.Body).Decode(&metric)
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println("Status code:", res.StatusCode)
	fmt.Println("Metric value:", *metric.Value)

	//Output:
	//Status code: 200
	//Metric value: 234.432
}

func ExampleMetricsHandlerGetJSON_second() {
	var metric Metrics

	memstorage.AddNewCounter(context.TODO(), "MyCounter", 555)

	r := handlers.ChiRouter(&memstorage, &cfg)

	mbytes, _ := json.Marshal(Metrics{ID: "MyCounter", MType: "counter"})
	bodyReader := strings.NewReader(string(mbytes))

	request := httptest.NewRequest(http.MethodPost, "/value/", bodyReader)
	request.Header.Set("Content-Type", "application/json; charset=UTF-8")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, request)

	res := w.Result()
	defer res.Body.Close()

	err := json.NewDecoder(res.Body).Decode(&metric)
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println("Status code:", res.StatusCode)
	fmt.Println("Metric value:", *metric.Delta)

	//Output:
	//Status code: 200
	//Metric value: 555
}

func ExampleMetricsHandlerGetJSON_third() {
	var metric Metrics

	r := handlers.ChiRouter(&memstorage, &cfg)

	mbytes, _ := json.Marshal(Metrics{ID: "NoExistsValue", MType: "counter"})
	bodyReader := strings.NewReader(string(mbytes))

	request := httptest.NewRequest(http.MethodPost, "/value/", bodyReader)
	request.Header.Set("Content-Type", "application/json; charset=UTF-8")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, request)

	res := w.Result()
	defer res.Body.Close()

	err := json.NewDecoder(res.Body).Decode(&metric)
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println("Status code:", res.StatusCode)
	if metric.Value == nil {
		fmt.Println("Metric value: nil")
	}

	//Output:
	//Status code: 404
	//Metric value: nil
}

func ExampleMetricsHandlerPostJSON_second() {
	var metric Metrics
	var countVal int64 = 666

	r := handlers.ChiRouter(&memstorage, &cfg)

	mbytes, _ := json.Marshal(Metrics{ID: "MyCount", MType: "counter", Delta: (*int64)(&countVal)})
	bodyReader := strings.NewReader(string(mbytes))

	request := httptest.NewRequest(http.MethodPost, "/update/", bodyReader)
	request.Header.Set("Content-Type", "application/json; charset=UTF-8")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, request)

	res := w.Result()
	defer res.Body.Close()

	err := json.NewDecoder(res.Body).Decode(&metric)
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println("Status code:", res.StatusCode)
	fmt.Println("Metric value:", *metric.Delta)

	//Output:
	//Status code: 200
	//Metric value: 666
}

func ExampleMetricsHandlerGet() {
	//set some metrics in our storage for unit-test
	memstorage.AddNewCounter(context.TODO(), "TstCounter", storage.Counter(345))
	memstorage.AddNewCounter(context.TODO(), "TstCounter", storage.Counter(200))

	r := handlers.ChiRouter(&memstorage, &cfg)

	request := httptest.NewRequest(http.MethodGet, "/value/counter/TstCounter", nil)
	request.Header.Set("Content-Type", "text/plain")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, request)

	res := w.Result()
	defer res.Body.Close()

	respBody, err := io.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println("Status code:", res.StatusCode)
	fmt.Println("Metric value:", string(respBody))

	//Output:
	//Status code: 200
	//Metric value: 545
}

func ExampleMetricsHandlerGetAll() {
	//set some metrics in our storage for unit-test
	memstorage.AddNewCounter(context.TODO(), "MyTstCounter", storage.Counter(100))
	memstorage.AddNewCounter(context.TODO(), "MyTstCounter", storage.Counter(566)) //must bee "666" = 100 + 566

	r := handlers.ChiRouter(&memstorage, &cfg)

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("Content-Type", "text/plain")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, request)

	res := w.Result()
	defer res.Body.Close()

	reader, err := gzip.NewReader(res.Body)
	if err != nil {
		fmt.Println(err)
	}
	defer reader.Close()

	result := make([]byte, 1000)

	reader.Read(result)

	fmt.Println("Status code:", res.StatusCode)
	fmt.Println("All metrics:", strings.Contains(string(result), "666"))

	//Output:
	//Status code: 200
	//All metrics: true
}

func ExampleMetricsHandlerPost() {

	r := handlers.ChiRouter(&memstorage, &cfg)

	request := httptest.NewRequest(http.MethodPost, "/update/gauge/OtherSys/5557552.000000", nil)
	request.Header.Set("Content-Type", "text/plain")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, request)

	res := w.Result()
	defer res.Body.Close()

	respBody, err := io.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println("Status code:", res.StatusCode)
	fmt.Println("Responce:", string(respBody))

	//Output:
	//reqMetrics gauge OtherSys 5557552.000000
	//Status code: 200
	//Responce: Registered successfully!
}

func ExampleMetricsHandlerPostBatch() {
	exampleJSON := `[{ "id": "MCacheSys", "type": "gauge", "value": 15600 },
  { "id": "StackInuse", "type": "gauge", "value": 327680 },
  { "id": "HeapInuse", "type": "gauge", "value": 811008 },
  { "id": "CPUutilization1", "type": "gauge", "value": 1.9801980198269442 },
  { "id": "StackSys", "type": "gauge", "value": 327680 },
  { "id": "GCSys", "type": "gauge", "value": 8055592 },
  { "id": "Alloc", "type": "gauge", "value": 308568 },
  { "id": "MCacheInuse", "type": "gauge", "value": 1200 }]`

	r := handlers.ChiRouter(&memstorage, &cfg)

	bodyReader := strings.NewReader(exampleJSON)

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

	fmt.Println("Status code:", res.StatusCode)
	fmt.Println("Responce:", string(respBody))

	//Output:
	//Status code: 200
	//Responce: Registered successfully!
}

type DBStorageTestSuite struct {
	suite.Suite
	DB      *storage.DBStorage
	TestDSN string
}

func (suite *DBStorageTestSuite) SetupSuite() {
	suite.DB = &storage.DBStorage{DB: nil}

	dsn := "postgresql://localhost:5432?user=postgres&password=postgres"
	dbname := "testdb"

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return
	}

	db.Exec("DROP DATABASE " + dbname)
	_, err = db.Exec("CREATE DATABASE " + dbname)
	db.Close()
	if err != nil {
		return
	}

	testDSN := "postgresql://localhost:5432/" + dbname + "?user=postgres&password=postgres"
	suite.DB, _ = storage.ConnectDB(context.TODO(), testDSN)
}

func ExampleDataBasePing() {
	var memStor storage.MemoryStoragerInterface
	dbStor := &DBStorageTestSuite{}

	dbStor.SetupSuite()
	memStor = dbStor.DB

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

	fmt.Println("Status code:", res.StatusCode)
	fmt.Println("Responce:", string(respBody))

	//Output:
	//Status code: 200
	//Responce: DB alive!
}
