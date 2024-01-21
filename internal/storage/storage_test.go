package storage_test

import (
	"bufio"
	"log"
	"os"
	"sync"
	"testing"

	"github.com/impr0ver/metrics-service/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemory(t *testing.T) {
	st := storage.MemoryStorage{Gauges: make(map[string]storage.Gauge), Counters: make(map[string]storage.Counter)}

	tests := []struct {
		testname string
		gauge    string
		value    storage.Gauge
	}{
		{"test#1", "gauge1", storage.Gauge(0.1)},
		{"test#2", "gauge2", storage.Gauge(1.12)},
		{"test#3", "gauge3", storage.Gauge(10452.1)},
		{"test#4", "gauge4", storage.Gauge(485.1)},
		{"test#5", "gauge5", storage.Gauge(222.00)},
		{"test#6", "gauge6", storage.Gauge(5.1)},
		{"test#7", "gauge7", storage.Gauge(40.23)},
		{"test#8", "gauge8", storage.Gauge(7.99)},
		{"test#9", "gauge9", storage.Gauge(67.9)},
		{"test#10", "gauge10", storage.Gauge(0.1)},
		{"test#11", "gauge11", storage.Gauge(10017.2)},
		{"test#12", "gauge12", storage.Gauge(1.1)},
		{"test#13", "gauge13", storage.Gauge(12.22)},
		{"test#14", "gauge14", storage.Gauge(214.2)},
		{"test#15", "gauge15", storage.Gauge(127.2)},
		{"test#16", "gauge16", storage.Gauge(16007.2)},
		{"test#17", "gauge17", storage.Gauge(217.2)},
		{"test#18", "gauge18", storage.Gauge(218.10)},
		{"test#19", "gauge19", storage.Gauge(56.8)},
		{"test#20", "gauge20", storage.Gauge(0.00002)},
		{"test#21", "gauge21", storage.Gauge(946.9)},
		{"test#22", "gauge22", storage.Gauge(32.3)},
		{"test#23", "gauge23", storage.Gauge(97.3)},
		{"test#24", "gauge24", storage.Gauge(4.3)},
		{"test#25", "gauge25", storage.Gauge(44.3)},
		{"test#26", "gauge26", storage.Gauge(33.53333)},
		{"test#27", "gauge27", storage.Gauge(317.2)},
		{"test#28", "gauge28", storage.Gauge(18.00000)},
		{"test#29", "gauge29", storage.Gauge(111.1)},
		{"test#30", "gauge30", storage.Gauge(20.5)},
		{"test#31", "gauge31", storage.Gauge(946.9)},
		{"test#32", "gauge32", storage.Gauge(32.3)},
		{"test#33", "gauge33", storage.Gauge(813.3)},
		{"test#34", "gauge34", storage.Gauge(4.3)},
		{"test#35", "gauge35", storage.Gauge(544.3)},
		{"test#36", "gauge36", storage.Gauge(6.33333333333)},
		{"test#37", "gauge37", storage.Gauge(317.2)},
		{"test#38", "gauge38", storage.Gauge(18.00000)},
		{"test#39", "gauge39", storage.Gauge(111.1)},
		{"test#40", "gauge40", storage.Gauge(20.5)},
	}
	var wg sync.WaitGroup
	wg.Add(4)

	go func(storage *storage.MemoryStorage) {
		for i := 0; i < 40; i++ {
			storage.UpdateGauge(tests[i].gauge, tests[i].value)
		}
		wg.Done()
	}(&st)

	go func() {
		for i := 0; i < 10; i++ {
			st.UpdateGauge(tests[i].gauge, tests[i].value)
		}
		wg.Done()
	}()

	go func(storage *storage.MemoryStorage) {
		for i := 10; i < 20; i++ {
			storage.UpdateGauge(tests[i].gauge, tests[i].value)
		}
		wg.Done()
	}(&st)

	go func() {
		for i := 20; i < 40; i++ {

			st.UpdateGauge(tests[i].gauge, tests[i].value)
		}
		wg.Done()
	}()

	wg.Wait()
	for _, tt := range tests {
		value, err := st.GetGaugeByKey(tt.gauge)
		require.NoError(t, err)
		assert.Equal(t, value, tt.value)
	}
}

func TestStoreToFile(t *testing.T) {
	st := storage.MemoryStorage{Gauges: make(map[string]storage.Gauge), Counters: make(map[string]storage.Counter)}

	st.UpdateGauge("key1", storage.Gauge(1.1))
	st.UpdateGauge("key2", storage.Gauge(2.22))
	st.UpdateGauge("key3", storage.Gauge(3.333))
	st.UpdateGauge("key4", storage.Gauge(4.4444))
	st.AddNewCounter("Counter1", storage.Counter(100))
	st.AddNewCounter("Counter2", storage.Counter(200))

	filePath := "./temp.json"
	err := storage.StoreToFile(&st, filePath)
	require.NoError(t, err)
	fm, err := os.Open(filePath)
	if err != nil {
		log.Fatal(err)
	}
	defer fm.Close()
	require.NoError(t, err)
	reader := bufio.NewReader(fm)
	line, _, _ := reader.ReadLine()
	expected := `{"Gauges":{"key1":1.1,"key2":2.22,"key3":3.333,"key4":4.4444},"Counters":{"Counter1":100,"Counter2":200}}`
	assert.Equal(t, expected, string(line))
	os.Remove(filePath)
}

func TestRestoreFromFile(t *testing.T) {
	st := storage.MemoryStorage{Gauges: make(map[string]storage.Gauge), Counters: make(map[string]storage.Counter)}

	st.UpdateGauge("key1", storage.Gauge(1.1))
	st.UpdateGauge("key2", storage.Gauge(2.22))
	st.UpdateGauge("key3", storage.Gauge(3.333))
	st.UpdateGauge("key4", storage.Gauge(4.4444))
	st.AddNewCounter("Counter1", storage.Counter(100))
	st.AddNewCounter("Counter2", storage.Counter(200))
	filePath := "./temp2.json"
	content := `{"Gauges":{"key1":1.1,"key2":2.22,"key3":3.333,"key4":4.4444},"Counters":{"Counter1":100,"Counter2":200}}`
	fm, err := os.Create(filePath)
	if err != nil {
		log.Fatal(err)
	}
	defer fm.Close()
	require.NoError(t, err)
	fm.Write([]byte(content))

	st2 := storage.MemoryStorage{Gauges: make(map[string]storage.Gauge), Counters: make(map[string]storage.Counter)}
	err = storage.RestoreFromFile(&st2, filePath)
	require.NoError(t, err)
	assert.Equal(t, st.Gauges, st2.Gauges)
	assert.Equal(t, st.Counters, st2.Counters)
	os.Remove(filePath)
}
