// Storage package contains the declaration and implementation of the MemoryStoragerInterface, which is an abstract storage of metrics.
// The package contains two implementations of the interface: MemoryStorage - a storage organized in RAM (map data type) and DBStorage - a storage that uses a db driver (postgres)
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

type (
	Gauge   float64
	Counter int64

	MemoryStorage struct {
		sync.Mutex
		Gauges   map[string]Gauge
		Counters map[string]Counter
	}

	FileStorage struct {
		MemoryStoragerInterface
		FilePath string `json:"-"`
	}

	// Metrics struct JSON-form
	Metrics struct {
		ID    string   `json:"id"`              // metric Name
		MType string   `json:"type"`            // type gauge or counter
		Delta *int64   `json:"delta,omitempty"` // pointer on CountValue (pointer need for check on nil)
		Value *float64 `json:"value,omitempty"` // pointer on GaugeValue (pointer need for check on nil)
	}

	// MemoryStoragerInterface. It create Memory interface{}
	MemoryStoragerInterface interface {
		AddNewCounter(ctx context.Context, key string, value Counter) error
		GetAllCounters(ctx context.Context) (map[string]Counter, error)
		GetAllGauges(ctx context.Context) (map[string]Gauge, error)
		GetCounterByKey(ctx context.Context, key string) (Counter, error)
		GetGaugeByKey(ctx context.Context, key string) (Gauge, error)
		UpdateGauge(ctx context.Context, key string, value Gauge) error
		DBPing(ctx context.Context) error
		AddNewMetricsAsBatch(ctx context.Context, metrics []Metrics) error
	}

	// Pagecontent for template/html storage.
	Pagecontent struct {
		AllMetrics []Metric
	}

	// Metric for template/html storage.
	Metric struct {
		Name  string
		Value string
	}
)

// NewStorage initialize storage and return MemoryStoragerInterface.
func NewStorage(ctx context.Context, cfg *servconfig.Config) MemoryStoragerInterface {
	var sLogger = logger.NewLogger()
	var memStor MemoryStoragerInterface

	if cfg.DatabaseDSN != "" { // init memory as DB
		ctxTimeOut, cancel := context.WithTimeout(context.Background(), cfg.DefaultCtxTimeout)
		defer cancel()

		db, err := ConnectDB(ctxTimeOut, cfg.DatabaseDSN)
		if err != nil {
			sLogger.Fatalf("error DB: %v", err)
		}
		memStor = &DBStorage{DB: db.DB}

	} else { // init memory as struct in memory
		memStor = &MemoryStorage{Gauges: make(map[string]Gauge), Counters: make(map[string]Counter)}

		if cfg.StoreFile != "" { // init memory as file storage and struct
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

// AddNewCounter add StoreToFile. Sync mode if i set 0
func (s *FileStorage) AddNewCounter(ctx context.Context, k string, c Counter) error {
	var sLogger = logger.NewLogger()
	s.MemoryStoragerInterface.AddNewCounter(ctx, k, c)
	err := StoreToFile(s, s.FilePath)
	if err != nil {
		sLogger.Errorf("error to save data in file: %w", err)
		return err
	}
	return nil
}

// UpdateGauge add StoreToFile. Sync mode if i set 0
func (s *FileStorage) UpdateGauge(ctx context.Context, k string, g Gauge) error {
	var sLogger = logger.NewLogger()
	s.MemoryStoragerInterface.UpdateGauge(ctx, k, g)
	err := StoreToFile(s, s.FilePath)
	if err != nil {
		sLogger.Errorf("error to save data in file: %w", err)
		return err
	}
	return nil
}

// DBPing - method stub (storage in memory).
func (st *MemoryStorage) DBPing(ctx context.Context) error {
	return errors.New("method is not implemented")
}

// AddNewCounter - add new counter (storage in memory).
func (st *MemoryStorage) AddNewCounter(ctx context.Context, key string, counter Counter) error {
	if counter != 0 {
		st.Lock()
		st.Counters[key] += counter
		st.Unlock()
	}
	return nil
}

// GetAllCounters - get all counters (storage in memory).
func (st *MemoryStorage) GetAllCounters(ctx context.Context) (map[string]Counter, error) {
	st.Lock()
	defer st.Unlock()

	res := make(map[string]Counter, len(st.Counters))
	for k, v := range st.Counters {
		res[k] = v
	}
	return res, nil
}

// GetAllGauges - get all gauges (storage in memory).
func (st *MemoryStorage) GetAllGauges(ctx context.Context) (map[string]Gauge, error) {
	st.Lock()
	defer st.Unlock()

	res := make(map[string]Gauge, len(st.Gauges))
	for k, v := range st.Gauges {
		res[k] = v
	}
	return res, nil
}

// GetCounterByKey - get counter value by key (storage in memory).
func (st *MemoryStorage) GetCounterByKey(ctx context.Context, key string) (Counter, error) {
	st.Lock()
	counter, ok := st.Counters[key]
	st.Unlock()
	if !ok {
		return Counter(0), fmt.Errorf("counter %s not found in the storage", key)
	}
	return counter, nil
}

// GetGaugeByKey - get gauge value by key (storage in memory).
func (st *MemoryStorage) GetGaugeByKey(ctx context.Context, key string) (Gauge, error) {
	st.Lock()
	gauge, ok := st.Gauges[key]
	st.Unlock()
	if !ok {
		return Gauge(0), fmt.Errorf("gauge %s not found in the storage", key)
	}
	return gauge, nil
}

// UpdateGauge - update gauge value (storage in memory).
func (st *MemoryStorage) UpdateGauge(ctx context.Context, key string, value Gauge) error {
	st.Lock()
	defer st.Unlock()
	st.Gauges[key] = value

	return nil
}

// AddNewMetricsAsBatch add or update metrics (storage in memory).
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

// RestoreFromFile read from file and JSON-decode data in storage.
func RestoreFromFile(memStor MemoryStoragerInterface, filePath string) error {
	fm, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer fm.Close()
	return json.NewDecoder(fm).Decode(&memStor)
}

// StoreToFile write in file JSON-encode data from storage.
func StoreToFile(memStor MemoryStoragerInterface, filePath string) error {
	fm, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer fm.Close()
	return json.NewEncoder(fm).Encode(&memStor)
}

// RunStoreToFileRoutine routine what write data in file.
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
