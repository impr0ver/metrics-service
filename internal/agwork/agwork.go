package agwork

import (
	"fmt"
	"math/rand"
	"net/http"
	"runtime"
	"sync"
	"time"

	"github.com/impr0ver/metrics-service/internal/agmemory"
)

func SetMetrics(metrics *agmemory.AgMemory, mu *sync.Mutex) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	var rtm runtime.MemStats
	runtime.ReadMemStats(&rtm)

	mu.Lock()
	defer mu.Unlock()

	metrics.RuntimeMetrics["Alloc"] = agmemory.Gauge(rtm.Alloc)
	metrics.RuntimeMetrics["BuckHashSys"] = agmemory.Gauge(rtm.BuckHashSys)
	metrics.RuntimeMetrics["BuckHashSys"] = agmemory.Gauge(rtm.BuckHashSys)
	metrics.RuntimeMetrics["Frees"] = agmemory.Gauge(rtm.Frees)
	metrics.RuntimeMetrics["GCCPUFraction"] = agmemory.Gauge(rtm.GCCPUFraction)
	metrics.RuntimeMetrics["GCSys"] = agmemory.Gauge(rtm.HeapAlloc)
	metrics.RuntimeMetrics["HeapAlloc"] = agmemory.Gauge(rtm.HeapAlloc)
	metrics.RuntimeMetrics["HeapIdle"] = agmemory.Gauge(rtm.HeapIdle)
	metrics.RuntimeMetrics["HeapInuse"] = agmemory.Gauge(rtm.HeapInuse)
	metrics.RuntimeMetrics["HeapObjects"] = agmemory.Gauge(rtm.HeapObjects)
	metrics.RuntimeMetrics["HeapReleased"] = agmemory.Gauge(rtm.HeapReleased)
	metrics.RuntimeMetrics["HeapSys"] = agmemory.Gauge(rtm.HeapSys)
	metrics.RuntimeMetrics["LastGC"] = agmemory.Gauge(rtm.LastGC)
	metrics.RuntimeMetrics["Lookups"] = agmemory.Gauge(rtm.Lookups)
	metrics.RuntimeMetrics["MCacheInuse"] = agmemory.Gauge(rtm.MCacheInuse)
	metrics.RuntimeMetrics["MCacheSys"] = agmemory.Gauge(rtm.MCacheSys)
	metrics.RuntimeMetrics["MSpanInuse"] = agmemory.Gauge(rtm.MSpanInuse)
	metrics.RuntimeMetrics["MSpanSys"] = agmemory.Gauge(rtm.MSpanSys)
	metrics.RuntimeMetrics["Mallocs"] = agmemory.Gauge(rtm.Mallocs)
	metrics.RuntimeMetrics["NextGC"] = agmemory.Gauge(rtm.NextGC)
	metrics.RuntimeMetrics["NumForcedGC"] = agmemory.Gauge(rtm.NumForcedGC)
	metrics.RuntimeMetrics["NumGC"] = agmemory.Gauge(rtm.NumGC)
	metrics.RuntimeMetrics["OtherSys"] = agmemory.Gauge(rtm.OtherSys)
	metrics.RuntimeMetrics["PauseTotalNs"] = agmemory.Gauge(rtm.PauseTotalNs)
	metrics.RuntimeMetrics["StackInuse"] = agmemory.Gauge(rtm.StackInuse)
	metrics.RuntimeMetrics["StackSys"] = agmemory.Gauge(rtm.StackSys)
	metrics.RuntimeMetrics["Sys"] = agmemory.Gauge(rtm.Sys)
	metrics.RuntimeMetrics["TotalAlloc"] = agmemory.Gauge(rtm.TotalAlloc)
	metrics.RuntimeMetrics["RandomValue"] = agmemory.Gauge(r.Float64())

	metrics.PollCount["PollCount"]++
}

func InitMetrics(mu *sync.Mutex, memory *agmemory.AgMemory, pollInterval int) {

	SetMetrics(memory, mu)

}

func SendMetrics(mu *sync.Mutex, memory *agmemory.AgMemory, reportInterval int, URL string) {
	mu.Lock()
	defer mu.Unlock()

	metricData := memory.RuntimeMetrics
	pollCount := memory.PollCount

	for key, value := range metricData {

		fullGaugeURL := fmt.Sprintf("http://%s/update/gauge/%s/%.2f", URL, key, value)
		resp, err := http.Post(fullGaugeURL, "text/plain", nil)
		if err != nil {
			fmt.Println(err)
			return
		}
		resp.Body.Close()
	}
	fullCountURL := fmt.Sprintf("http://%s/update/counter/PollCount/%d", URL, pollCount["PollCount"])
	resp, err := http.Post(fullCountURL, "text/plain", nil)
	if err != nil {
		fmt.Println(err)
		return
	}
	resp.Body.Close()
}
