package inventory

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/mattn/go-sqlite3"
	"github.com/tmwalaszek/katyusha/katyusha"
	"strings"
	"time"
)

type DB struct {
	db *sql.DB
}

func (d *DB) deleteBenchmark(ctx context.Context, bcID int64) error {
	tx, err := d.db.Begin()
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

func (d *DB) queryParametersTable(ctx context.Context, bcId int64) (katyusha.Parameters, error) {
	query := "SELECT parameter FROM parameters where benchmark_configuration = ?"

	rows, err := d.db.QueryContext(ctx, query, bcId)
	if err != nil {
		return nil, err
	}

	results := katyusha.NewParameter()

	for rows.Next() {
		var value string
		err = rows.Scan(&value)

		if err != nil {
			break
		}

		err = results.Set(value)
		if err != nil {
			break
		}
	}

	if closeErr := rows.Close(); closeErr != nil {
		return nil, fmt.Errorf("Close rows error: %w", closeErr)
	}

	if err != nil {
		return nil, fmt.Errorf("parameter scan error: %w", err)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return results, nil
}

// querykeyValueTable is used to read table parameters and headers
func (d *DB) queryHeadersTable(ctx context.Context, bcId int64) (katyusha.Headers, error) {
	query := "SELECT header FROM headers WHERE benchmark_configuration = ?"

	rows, err := d.db.QueryContext(ctx, query, bcId)
	if err != nil {
		return nil, err
	}

	defer rows.Close()
	results := katyusha.NewHeader()

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

	rerr := rows.Close()
	if rerr != nil {
		return nil, err
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return results, nil
}

// queryErrors returns errors table for one summary table
func (d *DB) queryErrors(ctx context.Context, smId int64) (map[string]int, error) {
	query := "SELECT name,count FROM errors WHERE benchmark_summary = ?"

	errorsMap := make(map[string]int)
	rows, err := d.db.QueryContext(ctx, query, smId)
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

	rerr := rows.Close()
	if rerr != nil {
		return nil, err
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return errorsMap, nil
}

// querySummary return benchmarks summaries based on provided query and args
func (d *DB) querySummary(ctx context.Context, query string, args ...interface{}) ([]*BenchmarkSummary, error) {
	results := make([]*BenchmarkSummary, 0)

	rows, err := d.db.QueryContext(ctx, query, args...)
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
			Summary: katyusha.Summary{
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

		errorsMap, err := d.queryErrors(ctx, id)
		if err != nil {
			return nil, err
		}

		s.Errors = errorsMap
		results = append(results, s)
	}

	return results, nil
}

// queryBenchmark return benchmarks summaries based on provided query and args
func (d *DB) queryBenchmark(ctx context.Context, query string, args ...interface{}) ([]*BenchmarkConfiguration, error) {
	results := make([]*BenchmarkConfiguration, 0)

	rows, err := d.db.QueryContext(ctx, query, args...)
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

		headers, err := d.queryHeadersTable(ctx, id)
		if err != nil {
			return nil, err
		}

		parameters, err := d.queryParametersTable(ctx, id)
		if err != nil {
			return nil, err
		}

		bc := &BenchmarkConfiguration{
			ID:          id,
			Description: description,
			BenchmarkParameters: katyusha.BenchmarkParameters{
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

	rerr := rows.Close()
	if rerr != nil {
		return nil, err
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return results, nil
}

func (d *DB) insertBenchmarkSummary(ctx context.Context, summary *katyusha.Summary, bcId int64, insertSummary, insertErrors string) error {
	tx, err := d.db.Begin()
	if err != nil {
		return fmt.Errorf("Can't start transaction: %v", err)
	}

	res, err := tx.ExecContext(ctx, insertSummary,
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

	for name, count := range summary.Errors {
		_, err := tx.ExecContext(ctx, insertErrors, name, count, smId)
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

func (d *DB) insertBenchmarkConfiguration(ctx context.Context, benchParameters *katyusha.BenchmarkParameters, description, benchmarkInsert, headersInsert, parametersInsert string) (int64, error) {
	tx, err := d.db.Begin()
	if err != nil {
		return 0, fmt.Errorf("Can't start transaction: %v", err)
	}

	res, err := tx.ExecContext(ctx, benchmarkInsert,
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
				return 0, sqliteErr
			}
		}

		return 0, fmt.Errorf("Can't create benchmark configuration in database: %v", err)
	}

	bcID, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("Can't get benchmark configuration ID: %v", err)
	}

	for key, value := range benchParameters.Headers {
		header := strings.Join([]string{key, value}, ":")
		_, err := tx.ExecContext(ctx, headersInsert, header, bcID)
		if err != nil {
			tx.Rollback()
			return 0, fmt.Errorf("Can't create header: %v", err)
		}
	}

	for _, params := range benchParameters.Parameters {
		var i int
		parameters := make([]string, len(params))
		for key, value := range params {
			keyValue := strings.Join([]string{key, value}, "=")
			parameters[i] = keyValue
			i++
		}

		parameterString := strings.Join(parameters, "&")
		_, err := tx.ExecContext(ctx, parametersInsert, parameterString, bcID)
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