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

/*type JSONMetrics struct {
	Name       string   `json:"name"`               // metric Name
	Type       string   `json:"type"`               // Type gauge or counter
	CountValue *int64   `json:"countval,omitempty"` // pointer on CountValue (pointer need for check on nil)
	GaugeValue *float64 `json:"gaugeval,omitempty"` // pointer on GaugeValue (pointer need for check on nil)
}*/

type Metrics struct {
	ID    string   `json:"id"`              // metric Name
	MType string   `json:"type"`            // Type gauge or counter
	Delta *int64   `json:"delta,omitempty"` // pointer on CountValue (pointer need for check on nil)
	Value *float64 `json:"value,omitempty"` // pointer on GaugeValue (pointer need for check on nil)
}


