package storage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/impr0ver/metrics-service/internal/logger"
	"github.com/impr0ver/metrics-service/internal/servconfig"
)

type Gauge float64
type Counter int64

type MemoryStorage struct {
	sync.Mutex
	Gauges   map[string]Gauge
	Counters map[string]Counter
}

type FileStorage struct {
	MemoryStoragerInterface
	FilePath string `json:"-"`
}


func NewMemoryStorage(ctx context.Context, cfg *servconfig.Config) MemoryStoragerInterface {
	var sLogger = logger.NewLogger()
	var memStor MemoryStoragerInterface

	if cfg.DatabaseDSN != "" { //Init memory as DB
		db, err := ConnectDB(cfg.DatabaseDSN)
		if err != nil {
			sLogger.Fatalf("error DB: %v", err)
		}
		memStor = &DBStorage{DB: db.DB}
		cfg.StoreFile = ""
		cfg.Restore = false
	} else {	//Init memory as struct in memory 
		memStor = &MemoryStorage{Gauges: make(map[string]Gauge), Counters: make(map[string]Counter)}

		if cfg.StoreFile != "" {//Init memory as file storage and struct
			if cfg.StoreInterval > 0 {
				RunStoreToFileRoutine(ctx, memStor, cfg.StoreFile, cfg.StoreInterval)
			} else { 
				memStor = &FileStorage{MemoryStoragerInterface: memStor, FilePath: cfg.StoreFile}
			}
		}
		if cfg.Restore {
			err := RestoreFromFile(memStor, cfg.StoreFile)
			if err != nil {
				sLogger.Infof("Warning: %v\n", err)
			}
		}
	}
	return memStor
}

// add StoreToFile with AddNewCounter - sync mode if i set 0
func (s *FileStorage) AddNewCounter(ctx context.Context, k string, c Counter) error {
	var sLogger = logger.NewLogger()
	s.MemoryStoragerInterface.AddNewCounter(ctx, k, c)
	err := StoreToFile(s, s.FilePath)
	if err != nil {
		sLogger.Errorf("error to save data in file: %v", err)
		return err
	}
	return nil
}

// add StoreToFile with UpdateGauge - sync mode if i set 0
func (s *FileStorage) UpdateGauge(ctx context.Context, k string, g Gauge) error {
	var sLogger = logger.NewLogger()
	s.MemoryStoragerInterface.UpdateGauge(ctx, k, g)
	err := StoreToFile(s, s.FilePath)
	if err != nil {
		sLogger.Errorf("error to save data in file: %v", err)
		return err
	}
	return nil
}

type Metrics struct {
	ID    string   `json:"id"`              // metric Name
	MType string   `json:"type"`            // Type gauge or counter
	Delta *int64   `json:"delta,omitempty"` // pointer on CountValue (pointer need for check on nil)
	Value *float64 `json:"value,omitempty"` // pointer on GaugeValue (pointer need for check on nil)
}

// Analog CRUD DB operations in memory
// create Memory interface{}
type MemoryStoragerInterface interface {
	AddNewCounter(ctx context.Context, key string, value Counter) error
	GetAllCounters(ctx context.Context) (map[string]Counter, error)
	GetAllGauges(ctx context.Context) (map[string]Gauge, error)
	GetCounterByKey(ctx context.Context, key string) (Counter, error)
	GetGaugeByKey(ctx context.Context, key string) (Gauge, error)
	UpdateGauge(ctx context.Context, key string, value Gauge) error
	DBPing(ctx context.Context) error
	AddNewMetricsAsBatch(ctx context.Context, metrics []Metrics) error
}

func (st *MemoryStorage) DBPing(ctx context.Context) error {
	return errors.New("method is not implemented")
}

func (st *MemoryStorage) AddNewCounter(ctx context.Context, key string, counter Counter) error {
	if counter != 0 {
		st.Lock()
		st.Counters[key] += counter
		st.Unlock()
	}
	return nil
}

func (st *MemoryStorage) GetAllCounters(ctx context.Context) (map[string]Counter, error) {
	st.Lock()
	defer st.Unlock()

	res := make(map[string]Counter, len(st.Counters))
	for k, v := range st.Counters {
		res[k] = v
	}
	return res, nil
}

func (st *MemoryStorage) GetAllGauges(ctx context.Context) (map[string]Gauge, error) {
	st.Lock()
	defer st.Unlock()

	res := make(map[string]Gauge, len(st.Gauges))
	for k, v := range st.Gauges {
		res[k] = v
	}
	return res, nil
}

func (st *MemoryStorage) GetCounterByKey(ctx context.Context, key string) (Counter, error) {
	st.Lock()
	counter, ok := st.Counters[key]
	st.Unlock()
	if !ok {
		return Counter(0), fmt.Errorf("counter %s not found in the storage", key)
	}
	return counter, nil
}

func (st *MemoryStorage) GetGaugeByKey(ctx context.Context, key string) (Gauge, error) {
	st.Lock()
	gauge, ok := st.Gauges[key]
	st.Unlock()
	if !ok {
		return Gauge(0), fmt.Errorf("gauge %s not found in the storage", key)
	}
	return gauge, nil
}

func (st *MemoryStorage) UpdateGauge(ctx context.Context, key string, value Gauge) error {
	st.Lock()
	defer st.Unlock()
	st.Gauges[key] = value

	return nil
}

///
func (st *MemoryStorage) AddNewMetricsAsBatch(ctx context.Context, metrics []Metrics) error {
	for _, metric := range metrics {
		switch metric.MType {
		case "counter":
			err := st.AddNewCounter(ctx, metric.ID, Counter(*metric.Delta))
			if err != nil {
				return err
			}
		case "gauge":
			err := st.UpdateGauge(ctx, metric.ID, Gauge(*metric.Value))
			if err != nil {
				return err
			}
		default:
			return fmt.Errorf("unsupported metric type")
		}
	}
	return nil
}

///

// operations with file (store data in file)
func RestoreFromFile(memStor MemoryStoragerInterface, filePath string) error {
	fm, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer fm.Close()
	return json.NewDecoder(fm).Decode(&memStor)
}

func StoreToFile(memStor MemoryStoragerInterface, filePath string) error {
	fm, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer fm.Close()
	return json.NewEncoder(fm).Encode(&memStor)
}

func RunStoreToFileRoutine(ctx context.Context, memStor MemoryStoragerInterface, filePath string, storeInterval time.Duration) {
	var sLogger = logger.NewLogger()

	go func() {
		tickerStoreToFile := time.NewTicker(storeInterval)
		defer tickerStoreToFile.Stop()
		for {
			select {
			case t := <-tickerStoreToFile.C:
				sLogger.Infoln("Write data to file at", t.Format("15:04:05"))
				StoreToFile(memStor, filePath)

			case <-ctx.Done():
				return
			}
		}
	}()
}

// handler template/html storage
type Pagecontent struct {
	AllMetrics []Metric
}

type Metric struct {
	Name  string
	Value string
}
