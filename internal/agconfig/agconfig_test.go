package agconfig

import (
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitConfig(t *testing.T) {
	os.Setenv("ADDRESS", "")
	cfgTest := InitConfig()
	assert.Equal(t, "localhost:8080", cfgTest.Address, "test #Address")
	os.Unsetenv("ADDRESS")

	assert.Equal(t, time.Duration(2000000000), cfgTest.PollInterval, "test #PollInterval")
	assert.Equal(t, time.Duration(10000000000), cfgTest.ReportInterval, "test #ReportInterval")
	assert.Equal(t, "", cfgTest.Key, "test #Key")
	assert.Equal(t, 2, cfgTest.RateLimit, "test #RateLimit")

	jsonData := `{
		"address": "localhost:8080",
		"report_interval": "1s",
		"poll_interval": "1s",
		"crypto_key": "../genkeys/public.pem"
	}`

	err := cfgTest.UnmarshalJSON([]byte(jsonData))
	require.NoError(t, err)

	assert.Equal(t, time.Duration(1*time.Second), cfgTest.PollInterval, "test #PollInterval after duration")
	assert.Equal(t, time.Duration(1*time.Second), cfgTest.ReportInterval, "test #ReportInterval after duration")

	f, err := os.Create("./testConfig.json")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	n2, err := f.WriteString(`{
		"address": "localhost:8080",
		"report_interval": "1s",
		"poll_interval": "1s",
		"crypto_key": "../genkeys/public.pem"
	}`)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("wrote %d bytes\n", n2)

	os.Setenv("CONFIG", "./testConfig.json")

	tmpCfg, err := readConfigFile()
	if err != nil {
		log.Fatal(err)
	}

	assert.Equal(t, "localhost:8080", tmpCfg.Address, "test #readConfigFile")
	assert.Equal(t, time.Duration(1*time.Second), tmpCfg.ReportInterval, "test #readConfigFile2")
	assert.Equal(t, time.Duration(1*time.Second), tmpCfg.PollInterval, "test #readConfigFile3")
	assert.Equal(t, "../genkeys/public.pem", tmpCfg.PathToPublicKey, "test #readConfigFile4")

	os.Unsetenv("CONFIG")
	os.Remove("./testConfig.json")
}
