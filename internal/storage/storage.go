package storage

import (
	"fmt"
	"sync"
)

type Gauge float64
type Counter int64

type MemoryStorage struct {
	sync.Mutex
	Gauges   map[string]Gauge
	Counters map[string]Counter
}


// CRUD DB operations analog in memory
// create Memory interface{}
type MemoryStorager interface {
	AddNewCounter(key string, value Counter)
	GetAllCounters() map[string]Counter
	GetAllGauges() map[string]Gauge
	GetCounterByKey(key string) (Counter, error)
	GetGaugeByKey(key string) (Gauge, error)
	UpdateGauge(key string, value Gauge)
}

func (st *MemoryStorage) AddNewCounter(key string, counter Counter) {
	st.Lock()
	st.Counters[key] += counter
	st.Unlock()
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


// ////////////////////////////////
// Agent storage
// type Gauge float64
// type Counter int64

type Metrics struct {
	RuntimeMetrics map[string]Gauge
	PollCount      map[string]Counter
}

func InitMetricsStorage() Metrics {
	metStor := Metrics{RuntimeMetrics: make(map[string]Gauge), PollCount: make(map[string]Counter)}
	return metStor
}


//////////////////////
//handler template/html
type Pagecontent struct {
	AllMetrics []Metric
}

type Metric struct {
  Name  string
  Value string
}
