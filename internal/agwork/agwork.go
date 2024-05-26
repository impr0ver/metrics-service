package agwork

import (
	"bytes"
	"context"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"net"
	"strconv"
	"syscall"

	"fmt"
	"math/rand"
	"net/http"
	"runtime"
	"sync"
	"time"

	"github.com/impr0ver/metrics-service/internal/agconfig"
	"github.com/impr0ver/metrics-service/internal/agmemory"
	"github.com/impr0ver/metrics-service/internal/crypt"
	"github.com/impr0ver/metrics-service/internal/gzip"
	proto "github.com/impr0ver/metrics-service/internal/rpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
)

type (
	Sender interface {
		SendMetricsJSONBatch()
	}

	HTTPSendMetrics struct {
		Cfg agconfig.Config
		Am  *agmemory.AgMemory
	}

	GRPCSendMetrics struct {
		Cfg agconfig.Config
		Am  *agmemory.AgMemory
	}
)

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

func (hs GRPCSendMetrics) SendMetricsJSONBatch() {
	// init semaphore with RATE_LIMIT
	sem := agconfig.NewSemaphore(hs.Cfg.RateLimit)

	var mu sync.RWMutex
	mu.RLock()
	metricsLength := len(hs.Am.RuntimeMetrics) + 1 // + 1 counter, PollCount
	metricsArray := make([]proto.Metrics, metricsLength)
	i := 0

	for k, v := range hs.Am.RuntimeMetrics {
		metricsArray[i].Mtype = proto.Metrics_GAUGE
		metricsArray[i].Value = float64(v)
		metricsArray[i].Id = k
		i++
	}

	metricsArray[i].Id = "PollCount"
	metricsArray[i].Mtype = proto.Metrics_COUNTER
	metricsArray[i].Delta = (int64)(hs.Am.PollCount["PollCount"])
	mu.RUnlock()

	gRPCWorker := func(sem *agconfig.Semaphore, start int, step int) {
		sem.Acquire()       //block routine via struct{}{} literal
		defer sem.Release() //unblock via read from chan

		end := start + step
		if end > metricsLength {
			end = metricsLength
		}

		metrics := proto.MetricsArray{}
		metrics.Metrics = make([]*proto.Metrics, 0, end-start)
		for i := start; i < end; i++ {
			metrics.Metrics = append(metrics.Metrics, &metricsArray[i])
		}

		// grpc.Dial is DEPRECATED, need to use grpc.NewClient!
		conn, err := grpc.NewClient("passthrough:///"+hs.Cfg.GRPCAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			fmt.Println(err)
			return
		}
		defer conn.Close()

		cli := proto.NewMetricsExhangeClient(conn)

		parent := context.Background()

		if hs.Cfg.PublicKey != nil {
			cryptMetrics := proto.CryptMetrics{}

			metricsBytes, _ := json.Marshal(&metrics)

			cryptMetrics.Cryptbuff, err = crypt.EncryptPKCS1v15(hs.Cfg.PublicKey, metricsBytes)
			if err != nil {
				fmt.Println(err)
				return
			}

			// Add in metadata hash if cfg.Key is set
			hash, err := crypt.SignDataWithSHA256([]byte(cryptMetrics.String()), hs.Cfg.Key)
			if err == nil {
				md := metadata.New(map[string]string{"hashsha256": hash})
				parent = metadata.NewOutgoingContext(context.Background(), md)
			}

			ctx, cancel := context.WithTimeout(parent, time.Second)
			defer cancel()

			// MetricsUpdatesResponse
			respUpdates, err := cli.CryptUpdates(ctx, &cryptMetrics)
			if err != nil {
				fmt.Println(err)
				return
			}
			if respUpdates.Error == "" {
				fmt.Println("gRPC response: Successfully updated!")
			} else {
				fmt.Println(respUpdates.Error)
			}
		} else { // Work with plain data

			// Add in metadata hash if cfg.Key is set
			hash, err := crypt.SignDataWithSHA256([]byte(metrics.String()), hs.Cfg.Key)
			if err == nil {
				md := metadata.New(map[string]string{"hashsha256": hash})
				parent = metadata.NewOutgoingContext(context.Background(), md)
			}

			ctx, cancel := context.WithTimeout(parent, time.Second)
			defer cancel()

			// MetricsUpdatesResponse
			respUpdates, err := cli.Updates(ctx, &metrics)
			if err != nil {
				fmt.Println(err)
				return
			}
			if respUpdates.Error == "" {
				fmt.Println("gRPC response: Successfully updated!")
			} else {
				fmt.Println(respUpdates.Error)
			}
		}
	}

	limit := hs.Cfg.RateLimit
	if metricsLength < limit {
		limit = metricsLength
	}
	step := (metricsLength / limit) + (metricsLength % 2)

	for w := 0; w < limit; w += 1 {
		go gRPCWorker(sem, w*step, step)
	}
}

func (hs HTTPSendMetrics) SendMetricsJSONBatch() {
	var mu sync.RWMutex
	mu.RLock()
	defer mu.RUnlock()

	// init semaphore with RATE_LIMIT
	sem := agconfig.NewSemaphore(hs.Cfg.RateLimit)

	metricData := hs.Am.RuntimeMetrics
	pollCount := hs.Am.PollCount["PollCount"]

	fullURL := fmt.Sprintf("http://%s/updates/", hs.Cfg.Address)
	var agMetrics agmemory.Metrics

	agMetricsArray := make([]agmemory.Metrics, 0)

	// prepare gauges metrics
	for key, value := range metricData {
		val := new(float64)
		*val = float64(value)
		agMetrics.Value = val
		agMetrics.MType = "gauge"
		agMetrics.ID = key
		agMetricsArray = append(agMetricsArray, agMetrics)
	}

	// prepare counter metric
	agMetrics.ID = "PollCount"
	agMetrics.MType = "counter"
	agMetrics.Value = nil
	agMetrics.Delta = (*int64)(&pollCount)
	agMetricsArray = append(agMetricsArray, agMetrics)

	// some checks
	agMetricsLenght := len(agMetricsArray)
	if agMetricsLenght < hs.Cfg.RateLimit {
		hs.Cfg.RateLimit = agMetricsLenght
	}
	chunk := agMetricsLenght / hs.Cfg.RateLimit

	w := 0
	if hs.Cfg.RateLimit > 1 {
		// worker pool
		for w = 0; w < hs.Cfg.RateLimit-1; w++ {
			go worker(sem, agMetricsArray[w*chunk:(w+1)*chunk], fullURL, hs.Cfg.Key, hs.Cfg.PublicKey, hs.Cfg.RealHostIP)
		}
	}
	go worker(sem, agMetricsArray[w*chunk:agMetricsLenght], fullURL, hs.Cfg.Key, hs.Cfg.PublicKey, hs.Cfg.RealHostIP)
}

func worker(sem *agconfig.Semaphore, agMetricsArray []agmemory.Metrics, fullURL string, signKey string, publicKey *rsa.PublicKey, realIP string) {
	sem.Acquire()       //block routine via struct{}{} literal
	defer sem.Release() //unblock via read from chan

	buff := new(bytes.Buffer)

	gzip.CompressJSON(buff, agMetricsArray)

	contentType := "application/json"

	if publicKey != nil {
		cryptBuff, err := crypt.EncryptPKCS1v15(publicKey, buff.Bytes())
		if err != nil {
			fmt.Println(err)
			return
		}
		contentType = "application/octet-stream"
		buff.Reset()
		buff.Write(cryptBuff)
	}

	res, err := sendRequest(http.MethodPost, contentType, fullURL, buff, signKey, realIP)
	if err != nil {
		fmt.Println(err)
		return
	}
	res.Body.Close()
}

func sendRequest(method, contentType, url string, body *bytes.Buffer, signKey string, realIP string) (*http.Response, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("error new request: %w", err)
	}

	if realIP != "" {
		req.Header.Add("X-Real-IP", realIP)
	}

	req.Header.Set("Content-Type", contentType)
	req.Header.Add("Content-Encoding", "gzip")

	// check if KEY is exists and sign plainttext with SHA256 algoritm
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

func GetHostIP(srvAddr string) string {
	conn, err := net.Dial("tcp", srvAddr)
	if err != nil {
		fmt.Println(err)
		return ""
	}

	ip, _, err := net.SplitHostPort(conn.LocalAddr().String())
	if err != nil {
		fmt.Println(err)
		return ""
	}
	return ip
}
