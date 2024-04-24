package agconfig

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInitConfig(t *testing.T) {
	os.Setenv("ADDRESS", "")
	cfg := InitConfig()
	assert.Equal(t, "localhost:8080", cfg.Address, "test #Address")
	os.Unsetenv("ADDRESS")

	assert.Equal(t, 2, cfg.PollInterval, "test #PollInterval")
	assert.Equal(t, 10, cfg.ReportInterval, "test #ReportInterval")
	assert.Equal(t, "", cfg.Key, "test #Key")
	assert.Equal(t, 2, cfg.RateLimit, "test #RateLimit")
}
