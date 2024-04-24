package storage_test

import (
	"bufio"
	"context"
	"database/sql"
	"log"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/impr0ver/metrics-service/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type DBStorageTestSuite struct {
	suite.Suite
	DB      *storage.DBStorage
	TestDSN string
}

func TestMemory(t *testing.T) {
	st := storage.MemoryStorage{Gauges: make(map[string]storage.Gauge), Counters: make(map[string]storage.Counter)}

	tests := []struct {
		testname string
		gauge    string
		value    storage.Gauge
	}{
		{"test#1", "gauge1", storage.Gauge(0.1)},
		{"test#2", "gauge2", storage.Gauge(1.12)},
		{"test#3", "gauge3", storage.Gauge(10452.1)},
		{"test#4", "gauge4", storage.Gauge(485.1)},
		{"test#5", "gauge5", storage.Gauge(222.00)},
		{"test#6", "gauge6", storage.Gauge(5.1)},
		{"test#7", "gauge7", storage.Gauge(40.23)},
		{"test#8", "gauge8", storage.Gauge(7.99)},
		{"test#9", "gauge9", storage.Gauge(67.9)},
		{"test#10", "gauge10", storage.Gauge(0.1)},
		{"test#11", "gauge11", storage.Gauge(10017.2)},
		{"test#12", "gauge12", storage.Gauge(1.1)},
		{"test#13", "gauge13", storage.Gauge(12.22)},
		{"test#14", "gauge14", storage.Gauge(214.2)},
		{"test#15", "gauge15", storage.Gauge(127.2)},
		{"test#16", "gauge16", storage.Gauge(16007.2)},
		{"test#17", "gauge17", storage.Gauge(217.2)},
		{"test#18", "gauge18", storage.Gauge(218.10)},
		{"test#19", "gauge19", storage.Gauge(56.8)},
		{"test#20", "gauge20", storage.Gauge(0.00002)},
		{"test#21", "gauge21", storage.Gauge(946.9)},
		{"test#22", "gauge22", storage.Gauge(32.3)},
		{"test#23", "gauge23", storage.Gauge(97.3)},
		{"test#24", "gauge24", storage.Gauge(4.3)},
		{"test#25", "gauge25", storage.Gauge(44.3)},
		{"test#26", "gauge26", storage.Gauge(33.53333)},
		{"test#27", "gauge27", storage.Gauge(317.2)},
		{"test#28", "gauge28", storage.Gauge(18.00000)},
		{"test#29", "gauge29", storage.Gauge(111.1)},
		{"test#30", "gauge30", storage.Gauge(20.5)},
		{"test#31", "gauge31", storage.Gauge(946.9)},
		{"test#32", "gauge32", storage.Gauge(32.3)},
		{"test#33", "gauge33", storage.Gauge(813.3)},
		{"test#34", "gauge34", storage.Gauge(4.3)},
		{"test#35", "gauge35", storage.Gauge(544.3)},
		{"test#36", "gauge36", storage.Gauge(6.33333333333)},
		{"test#37", "gauge37", storage.Gauge(317.2)},
		{"test#38", "gauge38", storage.Gauge(18.00000)},
		{"test#39", "gauge39", storage.Gauge(111.1)},
		{"test#40", "gauge40", storage.Gauge(20.5)},
	}
	var wg sync.WaitGroup
	wg.Add(4)

	go func(storage *storage.MemoryStorage) {
		for i := 0; i < 40; i++ {
			storage.UpdateGauge(context.TODO(), tests[i].gauge, tests[i].value)
		}
		wg.Done()
	}(&st)

	go func() {
		for i := 0; i < 10; i++ {
			st.UpdateGauge(context.TODO(), tests[i].gauge, tests[i].value)
		}
		wg.Done()
	}()

	go func(storage *storage.MemoryStorage) {
		for i := 10; i < 20; i++ {
			storage.UpdateGauge(context.TODO(), tests[i].gauge, tests[i].value)
		}
		wg.Done()
	}(&st)

	go func() {
		for i := 20; i < 40; i++ {

			st.UpdateGauge(context.TODO(), tests[i].gauge, tests[i].value)
		}
		wg.Done()
	}()

	wg.Wait()
	for _, tt := range tests {
		value, err := st.GetGaugeByKey(context.TODO(), tt.gauge)
		require.NoError(t, err)
		assert.Equal(t, value, tt.value)
	}
}

func TestStoreToFile(t *testing.T) {
	st := storage.MemoryStorage{Gauges: make(map[string]storage.Gauge), Counters: make(map[string]storage.Counter)}

	st.UpdateGauge(context.TODO(), "key1", storage.Gauge(1.1))
	st.UpdateGauge(context.TODO(), "key2", storage.Gauge(2.22))
	st.UpdateGauge(context.TODO(), "key3", storage.Gauge(3.333))
	st.UpdateGauge(context.TODO(), "key4", storage.Gauge(4.4444))
	st.AddNewCounter(context.TODO(), "Counter1", storage.Counter(100))
	st.AddNewCounter(context.TODO(), "Counter2", storage.Counter(200))

	filePath := "./temp.json"
	err := storage.StoreToFile(&st, filePath)
	require.NoError(t, err)
	fm, err := os.Open(filePath)
	if err != nil {
		log.Fatal(err)
	}
	defer fm.Close()
	require.NoError(t, err)
	reader := bufio.NewReader(fm)
	line, _, _ := reader.ReadLine()
	expected := `{"Gauges":{"key1":1.1,"key2":2.22,"key3":3.333,"key4":4.4444},"Counters":{"Counter1":100,"Counter2":200}}`
	assert.Equal(t, expected, string(line))
	os.Remove(filePath)
}

func TestRestoreFromFile(t *testing.T) {
	st := storage.MemoryStorage{Gauges: make(map[string]storage.Gauge), Counters: make(map[string]storage.Counter)}

	st.UpdateGauge(context.TODO(), "key1", storage.Gauge(1.1))
	st.UpdateGauge(context.TODO(), "key2", storage.Gauge(2.22))
	st.UpdateGauge(context.TODO(), "key3", storage.Gauge(3.333))
	st.UpdateGauge(context.TODO(), "key4", storage.Gauge(4.4444))
	st.AddNewCounter(context.TODO(), "Counter1", storage.Counter(100))
	st.AddNewCounter(context.TODO(), "Counter2", storage.Counter(200))
	filePath := "./temp2.json"
	content := `{"Gauges":{"key1":1.1,"key2":2.22,"key3":3.333,"key4":4.4444},"Counters":{"Counter1":100,"Counter2":200}}`
	fm, err := os.Create(filePath)
	if err != nil {
		log.Fatal(err)
	}
	defer fm.Close()
	require.NoError(t, err)
	fm.Write([]byte(content))

	st2 := storage.MemoryStorage{Gauges: make(map[string]storage.Gauge), Counters: make(map[string]storage.Counter)}
	err = storage.RestoreFromFile(&st2, filePath)
	require.NoError(t, err)
	assert.Equal(t, st.Gauges, st2.Gauges)
	assert.Equal(t, st.Counters, st2.Counters)
	os.Remove(filePath)
}

func (suite *DBStorageTestSuite) SetupSuite() {
	suite.DB = &storage.DBStorage{DB: nil}

	dsn := "postgresql://localhost:5432?user=postgres&password=postgres"
	dbname := "testdb"

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return
	}

	db.Exec("DROP DATABASE " + dbname)
	_, err = db.Exec("CREATE DATABASE " + dbname)
	db.Close()
	if err != nil {
		return
	}

	testDSN := "postgresql://localhost:5432/" + dbname + "?user=postgres&password=postgres"
	suite.DB, _ = storage.ConnectDB(context.TODO(), testDSN)
}

func (suite *DBStorageTestSuite) TestDBStorageAddCounterAndGetCounter() {
	ctx := context.Background()

	tests := []struct {
		name  string
		key   string
		value storage.Counter
		want  storage.Counter
	}{
		{"test#1", "CounterA", 555, 555},
		{"test#2", "CounterB", 100, 100},
		{"test#3", "CounterB", 100, 200},
		{"test#4", "CounterB", 100, 300},
	}
	for _, tt := range tests {
		suite.Run(tt.name, func() {
			err := suite.DB.AddNewCounter(ctx, tt.key, tt.value)
			suite.NoError(err, tt.name+" failed")

			res, err := suite.DB.GetCounterByKey(ctx, tt.key)
			suite.NoError(err, tt.name+", GetCounterByKey failed")

			suite.Equal(storage.Counter(res), tt.want)
		})
	}
}

func (suite *DBStorageTestSuite) TestDBStorageUpdateGaugeAndGetGauge() {
	ctx := context.Background()

	tests := []struct {
		name  string
		key   string
		value storage.Gauge
		want  storage.Gauge
	}{
		{"simple test #1", "Gauge1", 54321.12345, 54321.12345},
		{"simple test #2", "Gauge2", 99.10, 99.10},
		{"simple test #3", "Gauge3", 800000.1, 800000.1},
		{"simple test #4", "Gauge4", 100000, 100000},
	}
	for _, tt := range tests {
		suite.Run(tt.name, func() {
			err := suite.DB.UpdateGauge(ctx, tt.key, tt.value)
			suite.NoError(err, tt.name+" failed")

			res, err := suite.DB.GetGaugeByKey(ctx, tt.key)
			suite.NoError(err, tt.name+", GetGaugeByKey failed")

			suite.Equal(storage.Gauge(res), tt.want)
		})
	}
}

func (suite *DBStorageTestSuite) TestDBStorageGetAllGauges() {
	ctx := context.Background()

	err := suite.DB.UpdateGauge(ctx, "GaugeA", 83584389.129845)
	suite.NoError(err, "UpdageGauge failed")
	err = suite.DB.UpdateGauge(ctx, "GaugeB", 19858.999)
	suite.NoError(err, "UpdageGauge failed")
	err = suite.DB.UpdateGauge(ctx, "GaugeC", 69694)
	suite.NoError(err, "UpdageGauge failed")
	err = suite.DB.UpdateGauge(ctx, "GaugeD", 945345.14848)
	suite.NoError(err, "UpdageGauge failed")
	err = suite.DB.UpdateGauge(ctx, "GaugeE", 968332.666)
	suite.NoError(err, "UpdageGauge failed")

	err = suite.DB.UpdateGauge(ctx, "GaugeE", 968332.667) //update 968332.666 -> 968332.667
	suite.NoError(err, "UpdageGauge failed")

	mapGauges, err := suite.DB.GetAllGauges(ctx)
	suite.NoError(err, "GetAllGauges failed")
	len := len(mapGauges)
	suite.Equal(5, len)

}

func (suite *DBStorageTestSuite) TestDBStorageGetAllCounters() {
	ctx := context.Background()
	err := suite.DB.AddNewCounter(ctx, "Counter1", 5)
	suite.NoError(err, "AddCounter failed")
	err = suite.DB.AddNewCounter(ctx, "Counter2", 10)
	suite.NoError(err, "AddCounter failed")
	err = suite.DB.AddNewCounter(ctx, "Counter3", 15)
	suite.NoError(err, "AddCounter failed")
	err = suite.DB.AddNewCounter(ctx, "Counter4", 20)
	suite.NoError(err, "AddCounter failed")
	err = suite.DB.AddNewCounter(ctx, "Counter5", 25)
	suite.NoError(err, "AddCounter failed")

	err = suite.DB.AddNewCounter(ctx, "Counter5", 30) //25 + 30 = 55
	suite.NoError(err, "AddCounter failed")

	mapCounters, err := suite.DB.GetAllCounters(ctx)
	suite.NoError(err, "GetAllCounters failed")
	len := len(mapCounters)
	suite.Equal(5, len)

}

func (suite *DBStorageTestSuite) TestAddNewMetricsAsBatch() {

	counter1 := int64(10)
	counter2 := int64(20)
	counter3 := int64(10)

	gauge1 := float64(123.234)
	gauge2 := float64(234.345)

	metrics := [5]storage.Metrics{
		{ID: "tstMetric #1", MType: "counter", Value: nil, Delta: &counter1},
		{ID: "tstMetric #2", MType: "counter", Value: nil, Delta: &counter2},
		{ID: "tstMetric #2", MType: "counter", Value: nil, Delta: &counter3},
		{ID: "tstMetric #3", MType: "gauge", Value: &gauge1, Delta: nil},
		{ID: "tstMetric #4", MType: "gauge", Value: &gauge2, Delta: nil},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := suite.DB.AddNewMetricsAsBatch(ctx, metrics[:])
	suite.NoError(err)

	allCounters, err := suite.DB.GetAllCounters(ctx)
	suite.NotNil(allCounters)
	suite.NoError(err)
	counterOne, ok := allCounters["tstMetric #1"]
	suite.Equal(true, ok)
	suite.Equal(counter1, int64(counterOne))
	counterTwo, ok := allCounters["tstMetric #2"]
	suite.Equal(true, ok)
	suite.Equal(counter2+counter3, int64(counterTwo))

	allGauges, err := suite.DB.GetAllGauges(ctx)
	suite.NotNil(allGauges)
	suite.NoError(err)
	gaugeOne, ok := allGauges["tstMetric #3"]
	suite.Equal(true, ok)
	suite.Equal(gauge1, float64(gaugeOne))

	gaugeTwo, ok := allGauges["tstMetric #4"]
	suite.Equal(true, ok)
	suite.Equal(gauge2, float64(gaugeTwo))

}

func (suite *DBStorageTestSuite) SetupTest() {
	suite.DB.DB.Exec("TRUNCATE Gauge, Counter CASCADE;")
}

func TestDBStorageTestSuite(t *testing.T) {
	dsn := "postgresql://localhost:5432?user=postgres&password=postgres"

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return
	}
	err = db.PingContext(context.TODO())
	if err != nil {
		return
	}
	db.Close()
	suite.Run(t, new(DBStorageTestSuite))
}
