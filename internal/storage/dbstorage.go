package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jackc/pgx"
	_ "github.com/jackc/pgx/v5/stdlib"
)

type DBStorage struct {
	DB      *sql.DB
}

// DB init
func ConnectDB(dsn string) (*DBStorage, error) {
	dbs := &DBStorage{}

	if err := checkDSN(dsn); err != nil {
		return dbs, err
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}
	dbs.DB = db
	err = createTables(dbs)

	return dbs, err
}

func checkDSN(dsn string) (err error) {
	_, err = pgx.ParseDSN(dsn)
	return err
}

func createTables(d *DBStorage) (err error) {
	const (
		tableCounter = `CREATE TABLE IF NOT EXISTS Counter (id varchar(255) PRIMARY KEY, delta bigint);`
		tableGauge   = `CREATE TABLE IF NOT EXISTS Gauge (id varchar(255) PRIMARY KEY, value double precision);`
	)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if _, err = d.DB.ExecContext(ctx, tableCounter); err != nil {
		return fmt.Errorf("error create table Counter: %v", err)
	}
	if _, err = d.DB.ExecContext(ctx, tableGauge); err != nil {
		return fmt.Errorf("error create table Gauge: %v", err)
	}
	return nil
}

func (d *DBStorage) AddNewCounter(ctx context.Context, key string, value Counter) error {
	insertQuery := `INSERT INTO Counter (id, delta) VALUES ($1, $2);`
	updateQuery := `UPDATE Counter SET delta = delta + $1 WHERE id = $2`

	//UPDATE
	res, err := d.DB.ExecContext(ctx, updateQuery, int64(value), key)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows > 0 {
		return nil
	}
	//or INSERT
	_, err = d.DB.ExecContext(ctx, insertQuery, key, int64(value))
	return err
}

func (d *DBStorage) UpdateGauge(ctx context.Context, key string, value Gauge) error {
	insertQuery := `INSERT INTO Gauge (id, value) VALUES ($1, $2);`
	updateQuery := `UPDATE Gauge SET value = $1 WHERE id = $2;`

	//UPDATE
	res, err := d.DB.ExecContext(ctx, updateQuery, float64(value), key)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows > 0 {
		return nil
	}
	//or INSERT
	_, err = d.DB.ExecContext(ctx, insertQuery, key, float64(value))
	return err
}

func (d *DBStorage) GetGaugeByKey(ctx context.Context, key string) (Gauge, error) {
	selectQuery := `SELECT value FROM Gauge WHERE id = $1`
	row := d.DB.QueryRowContext(ctx, selectQuery, key)
	var val float64
	err := row.Scan(&val)
	return Gauge(val), err
}

func (d *DBStorage) GetCounterByKey(ctx context.Context, key string) (Counter, error) {
	selectQuery := `SELECT delta FROM Counter WHERE id = $1`
	row := d.DB.QueryRowContext(ctx, selectQuery, key)
	var val int64
	err := row.Scan(&val)
	return Counter(val), err
}

func (d *DBStorage) GetAllGauges(ctx context.Context) (map[string]Gauge, error) {
	m := make(map[string]Gauge)
	selectQuery := `SELECT id, value FROM Gauge`
	rows, err := d.DB.QueryContext(ctx, selectQuery)

	if err != nil {
		return m, err
	}
	defer rows.Close()

	var id string
	var value float64

	for rows.Next() {
		err := rows.Scan(&id, &value)
		if err != nil {
			return m, err
		}
		m[id] = Gauge(value)
	}

	err = rows.Err()
	if err != nil {
		return nil, err
	}

	return m, nil
}

func (d *DBStorage) GetAllCounters(ctx context.Context) (map[string]Counter, error) {
	m := make(map[string]Counter)
	selectQuery := `SELECT id, delta FROM Counter`
	rows, err := d.DB.QueryContext(ctx, selectQuery)

	if err != nil {
		return m, err
	}
	defer rows.Close()

	var id string
	var value int64

	for rows.Next() {
		err := rows.Scan(&id, &value)
		if err != nil {
			return m, err
		}
		m[id] = Counter(value)
	}

	err = rows.Err()
	if err != nil {
		return nil, err
	}

	return m, nil
}

func (d *DBStorage) DBPing(ctx context.Context) error {
	err := d.DB.PingContext(ctx)
	return err
}
