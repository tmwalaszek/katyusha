package inventory

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/mattn/go-sqlite3"
	"github.com/tmwalaszek/katyusha/katyusha"
	"os"
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
	query := "DELETE FROM benchmark_configuration WHERE id = ?"
	err := s.deleteBenchmark(ctx, bcID, query)

	return err
}

// InsertBenchmarkSummary creates summary for specific benchmark configuration
func (s *SQLite) InsertBenchmarkSummary(ctx context.Context, summary *katyusha.Summary, bcID int64) error {
	insertSummaryQuery := fmt.Sprintf("INSERT INTO benchmark_summary(%s,benchmark_configuration) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)", summaryFields)
	insertErrorsQuery := "INSERT INTO errors(name,count,benchmark_summary) VALUES(?,?,?)"

	err := s.insertBenchmarkSummary(ctx, summary, bcID, insertSummaryQuery, insertErrorsQuery)
	return err
}

// InsertBenchmarkConfiguration creates new benchmark configuration with unique url and description
func (s *SQLite) InsertBenchmarkConfiguration(ctx context.Context, benchParameters *katyusha.BenchmarkParameters, description string) (int64, error) {
	benchmarkInsert := fmt.Sprintf("INSERT INTO benchmark_configuration(%s) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)", benchmarkFields)
	headersInsert := "INSERT INTO headers(header,benchmark_configuration) VALUES(?,?)"
	parametersInsert := "INSERT INTO parameters(parameter,benchmark_configuration) VALUES(?,?,?)"

	id, err := s.insertBenchmarkConfiguration(ctx, benchParameters, description, benchmarkInsert, headersInsert, parametersInsert)
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

	return id, err
}