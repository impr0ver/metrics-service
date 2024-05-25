package agmemory

// Agent memory
type (
	Gauge   float64
	Counter int64

	AgMemory struct {
		RuntimeMetrics map[string]Gauge
		PollCount      map[string]Counter
	}

	Metrics struct {
		ID    string   `json:"id"`              // metric Name
		MType string   `json:"type"`            // Type gauge or counter
		Delta *int64   `json:"delta,omitempty"` // pointer on CountValue (pointer need for check on nil)
		Value *float64 `json:"value,omitempty"` // pointer on GaugeValue (pointer need for check on nil)
	}
)

func NewAgMemory() AgMemory {
	agMemory := AgMemory{RuntimeMetrics: make(map[string]Gauge), PollCount: make(map[string]Counter)}
	return agMemory
}
