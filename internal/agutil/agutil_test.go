package agutil

import (
	"metrics-service/internal/storage"
	"os"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// func TestInitConfig1(t *testing.T) {
// 	var cfg Config

// 	os.Setenv("ADDRESS", "192.168.30.1:7777")
// 	InitConfig(&cfg)
// 	assert.Equal(t, "192.168.30.1:7777", cfg.Address, "test #")
	
// 	os.Unsetenv("ADDRESS")
// }

func TestInitConfig2(t *testing.T) {
	var cfg Config

	os.Setenv("ADDRESS", "")
	InitConfig(&cfg)
	assert.Equal(t, "localhost:8080", cfg.Address, "test ##")
	
	os.Unsetenv("ADDRESS")
}

func TestSetMetrics(t *testing.T) {
	st := storage.InitMetricsStorage()
	var mu sync.Mutex

	SetMetrics(&st, &mu)

	_, ok := st.RuntimeMetrics["Alloc"]
	require.True(t, ok)
	_, ok = st.RuntimeMetrics["BuckHashSys"]
	require.True(t, ok)
	_, ok = st.RuntimeMetrics["Frees"]
	require.True(t, ok)
	_, ok = st.RuntimeMetrics["GCCPUFraction"]
	require.True(t, ok)
	_, ok = st.RuntimeMetrics["GCSys"]
	require.True(t, ok)
	_, ok = st.RuntimeMetrics["HeapAlloc"]
	require.True(t, ok)
	_, ok = st.RuntimeMetrics["HeapIdle"]
	require.True(t, ok)
	_, ok = st.RuntimeMetrics["HeapInuse"]
	require.True(t, ok)
	_, ok = st.RuntimeMetrics["HeapObjects"]
	require.True(t, ok)
	_, ok = st.RuntimeMetrics["HeapReleased"]
	require.True(t, ok)
	_, ok = st.RuntimeMetrics["HeapSys"]
	require.True(t, ok)
	_, ok = st.RuntimeMetrics["LastGC"]
	require.True(t, ok)
	_, ok = st.RuntimeMetrics["Lookups"]
	require.True(t, ok)
	_, ok = st.RuntimeMetrics["MCacheInuse"]
	require.True(t, ok)
	_, ok = st.RuntimeMetrics["MCacheSys"]
	require.True(t, ok)
	_, ok = st.RuntimeMetrics["MSpanInuse"]
	require.True(t, ok)
	_, ok = st.RuntimeMetrics["MSpanSys"]
	require.True(t, ok)
	_, ok = st.RuntimeMetrics["Mallocs"]
	require.True(t, ok)
	_, ok = st.RuntimeMetrics["NextGC"]
	require.True(t, ok)
	_, ok = st.RuntimeMetrics["NumForcedGC"]
	require.True(t, ok)
	_, ok = st.RuntimeMetrics["NumGC"]
	require.True(t, ok)
	_, ok = st.RuntimeMetrics["OtherSys"]
	require.True(t, ok)
	_, ok = st.RuntimeMetrics["PauseTotalNs"]
	require.True(t, ok)
	_, ok = st.RuntimeMetrics["StackInuse"]
	require.True(t, ok)
	_, ok = st.RuntimeMetrics["StackSys"]
	require.True(t, ok)
	_, ok = st.RuntimeMetrics["Sys"]
	require.True(t, ok)
	_, ok = st.RuntimeMetrics["TotalAlloc"]
	require.True(t, ok)
	_, ok = st.RuntimeMetrics["RandomValue"]
	require.True(t, ok)
	_, ok = st.PollCount["PollCount"]
	require.True(t, ok)

	got := st.PollCount["PollCount"]
	var want storage.Counter = 1
	//or
	if got != want {
		t.Errorf("got %q, wanted %q", got, want)
	}

	assert.Equal(t, got, want)

	assert.NotEmpty(t, got)
}
