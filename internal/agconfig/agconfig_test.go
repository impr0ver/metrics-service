package agconfig

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestInitConfig(t *testing.T) {
	os.Setenv("ADDRESS", "")
	cfg := InitConfig()
	assert.Equal(t, "localhost:8080", cfg.Address, "test #Address")
	os.Unsetenv("ADDRESS")

	assert.Equal(t, time.Duration(2000000000), cfg.PollInterval, "test #PollInterval")
	assert.Equal(t, time.Duration(10000000000), cfg.ReportInterval, "test #ReportInterval")
	assert.Equal(t, "", cfg.Key, "test #Key")
	assert.Equal(t, 2, cfg.RateLimit, "test #RateLimit")
}
