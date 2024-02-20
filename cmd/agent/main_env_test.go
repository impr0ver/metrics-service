package main

import (
	"os"
	"testing"

	"github.com/impr0ver/metrics-service/internal/agconfig"
	"github.com/stretchr/testify/assert"
)

func TestInitConfig2(t *testing.T) {
	//var cfg Config

	os.Setenv("ADDRESS", "")
	cfg := agconfig.InitConfig()
	assert.Equal(t, "localhost:8080", cfg.Address, "test ##")

	os.Unsetenv("ADDRESS")
}
