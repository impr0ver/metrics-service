package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/impr0ver/metrics-service/internal/logger"
	"github.com/impr0ver/metrics-service/internal/servconfig"

	"github.com/jackc/pgx"
	_ "github.com/jackc/pgx/v5/stdlib"
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

type DBStorage struct {
	memStor *FileStorage
	DB      *sql.DB
}

func (d *DBStorage) AddNewCounter(k string, c Counter) {
	d.memStor.AddNewCounter(k, c)
}

func (d *DBStorage) GetAllCounters() map[string]Counter {
	counters := d.memStor.GetAllCounters()
	return counters
}

func (d *DBStorage) GetAllGauges() map[string]Gauge {
	gauges := d.memStor.GetAllGauges()
	return gauges
}

func (d *DBStorage) GetCounterByKey(k string) (Counter, error) {
	counter, err := d.memStor.GetCounterByKey(k)
	return counter, err
}

func (d *DBStorage) GetGaugeByKey(k string) (Gauge, error) {
	gauge, err := d.memStor.GetGaugeByKey(k)
	return gauge, err
}

func (d *DBStorage) UpdateGauge(k string, v Gauge) {
	d.memStor.UpdateGauge(k, v)
}

func (d *DBStorage) DBPing(ctx context.Context) error {
	err := d.DB.PingContext(ctx)
	return err
}

func NewMemoryStorage(ctx context.Context, cfg *servconfig.Config) MemoryStoragerInterface {
	var sLogger = logger.NewLogger()
	var memStor MemoryStoragerInterface

	memStor = &MemoryStorage{Gauges: make(map[string]Gauge), Counters: make(map[string]Counter)}

	if cfg.Restore {
		err := RestoreFromFile(memStor, cfg.StoreFile)
		if err != nil {
			sLogger.Infof("Warning: %v\n", err)
		}
	}

	if cfg.StoreFile != "" {
		if cfg.StoreInterval > 0 {
			RunStoreToFileRoutine(ctx, memStor, cfg.StoreFile, cfg.StoreInterval)
		} else { //Sync
			memStor = &FileStorage{MemoryStoragerInterface: memStor, FilePath: cfg.StoreFile}
		}
	}

	if cfg.DatabaseDSN != "" {
		db, err := ConnectDB(cfg.DatabaseDSN)
		if err != nil {
			sLogger.Fatalf("error DB: %v", err)
		}
		inMemStor := &FileStorage{MemoryStoragerInterface: memStor, FilePath: cfg.StoreFile}
		memStor = &DBStorage{DB: db.DB, memStor: inMemStor}
	}

	return memStor
}

// add StoreToFile with AddNewCounter - sync mode if i set 0
func (s *FileStorage) AddNewCounter(k string, c Counter) {
	var sLogger = logger.NewLogger()
	s.MemoryStoragerInterface.AddNewCounter(k, c)
	err := StoreToFile(s, s.FilePath)
	if err != nil {
		sLogger.Errorf("error to save data in file: %v", err)
	}
}

// add StoreToFile with UpdateGauge - sync mode if i set 0
func (s *FileStorage) UpdateGauge(k string, g Gauge) {
	var sLogger = logger.NewLogger()
	s.MemoryStoragerInterface.UpdateGauge(k, g)
	err := StoreToFile(s, s.FilePath)
	if err != nil {
		sLogger.Errorf("error to save data in file: %v", err)
	}
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
	AddNewCounter(key string, value Counter)
	GetAllCounters() map[string]Counter
	GetAllGauges() map[string]Gauge
	GetCounterByKey(key string) (Counter, error)
	GetGaugeByKey(key string) (Gauge, error)
	UpdateGauge(key string, value Gauge)
	DBPing(ctx context.Context) error
}

func (st *MemoryStorage) DBPing(ctx context.Context) error {
	return errors.New("method is not implemented")
}

func (st *MemoryStorage) AddNewCounter(key string, counter Counter) {
	if counter != 0 {
		st.Lock()
		st.Counters[key] += counter
		st.Unlock()
	}
}

func (st *MemoryStorage) GetAllCounters() map[string]Counter {
	st.Lock()
	defer st.Unlock()

	res := make(map[string]Counter, len(st.Counters))
	for k, v := range st.Counters {
		res[k] = v
	}
	return res
}

func (st *MemoryStorage) GetAllGauges() map[string]Gauge {
	st.Lock()
	defer st.Unlock()

	res := make(map[string]Gauge, len(st.Gauges))
	for k, v := range st.Gauges {
		res[k] = v
	}
	return res
}

func (st *MemoryStorage) GetCounterByKey(key string) (Counter, error) {
	st.Lock()
	counter, ok := st.Counters[key]
	st.Unlock()
	if !ok {
		return Counter(0), fmt.Errorf("counter %s not found in the storage", key)
	}
	return counter, nil
}

func (st *MemoryStorage) GetGaugeByKey(key string) (Gauge, error) {
	st.Lock()
	gauge, ok := st.Gauges[key]
	st.Unlock()
	if !ok {
		return Gauge(0), fmt.Errorf("gauge %s not found in the storage", key)
	}
	return gauge, nil
}

func (st *MemoryStorage) UpdateGauge(key string, value Gauge) {
	st.Lock()
	defer st.Unlock()
	st.Gauges[key] = value
}

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

//DB connect
func ConnectDB(dsn string) (*DBStorage, error) {
	dbs := &DBStorage{}

	if err := checkDSN(dsn); err != nil {
		return dbs, err
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}
	dbs.DB = db
	return dbs, nil
}

func checkDSN(dsn string) (err error) {
	_, err = pgx.ParseDSN(dsn)
	return err
}
