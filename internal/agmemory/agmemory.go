package agmemory

// Agent memory
type Gauge float64
type Counter int64


type AgMemory struct {
	RuntimeMetrics map[string]Gauge
	PollCount      map[string]Counter
}

func NewAgMemory() AgMemory {
	agMemory := AgMemory{RuntimeMetrics: make(map[string]Gauge), PollCount: make(map[string]Counter)}
	return agMemory
}