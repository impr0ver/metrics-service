package gzip_test

import (
	"bytes"

	"testing"

	"github.com/impr0ver/metrics-service/internal/gzip"
	"github.com/stretchr/testify/assert"
)

func TestCompressJSON(t *testing.T) {
	exampleJSON := `[{ "id": "MCacheSys", "type": "gauge", "value": 15600 },
  { "id": "StackInuse", "type": "gauge", "value": 327680 },
  { "id": "HeapInuse", "type": "gauge", "value": 811008 },
  { "id": "CPUutilization1", "type": "gauge", "value": 1.9801980198269442 },
  { "id": "StackSys", "type": "gauge", "value": 327680 },
  { "id": "GCSys", "type": "gauge", "value": 8055592 },
  { "id": "Alloc", "type": "gauge", "value": 308568 },
  { "id": "MCacheInuse", "type": "gauge", "value": 1200 }]`

	buff := new(bytes.Buffer)

	gzip.CompressJSON(buff, exampleJSON)

	assert.Equal(t, len(exampleJSON), 480, "test #Len before")
	assert.Equal(t, buff.Len(), 198, "test #Len after")
}
