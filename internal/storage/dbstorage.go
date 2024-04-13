package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx"
	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"
)

type DBStorage struct {
	DB *sql.DB
}

// ConnectDB init connect to database.
func ConnectDB(ctx context.Context, dsn string) (*DBStorage, error) {
	dbs := &DBStorage{}

	if err := checkDSN(dsn); err != nil {
		return dbs, fmt.Errorf("wrong DSN: %w", err)
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return dbs, fmt.Errorf("unable connect to db: %w", err)
	}

	dbs.DB = db

	err = dbs.DB.PingContext(ctx)
	if err != nil {
		return dbs, err
	}

	err = createTables(ctx, dbs)

	return dbs, err
}

// checkDSN parse DSN string.
func checkDSN(dsn string) (err error) {
	_, err = pgx.ParseDSN(dsn)
	return err
}

//createTables creates two tables in db% Counter and Gauge
func createTables(ctx context.Context, d *DBStorage) (err error) {
	const (
		tableCounter = `CREATE TABLE IF NOT EXISTS Counter (id varchar(255) PRIMARY KEY, delta bigint);`
		tableGauge   = `CREATE TABLE IF NOT EXISTS Gauge (id varchar(255) PRIMARY KEY, value double precision);`
	)

	if _, err = d.DB.ExecContext(ctx, tableCounter); err != nil {
		return fmt.Errorf("error create table \"Counter\": %w", err)
	}
	if _, err = d.DB.ExecContext(ctx, tableGauge); err != nil {
		return fmt.Errorf("error create table \"Gauge\": %w", err)
	}
	return nil
}

// AddNewCounter - add new counter (storage in db).
func (d *DBStorage) AddNewCounter(ctx context.Context, key string, value Counter) error {
	_, err := d.DB.ExecContext(ctx, `INSERT INTO Counter (id, delta) VALUES ($1, $2);`, key, int64(value))
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
		_, err = d.DB.ExecContext(ctx, `UPDATE Counter SET delta = delta + $1 WHERE id = $2;`, int64(value), key)
	}
	return err
}

// UpdateGauge - update gauge value (storage in db).
func (d *DBStorage) UpdateGauge(ctx context.Context, key string, value Gauge) error {
	_, err := d.DB.ExecContext(ctx, `INSERT INTO Gauge (id, value) VALUES ($1, $2);`, key, float64(value))
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
		_, err = d.DB.ExecContext(ctx, `UPDATE Gauge SET value = $1 WHERE id = $2;`, int64(value), key)
	}

	return err
}

// GetGaugeByKey - get gauge value by key (storage in db).
func (d *DBStorage) GetGaugeByKey(ctx context.Context, key string) (Gauge, error) {
	selectQuery := `SELECT value FROM Gauge WHERE id = $1;`
	row := d.DB.QueryRowContext(ctx, selectQuery, key)
	var val float64
	err := row.Scan(&val)
	return Gauge(val), err
}

// GetCounterByKey - get counter value by key (storage in db).
func (d *DBStorage) GetCounterByKey(ctx context.Context, key string) (Counter, error) {
	selectQuery := `SELECT delta FROM Counter WHERE id = $1;`
	row := d.DB.QueryRowContext(ctx, selectQuery, key)
	var val int64
	err := row.Scan(&val)
	return Counter(val), err
}

// GetAllGauges - get all gauges (storage in db).
func (d *DBStorage) GetAllGauges(ctx context.Context) (map[string]Gauge, error) {
	res := make(map[string]Gauge)
	selectQuery := `SELECT id, value FROM Gauge;`
	rows, err := d.DB.QueryContext(ctx, selectQuery)

	if err != nil {
		return res, err
	}
	defer rows.Close()

	var id string
	var value float64

	for rows.Next() {
		err := rows.Scan(&id, &value)
		if err != nil {
			return res, err
		}
		res[id] = Gauge(value)
	}

	err = rows.Err()
	if err != nil {
		return nil, err
	}

	return res, nil
}

// GetAllCounters - get all counters (storage in db).
func (d *DBStorage) GetAllCounters(ctx context.Context) (map[string]Counter, error) {
	res := make(map[string]Counter)
	selectQuery := `SELECT id, delta FROM Counter;`
	rows, err := d.DB.QueryContext(ctx, selectQuery)

	if err != nil {
		return res, err
	}
	defer rows.Close()

	var id string
	var value int64

	for rows.Next() {
		err := rows.Scan(&id, &value)
		if err != nil {
			return res, err
		}
		res[id] = Counter(value)
	}

	err = rows.Err()
	if err != nil {
		return nil, err
	}

	return res, nil
}

// DBPing ping db for alive
func (d *DBStorage) DBPing(ctx context.Context) error {
	err := d.DB.PingContext(ctx)
	return err
}

// AddNewMetricsAsBatch add or update metrics (storage in db). 
func (d *DBStorage) AddNewMetricsAsBatch(ctx context.Context, metrics []Metrics) error {
	tx, err := d.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	counterPrepareStatement, err := tx.PrepareContext(ctx, `INSERT INTO Counter (id, delta) VALUES ($1, $2) ON CONFLICT (id) DO UPDATE SET delta = counter.delta + excluded.delta;`)
	if err != nil {
		return err
	}
	defer counterPrepareStatement.Close()

	gaugePrepareStatement, err := tx.PrepareContext(ctx, `INSERT INTO Gauge (id, value) VALUES ($1, $2) ON CONFLICT (id) DO UPDATE SET value = excluded.value;`)
	if err != nil {
		return err
	}
	defer gaugePrepareStatement.Close()

	for _, metric := range metrics {
		switch metric.MType {
		case "counter":
			if _, err = counterPrepareStatement.ExecContext(ctx, metric.ID, *metric.Delta); err != nil {
				return err
			}
		case "gauge":
			if _, err = gaugePrepareStatement.ExecContext(ctx, metric.ID, *metric.Value); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unsupport metric type")
		}
	}
	return tx.Commit()
}
