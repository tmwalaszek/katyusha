// Inventory supports only SQLite3
package katyusha

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/mattn/go-sqlite3"
)

var summaryFields = "start,end,duration,requests_count,success_req,fail_req,data_transfered,req_per_sec,avg_req_time,min_req_time,max_req_time,p50_req_time,p75_req_time,p90_req_time,p99_req_time"
var benchmarkFields = "description,url,method,requests_count,concurrent_conns,skip_verify,abort_after,ca,cert,key,duration,keep_alive,request_delay,read_timeout,write_timeout,body"

type BenchmarkConfiguration struct {
	ID          int64
	Description string

	BenchmarkParameters
}

func (b BenchmarkConfiguration) String() string {
	return fmt.Sprintf(`Benchmark configuration:
ID:				%d
Description: 			%s
URL:				%s
Method:				%s
Request count:			%d
Abort:				%d
Concurrent connections:		%d
SkipVerify:			%t
CA:				%s
Cert:			%s
Key:			%s
Duration:			%v
Keep Alive: 			%v
Request Delay:			%v
Read Timeout:			%v
Write Timeout:			%v
Headers: 			%v
Query args: 			%v
Body: 		%s
`, b.ID, b.Description, b.URL, b.Method, b.ReqCount, b.AbortAfter, b.ConcurrentConns, b.SkipVerify, b.CA, b.Cert, b.Key, b.Duration,
		b.KeepAlive, b.RequestDelay, b.ReadTimeout, b.WriteTimeout, b.Headers, b.Parameters, string(b.Body))
}

type BenchmarkSummary struct {
	ID int64

	Summary
}

type Inventory struct {
	db *sql.DB
}

// Sqlite3 does not provide bool type
// In Sqlite3 true is int 1 and false is int 0
func boolToInt(b bool) int {
	if b {
		return 1
	}

	return 0
}

// NetInventory creates and initiate new Inventory object with ready to use db handler
// If file does not exists it will try to create schema
func NewInventory(dbFile string) (*Inventory, error) {
	//go:embed schema.sql
	var schema string
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
		_, err = db.Exec(schema)
		if err != nil {
			return nil, fmt.Errorf("Could not create inventory schema: %w", err)
		}
	}

	return &Inventory{
		db: db,
	}, nil
}

func closeRows(rows *sql.Rows) error {
	err := rows.Close()
	if err != nil {
		return err
	}

	return rows.Err()
}

func (i *Inventory) queryParametersTable(ctx context.Context, bcID int64) (parameters, error) {
	query := "SELECT parameter FROM parameters where benchmark_configuration = ?"

	rows, err := i.db.QueryContext(ctx, query, bcID)
	if err != nil {
		return nil, err
	}

	defer rows.Close()
	results := NewParameter()

	for rows.Next() {
		var value string
		err = rows.Scan(&value)

		if err != nil {
			return nil, err
		}

		err = results.Set(value)
		if err != nil {
			return nil, err
		}
	}

	err = closeRows(rows)
	return results, err
}

// querykeyValueTable is used to read table parameters and headers
func (i *Inventory) queryHeadersTable(ctx context.Context, bcID int64) (headers, error) {
	query := "SELECT header FROM headers WHERE benchmark_configuration = ?"

	rows, err := i.db.QueryContext(ctx, query, bcID)
	if err != nil {
		return nil, err
	}

	defer rows.Close()
	results := NewHeader()

	for rows.Next() {
		var header string
		err = rows.Scan(&header)
		if err != nil {
			return nil, err
		}

		err = results.Set(header)
		if err != nil {
			return nil, err
		}
	}

	err = closeRows(rows)
	return results, err
}

// Find and return all benchmarks configurations
func (i *Inventory) FindAllBenchmarks(ctx context.Context) ([]*BenchmarkConfiguration, error) {
	query := fmt.Sprintf("SELECT id,%s FROM benchmark_configuration", benchmarkFields)
	bcs, err := i.queryBenchmark(ctx, query)

	return bcs, err
}

// Find and return Bencharm configuration using ID
func (i *Inventory) FindBenchmarkByID(ctx context.Context, ID int64) ([]*BenchmarkConfiguration, error) {
	query := fmt.Sprintf("SELECT id,%s FROM benchmark_configuration where id = ?", benchmarkFields)
	bcs, err := i.queryBenchmark(ctx, query, ID)

	return bcs, err
}

// FindBenchmark by two unique fields url and description
func (i *Inventory) FindBenchmark(ctx context.Context, URL string, description string) (*BenchmarkConfiguration, error) {
	query := fmt.Sprintf("SELECT id,%s FROM benchmark_configuration WHERE url = ? AND description = ?", benchmarkFields)
	bcs, err := i.queryBenchmark(ctx, query, URL, description)
	if err != nil {
		return nil, err
	}

	if len(bcs) != 1 {
		return nil, nil
	}

	return bcs[0], nil
}

// FindBenchmarkByURL by url
func (i *Inventory) FindBenchmarkByURL(ctx context.Context, URL string) ([]*BenchmarkConfiguration, error) {
	query := fmt.Sprintf("SELECT id,%s FROM benchmark_configuration WHERE url = ?", benchmarkFields)

	bcs, err := i.queryBenchmark(ctx, query, URL)
	return bcs, err
}

// queryErrors returns errors table for one summary table
func (i *Inventory) queryErrors(ctx context.Context, smId int64) (map[string]int, error) {
	query := "SELECT name,count FROM errors WHERE benchmark_summary = ?"

	errorsMap := make(map[string]int)
	rows, err := i.db.QueryContext(ctx, query, smId)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		var name string
		var count int

		err = rows.Scan(&name, &count)
		if err != nil {
			return nil, err
		}

		errorsMap[name] = count
	}

	err = closeRows(rows)
	return errorsMap, err
}

// FindSummaryForBenchmark return summaries for benchmark
func (i *Inventory) FindSummaryForBenchmark(ctx context.Context, bcID int64) ([]*BenchmarkSummary, error) {
	query := fmt.Sprintf("SELECT id,%s FROM benchmark_summary WHERE benchmark_configuration = ?", summaryFields)

	summaries, err := i.querySummary(ctx, query, bcID)
	if err != nil {
		return nil, err
	}

	return summaries, err
}

// querySummary return benchmarks summaries based on provided query and args
func (i *Inventory) querySummary(ctx context.Context, query string, args ...interface{}) ([]*BenchmarkSummary, error) {
	results := make([]*BenchmarkSummary, 0)

	rows, err := i.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		var id int64
		var reqCount, successReq, failReq, dataTransfered int
		var start, end string
		var duration, avgReq, minReq, maxReq time.Duration
		var p50Req, p75Req, p90Req, p99Req time.Duration
		var reqPerSec float64

		err = rows.Scan(&id, &start, &end, &duration, &reqCount, &successReq, &failReq, &dataTransfered,
			&reqPerSec, &avgReq, &minReq, &maxReq, &p50Req, &p75Req, &p90Req, &p99Req)
		if err != nil {
			return nil, err
		}

		timeStart, err := time.Parse(time.RFC3339, start)
		if err != nil {
			return nil, err
		}

		timeEnd, err := time.Parse(time.RFC3339, end)
		if err != nil {
			return nil, err
		}

		s := &BenchmarkSummary{
			ID: id,
			Summary: Summary{
				Start:          timeStart,
				End:            timeEnd,
				TotalTime:      duration,
				ReqCount:       reqCount,
				SuccessReq:     successReq,
				FailReq:        failReq,
				DataTransfered: dataTransfered,
				ReqPerSec:      reqPerSec,
				AvgReqTime:     avgReq,
				MinReqTime:     minReq,
				MaxReqTime:     maxReq,
				P50ReqTime:     p50Req,
				P75ReqTime:     p75Req,
				P90ReqTime:     p90Req,
				P99ReqTime:     p99Req,
			},
		}

		errorsMap, err := i.queryErrors(ctx, id)
		if err != nil {
			return nil, err
		}

		s.Errors = errorsMap
		results = append(results, s)
	}

	return results, nil
}

// queryBenchmark return benchmarks summaries based on provided query and args
func (i *Inventory) queryBenchmark(ctx context.Context, query string, args ...interface{}) ([]*BenchmarkConfiguration, error) {
	results := make([]*BenchmarkConfiguration, 0)

	rows, err := i.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		var id int64
		var reqCount, abortAfter, concurrentConns int
		var description, url, method, ca, cert, key string
		var duration, keepAlive, requestDelay, readTimeout, writeTimeout time.Duration
		var skipVerify bool
		var body []byte

		err = rows.Scan(&id, &description, &url, &method, &reqCount, &concurrentConns,
			&skipVerify, &abortAfter, &ca, &cert, &key, &duration, &keepAlive, &requestDelay,
			&readTimeout, &writeTimeout, &body)
		if err != nil {
			return nil, err
		}

		headers, err := i.queryHeadersTable(ctx, id)
		if err != nil {
			return nil, err
		}

		parameters, err := i.queryParametersTable(ctx, id)
		if err != nil {
			return nil, err
		}

		bc := &BenchmarkConfiguration{
			ID:          id,
			Description: description,
			BenchmarkParameters: BenchmarkParameters{
				URL:             url,
				Method:          method,
				ReqCount:        reqCount,
				AbortAfter:      abortAfter,
				ConcurrentConns: concurrentConns,
				SkipVerify:      skipVerify,
				CA:              ca,
				Cert:            cert,
				Key:             key,
				Duration:        duration,
				KeepAlive:       keepAlive,
				RequestDelay:    requestDelay,
				ReadTimeout:     readTimeout,
				WriteTimeout:    writeTimeout,
				Headers:         headers,
				Parameters:      parameters,
				Body:            body,
			},
		}

		results = append(results, bc)
	}

	err = closeRows(rows)
	return results, nil
}

// DeleteBenchmark deletes benchmark configuration and all associated summaries
func (i *Inventory) DeleteBenchmark(ctx context.Context, bcID int64) error {
	tx, err := i.db.Begin()
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
func (i *Inventory) InsertBenchmarkSummary(ctx context.Context, summary *Summary, bcId int64) error {
	tx, err := i.db.Begin()
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
func (i *Inventory) InsertBenchmarkConfiguration(ctx context.Context, benchParameters *BenchmarkParameters, description string) (int64, error) {
	tx, err := i.db.Begin()
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
				bc, err := i.FindBenchmark(ctx, benchParameters.URL, description)
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
