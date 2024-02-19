package agwork

import (
	"bytes"
	"errors"
	"strconv"
	"syscall"

	"fmt"
	"math/rand"
	"net/http"
	"runtime"
	"sync"
	"time"

	"github.com/impr0ver/metrics-service/internal/agmemory"
	"github.com/impr0ver/metrics-service/internal/crypt"
	"github.com/impr0ver/metrics-service/internal/gzip"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
)

type Semaphore struct {
	C chan struct{}
}

func (s *Semaphore) Acquire() {
	s.C <- struct{}{}
}

func (s *Semaphore) Release() {
	<-s.C
}

func SetRTMetrics(metrics *agmemory.AgMemory, mu *sync.RWMutex) {
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

func SetGopsMetrics(metrics *agmemory.AgMemory, mu *sync.RWMutex) error {
	key := "CPUutilization"

	percentage, err := cpu.Percent(time.Second*1, true)
	if err != nil {
		return err
	}

	mu.Lock()
	defer mu.Unlock()

	for i, p := range percentage {
		keyCPUCount := key + strconv.FormatInt(int64(i+1), 10)
		metrics.RuntimeMetrics[keyCPUCount] = agmemory.Gauge(p)
	}

	memory, err := mem.VirtualMemory()
	if err != nil {
		return err
	}
	metrics.RuntimeMetrics["TotalMemory"] = agmemory.Gauge(memory.Total)
	metrics.RuntimeMetrics["FreeMemory"] = agmemory.Gauge(memory.Free)

	return nil
}

func SendMetrics(mu *sync.Mutex, memory *agmemory.AgMemory, URL string, signKey string) {
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

func SendMetricsJSON(mu *sync.RWMutex, memory *agmemory.AgMemory, URL string, signKey string) {
	mu.RLock()
	defer mu.RUnlock()

	metricData := memory.RuntimeMetrics
	pollCount := memory.PollCount["PollCount"]

	fullURL := fmt.Sprintf("http://%s/update/", URL)
	var agMetrics agmemory.Metrics

	//prepare and send gauges
	for key, value := range metricData {
		agMetrics.Value = (*float64)(&value)
		agMetrics.MType = "gauge"
		agMetrics.ID = key

		buff := new(bytes.Buffer)
		gzip.CompressJSON(buff, agMetrics)

		res, err := sendRequest(http.MethodPost, "application/json", fullURL, buff, signKey)
		if err != nil {
			fmt.Println(err)
			return
		}

		res.Body.Close()
	}
	//prepare and send counter
	agMetrics.ID = "PollCount"
	agMetrics.MType = "counter"
	agMetrics.Value = nil
	agMetrics.Delta = (*int64)(&pollCount)

	buff := new(bytes.Buffer)
	gzip.CompressJSON(buff, agMetrics)

	res, err := sendRequest(http.MethodPost, "application/json", fullURL, buff, signKey)
	if err != nil {
		fmt.Println(err)
		return
	}
	res.Body.Close()
}


func SendMetricsJSONBatch(mu *sync.RWMutex, memory *agmemory.AgMemory, URL string, signKey string, rateLimit int) {
	mu.RLock()
	defer mu.RUnlock()

	metricData := memory.RuntimeMetrics
	pollCount := memory.PollCount["PollCount"]

	fullURL := fmt.Sprintf("http://%s/updates/", URL)
	var agMetrics agmemory.Metrics
	agMetricsArray := make([]agmemory.Metrics, 0)

	//prepare gauges metrics
	for key, value := range metricData {
		val := new(float64)
		*val = float64(value)
		agMetrics.Value = val
		agMetrics.MType = "gauge"
		agMetrics.ID = key
		agMetricsArray = append(agMetricsArray, agMetrics)
	}

	//prepare counter metric
	agMetrics.ID = "PollCount"
	agMetrics.MType = "counter"
	agMetrics.Value = nil
	agMetrics.Delta = (*int64)(&pollCount)
	agMetricsArray = append(agMetricsArray, agMetrics)

	//some checks
	agMetricsLenght := len(agMetricsArray)
	if agMetricsLenght < rateLimit {
		rateLimit = agMetricsLenght
	}
	chunk := agMetricsLenght / rateLimit
	
	//init semaphore with RATE_LIMIT
	sem := Semaphore{
		C: make(chan struct{}, rateLimit),
	}

	w := 0
	if rateLimit > 1 {
		//worker pool
		for w = 0; w < rateLimit-1; w++ {
			go worker(sem, agMetricsArray[w*chunk:(w+1)*chunk], fullURL, signKey)
		}
	}
	go worker(sem, agMetricsArray[w*chunk:agMetricsLenght], fullURL, signKey)
}

func worker(sem Semaphore, agMetricsArray []agmemory.Metrics, fullURL string, signKey string) {
	sem.Acquire()       //block routine via struct{}{} literal
	defer sem.Release() //unblock via read from chan

	buff := new(bytes.Buffer)
	gzip.CompressJSON(buff, agMetricsArray)

	res, err := sendRequest(http.MethodPost, "application/json", fullURL, buff, signKey)
	if err != nil {
		fmt.Println(err)
		return
	}
	res.Body.Close()
}

func sendRequest(method, contentType, url string, body *bytes.Buffer, signKey string) (*http.Response, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("error new request: %w", err)
	}

	req.Header.Set("Content-Type", contentType)
	req.Header.Add("Content-Encoding", "gzip")

	//check if KEY is exists and sign plainttext with SHA256 algoritm
	hash, err := crypt.SignDataWithSHA256(body.Bytes(), signKey)
	if err == nil {
		req.Header.Add("HashSHA256", hash)
	}

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		if errors.Is(err, syscall.ECONNREFUSED) {
			res, err := tryToResendReq(client, req)
			if err != nil {
				return nil, err
			}
			return res, nil
		} else {
			return nil, fmt.Errorf("other error send data: %w", err)
		}
	}
	return res, nil
}

func tryToResendReq(client *http.Client, req *http.Request) (*http.Response, error) {
	var err error
	var res *http.Response

	for attempts := 1; attempts < 4; attempts++ {
		//1s 3s 5s
		switch attempts {
		case 1:
			Sleep("1s")

			res, err = client.Do(req)
			if err == nil {
				return res, nil
			}

		case 2:
			Sleep("3s")

			res, err = client.Do(req)
			if err == nil {
				return res, nil
			}
		case 3:
			Sleep("5s")

			res, err = client.Do(req)
			if err == nil {
				return res, nil
			}
		}
	}
	return nil, fmt.Errorf("error send data after 3 attempts: %w", err)
}

func Sleep(suffix string) {
	t, _ := time.ParseDuration(suffix)
	fmt.Println("Try to anower resend after: ", t, ".....")
	time.Sleep(t)
}
