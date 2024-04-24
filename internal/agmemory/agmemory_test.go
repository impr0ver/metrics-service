package agmemory_test

import (
	"sync"
	"testing"

	"github.com/impr0ver/metrics-service/internal/agmemory"
	"github.com/impr0ver/metrics-service/internal/agwork"
	"github.com/stretchr/testify/require"
)

func TestSetGopsMetrics(t *testing.T) {
	st := agmemory.NewAgMemory()
	var mu sync.RWMutex

	err := agwork.SetGopsMetrics(&st, &mu)
	require.NoError(t, err)
	_, ok := st.RuntimeMetrics["CPUutilization1"]
	require.True(t, ok)
	_, ok = st.RuntimeMetrics["TotalMemory"]
	require.True(t, ok)
	_, ok = st.RuntimeMetrics["FreeMemory"]
	require.True(t, ok)
}
