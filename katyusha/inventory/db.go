package inventory

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/tmwalaszek/katyusha/katyusha"
	"time"
)

type DB struct {
	db *sql.DB
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