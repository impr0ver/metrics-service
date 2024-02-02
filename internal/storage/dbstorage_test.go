package storage_test

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/impr0ver/metrics-service/internal/storage"
	"github.com/stretchr/testify/suite"
)

type DBStorageTestSuite struct {
	suite.Suite
	DB      *storage.DBStorage
	TestDSN string
}

func (suite *DBStorageTestSuite) SetupSuite() {
	suite.DB = &storage.DBStorage{}

	myDSN := "user=postgres password=mypassword host=localhost port=5432 dbname=metrics sslmode=disable"

	db, err := sql.Open("pgx", myDSN)
	if err != nil {
		return
	}

	dbName := "mydb"
	db.Exec("DROP DATABASE " + dbName)
	_, err = db.Exec("CREATE DATABASE " + dbName)
	db.Close()
	if err != nil {
		return
	}

	suite.DB, err = storage.ConnectDB(myDSN)
	if err != nil {
		fmt.Println("error in ConncetDB:", err)
		return
	}
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

	allCounters, err := suite.DB.GetAllCounters(context.TODO())
	suite.NotNil(allCounters)
	suite.NoError(err)
	counterOne, ok := allCounters["tstMetric #1"]
	suite.Equal(true, ok)
	suite.Equal(counter1, int64(counterOne))
	counterTwo, ok := allCounters["tstMetric #2"]
	suite.Equal(true, ok)
	suite.Equal(counter2+counter3, int64(counterTwo))

	allGauges, err := suite.DB.GetAllGauges(context.TODO())
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
	dsn := "user=postgres password=mypassword host=localhost port=5432 dbname=metrics sslmode=disable"

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return
	}
	db.Close()
	suite.Run(t, new(DBStorageTestSuite))
}
