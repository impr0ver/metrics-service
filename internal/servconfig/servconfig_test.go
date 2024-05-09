package servconfig

import (
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseParameters(t *testing.T) {
	cfg := ParseParameters()
	assert.Equal(t, "localhost:8080", cfg.ListenAddr, "test #ListenAddr")
	assert.Equal(t, time.Duration(300*time.Second), cfg.StoreInterval, "test #StoreInterval")
	assert.Equal(t, "/tmp/metrics-db.json", cfg.StoreFile, "test #StoreFile")
	assert.Equal(t, true, cfg.Restore, "test #Restore")
	assert.Equal(t, "", cfg.DatabaseDSN, "test #DatabaseDSN")
	assert.Equal(t, "", cfg.Key, "test #DatabaseDSN")

	jsonData := `{
		"address": "localhost:8080",
		"restore": true,
		"store_interval": "1s",
		"store_file": "/tmp/metrics-db.json",
		"database_dsn": "",
		"crypto_key": "../genkeys/private.pem"
	}`

	err := cfg.UnmarshalJSON([]byte(jsonData))
	require.NoError(t, err)

	assert.Equal(t, time.Duration(1*time.Second), cfg.StoreInterval, "test #PollInterval after duration")

	f, err := os.Create("./testConfig.json")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	
	n2, err := f.WriteString(`{
		"address": "localhost:8080",
		"restore": true,
		"store_interval": "1s",
		"store_file": "/tmp/metrics-db.json",
		"database_dsn": "",
		"crypto_key": "../genkeys/private.pem"
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

	assert.Equal(t, "localhost:8080", tmpCfg.ListenAddr, "test #readConfigFile")
	assert.Equal(t, time.Duration(1*time.Second), tmpCfg.StoreInterval, "test #readConfigFile2")
	assert.Equal(t, "/tmp/metrics-db.json", tmpCfg.StoreFile, "test #readConfigFile3")
	assert.Equal(t, "", tmpCfg.DatabaseDSN, "test #readConfigFile4")
	assert.Equal(t, true, tmpCfg.Restore, "test #readConfigFile5")
	assert.Equal(t, "../genkeys/private.pem", tmpCfg.PathToPrivKey, "test #readConfigFile6")

	assert.NotEqual(t, tmpCfg.ListenAddr, "")
	assert.NotEqual(t, tmpCfg.StoreFile, "")
	assert.NotEqual(t, tmpCfg.PathToPrivKey, "")

	os.Unsetenv("CONFIG")
	os.Remove("./testConfig.json")
}
