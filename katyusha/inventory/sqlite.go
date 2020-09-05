package inventory

import (
	"context"
	"fmt"
	"database/sql"
	"github.com/mattn/go-sqlite3"
	"github.com/tmwalaszek/katyusha/katyusha"
	"os"
	"strings"
	"time"
)
var SQLiteSchema = `CREATE TABLE benchmark_configuration (
    id INTEGER PRIMARY KEY,
    description TEXT,
    url TEXT,
    method TEXT,
    requests_count INTEGER,
    concurrent_conns INTEGER,
    skip_verify INTEGER,
    abort_after INTEGER,
    ca TEXT, 
    cert TEXT,
    key TEXT,
    duration TEXT,
    keep_alive TEXT,
    request_delay TEXT,
    read_timeout TEXT,
    write_timeout TEXT,
    body BLOB,
    UNIQUE(description,url)
);

CREATE TABLE headers (
    id INTEGER PRIMARY KEY,
    header TEXT,
    benchmark_configuration INTEGER,    

    FOREIGN KEY(benchmark_configuration) REFERENCES benchmark_configuration(id)
    ON DELETE CASCADE
);

CREATE TABLE parameters (
    id INTEGER PRIMARY KEY,
    parameter TEXT,
    benchmark_configuration INTEGER,

    FOREIGN KEY(benchmark_configuration) REFERENCES benchmark_configuration(id) 
    ON DELETE CASCADE
);

CREATE TABLE benchmark_summary (
    id INTEGER PRIMARY KEY,
    start TEXT,
    end TEXT,
    duration TEXT,
    requests_count INTEGER,
    success_req INTEGER,
    fail_req INTEGER,
    data_transfered INTEGER,
    req_per_sec REAL,
    avg_req_time TEXT,
    min_req_time TEXT,
    max_req_time TEXT,
    p50_req_time TEXT,
    p75_req_time TEXT,
    p90_req_time TEXT,
    p99_req_time TEXT,
    benchmark_configuration INTEGER,

    FOREIGN KEY(benchmark_configuration) REFERENCES benchmark_configuration(id)
    ON DELETE CASCADE
);

CREATE TABLE errors (
    id INTEGER PRIMARY KEY,
    name TEXT,
    count INTEGER,
    benchmark_summary INTEGER,

    FOREIGN KEY(benchmark_summary) REFERENCES benchmark_summary(id) 
    ON DELETE CASCADE
);`

// Sqlite3 does not provide bool type
// In Sqlite3 true is int 1 and false is int 0
func boolToInt(b bool) int {
	if b {
		return 1
	}

	return 0
}

type SQLite struct {
	*DB
}

func NewSQLite(dbFile string) (*SQLite, error) {
	var createSchema bool

	_, err := os.Stat(dbFile)
	if os.IsNotExist(err) {
		createSchema = true
	}

	db, err := sql.Open("sqlite3", dbFile)
	if err != nil {
		return nil, err
	}

	if createSchema {
		_, err = db.Exec(SQLiteSchema)
		if err != nil {
			return nil, fmt.Errorf("Could not create inventory schema: %w", err)
		}
	}

	return &SQLite{
		&DB{db: db},
	}, nil
}

// Find and return all benchmarks configurations
func (s *SQLite) FindAllBenchmarks(ctx context.Context) ([]*BenchmarkConfiguration, error) {
	query := fmt.Sprintf("SELECT id,%s FROM benchmark_configuration", benchmarkFields)
	bcs, err := s.queryBenchmark(ctx, query)

	return bcs, err
}

// Find and return Bencharm configuration using ID
func (s *SQLite) FindBenchmarkByID(ctx context.Context, ID int64) ([]*BenchmarkConfiguration, error) {
	query := fmt.Sprintf("SELECT id,%s FROM benchmark_configuration where id = ?", benchmarkFields)
	bcs, err := s.queryBenchmark(ctx, query, ID)

	return bcs, err
}

// FindBenchmark by two unique fields url and description
func (s *SQLite) FindBenchmark(ctx context.Context, URL string, description string) (*BenchmarkConfiguration, error) {
	query := fmt.Sprintf("SELECT id,%s FROM benchmark_configuration WHERE url = ? AND description = ?", benchmarkFields)
	bcs, err := s.queryBenchmark(ctx, query, URL, description)
	if err != nil {
		return nil, err
	}

	if len(bcs) != 1 {
		return nil, nil
	}

	return bcs[0], nil
}

// FindBenchmarkByURL by url
func (s *SQLite) FindBenchmarkByURL(ctx context.Context, URL string) ([]*BenchmarkConfiguration, error) {
	query := fmt.Sprintf("SELECT id,%s FROM benchmark_configuration WHERE url = ?", benchmarkFields)

	bcs, err := s.queryBenchmark(ctx, query, URL)
	return bcs, err
}

// FindSummaryForBenchmark return summaries for benchmark
func (s *SQLite) FindSummaryForBenchmark(ctx context.Context, bcID int64) ([]*BenchmarkSummary, error) {
	query := fmt.Sprintf("SELECT id,%s FROM benchmark_summary WHERE benchmark_configuration = ?", summaryFields)

	summaries, err := s.querySummary(ctx, query, bcID)
	if err != nil {
		return nil, err
	}

	return summaries, err
}

// DeleteBenchmark deletes benchmark configuration and all associated summaries
func (s *SQLite) DeleteBenchmark(ctx context.Context, bcID int64) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("Can't start transaction: %v", err)
	}

	query := "DELETE FROM benchmark_configuration WHERE id = ?"
	_, err = tx.ExecContext(ctx, query, bcID)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("Can't delete benchmark configuration: %v", err)
	}

	err = tx.Commit()
	return err
}

// InsertBenchmarkSummary creates summary for specific benchmark configuration
func (s *SQLite) InsertBenchmarkSummary(ctx context.Context, summary *katyusha.Summary, bcId int64) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("Can't start transaction: %v", err)
	}

	query := fmt.Sprintf("INSERT INTO benchmark_summary(%s,benchmark_configuration) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)", summaryFields)
	res, err := tx.ExecContext(ctx, query,
		summary.Start.Format(time.RFC3339),
		summary.End.Format(time.RFC3339),
		summary.TotalTime,
		summary.ReqCount,
		summary.SuccessReq,
		summary.FailReq,
		summary.DataTransfered,
		summary.ReqPerSec,
		summary.AvgReqTime,
		summary.MinReqTime,
		summary.MaxReqTime,
		summary.P50ReqTime,
		summary.P75ReqTime,
		summary.P90ReqTime,
		summary.P99ReqTime,
		bcId,
	)

	if err != nil {
		tx.Rollback()
		return fmt.Errorf("Can't create summary in database: %v", err)
	}

	smId, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("Can't get summary ID: %v", err)
	}

	query = "INSERT INTO errors(name,count,benchmark_summary) VALUES(?,?,?)"
	for name, count := range summary.Errors {
		_, err := tx.ExecContext(ctx, query, name, count, smId)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("Can't create error for summary: %v", err)
		}
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("Can't commit summary: %v", err)
	}

	return nil
}

// InsertBenchmarkConfiguration creates new benchmark configuration with unique url and description
func (s *SQLite) InsertBenchmarkConfiguration(ctx context.Context, benchParameters *katyusha.BenchmarkParameters, description string) (int64, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return 0, fmt.Errorf("Can't start transaction: %v", err)
	}

	query := fmt.Sprintf("INSERT INTO benchmark_configuration(%s) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)", benchmarkFields)

	res, err := tx.ExecContext(ctx, query,
		description,
		benchParameters.URL,
		benchParameters.Method,
		benchParameters.ReqCount,
		benchParameters.ConcurrentConns,
		boolToInt(benchParameters.SkipVerify),
		benchParameters.AbortAfter,
		benchParameters.CA,
		benchParameters.Cert,
		benchParameters.Key,
		benchParameters.Duration,
		benchParameters.KeepAlive,
		benchParameters.RequestDelay,
		benchParameters.ReadTimeout,
		benchParameters.WriteTimeout,
		benchParameters.Body)

	if err != nil {
		tx.Rollback()
		if sqliteErr, ok := err.(sqlite3.Error); ok {
			if sqliteErr.ExtendedCode == sqlite3.ErrConstraintUnique {
				bc, err := s.FindBenchmark(ctx, benchParameters.URL, description)
				if err != nil {
					return 0, err
				}

				if bc != nil {
					return 0, fmt.Errorf("Benchmark with provided URL and Description already exists (id %d)", bc.ID)
				}

				return int64(bc.ID), nil
			}
		}

		return 0, fmt.Errorf("Can't create benchmark configuration in database: %v", err)
	}

	bcID, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("Can't get benchmark configuration ID: %v", err)
	}

	query = "INSERT INTO headers(header,benchmark_configuration) VALUES(?,?)"

	for key, value := range benchParameters.Headers {
		header := strings.Join([]string{key, value}, ":")
		_, err := tx.ExecContext(ctx, query, header, bcID)
		if err != nil {
			tx.Rollback()
			return 0, fmt.Errorf("Can't create header: %v", err)
		}
	}

	query = "INSERT INTO parameters(parameter,benchmark_configuration) VALUES(?,?,?)"

	for _, params := range benchParameters.Parameters {
		var i int
		parameters := make([]string, len(params))
		for key, value := range params {
			keyValue := strings.Join([]string{key, value}, "=")
			parameters[i] = keyValue
			i++
		}

		parameterString := strings.Join(parameters, "&")
		_, err := tx.ExecContext(ctx, query, parameterString, bcID)
		if err != nil {
			tx.Rollback()
			return 0, fmt.Errorf("Can't create parameter: %v", err)
		}
	}

	err = tx.Commit()
	if err != nil {
		return 0, fmt.Errorf("Can't save benchmark configuration: %v", err)
	}

	return bcID, nil
}