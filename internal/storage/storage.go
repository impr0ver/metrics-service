package storage

import (
	"fmt"
	"sync"
)

// Server storage
type Memory struct {
	Gauges   map[string]float64
	Counters map[string]int64
}

func InitMemory() Memory {
	memory := Memory{Gauges: make(map[string]float64), Counters: make(map[string]int64)}
	return memory
}

// CRUD DB operations analog in memory
// create Memory interface{}
type MemoryStorager interface {
	AddCounter(source string, key string, countValue Counter)
	GetGaugeByKey(source string, key string) (Gauge, error)
	GetCounterByKey(source string, key string) (Counter, error)
	UpdateGauge(source string, key string, gaugeValue Gauge)
}

type MemoryStorage struct {
	sync.Mutex
	MemoryMap map[string]Memory
}

func (st *MemoryStorage) AddCounter(source string, key string, countValue Counter) {
	st.Lock()
	defer st.Unlock()
	gp, ok := st.MemoryMap[source]
	if !ok {
		gp := InitMemory()
		existValue := gp.Counters[key]
		gp.Counters[key] = existValue + int64(countValue)
		st.MemoryMap[source] = gp
		return
	}
	existValue := gp.Counters[key]
	gp.Counters[key] = existValue + int64(countValue)
	st.MemoryMap[source] = gp
}

func (st *MemoryStorage) UpdateGauge(source string, key string, gaugeValue Gauge) {
	st.Lock()
	defer st.Unlock()
	gp, ok := st.MemoryMap[source]
	if !ok {
		gp := InitMemory()
		gp.Gauges[key] = float64(gaugeValue)
		st.MemoryMap[source] = gp
		return
	}
	gp.Gauges[key] = float64(gaugeValue)
	st.MemoryMap[source] = gp
}

func (st *MemoryStorage) GetGaugeByKey(source string, key string) (Gauge, error) {
	gp, ok := st.MemoryMap[source]
	if !ok {
		return Gauge(0), fmt.Errorf("source '%s' not found in the storage", source)
	}
	gauge, ok := gp.Gauges[key]
	if !ok {
		return Gauge(0), fmt.Errorf("gauge '%s' not found in the storage", key)
	}
	return Gauge(gauge), nil
}

func (st *MemoryStorage) GetCounterByKey(source string, key string) (Counter, error) {
	gp, ok := st.MemoryMap[source]
	if !ok {
		return Counter(0), fmt.Errorf("source %s not found in the storage", source)
	}
	counter, ok := gp.Counters[key]
	if !ok {
		return Counter(0), fmt.Errorf("counter %s not found in the storage", key)
	}
	return Counter(counter), nil
}

// ////////////////////////////////
// Agent storage
type Gauge float64
type Counter int64

type Metrics struct {
	RuntimeMetrics map[string]Gauge
	PollCount      map[string]Counter
}

func InitMetricsStorage() Metrics {
	metStor := Metrics{RuntimeMetrics: make(map[string]Gauge), PollCount: make(map[string]Counter)}
	return metStor
}
