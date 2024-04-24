package servconfig

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestParseParameters(t *testing.T) {
	cfg := ParseParameters()
	assert.Equal(t, "localhost:8080", cfg.ListenAddr, "test #ListenAddr")
	assert.Equal(t, time.Duration(300*time.Second), cfg.StoreInterval, "test #StoreInterval")
	assert.Equal(t, "/tmp/metrics-db.json", cfg.StoreFile, "test #StoreFile")
	assert.Equal(t, true, cfg.Restore, "test #Restore")
	assert.Equal(t, "", cfg.DatabaseDSN, "test #DatabaseDSN")
	assert.Equal(t, "", cfg.Key, "test #DatabaseDSN")
}
