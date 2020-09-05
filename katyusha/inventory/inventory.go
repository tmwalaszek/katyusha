// Inventory supports only SQLite3
package inventory

import (
	"context"
	"fmt"
	"github.com/tmwalaszek/katyusha/katyusha"
	"strings"
)

type BenchmarkConfiguration struct {
	ID          int64
	Description string

	katyusha.BenchmarkParameters
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

	katyusha.Summary
}

type Inventory interface {
	FindAllBenchmarks(ctx context.Context) ([]*BenchmarkConfiguration, error)
	FindBenchmarkByID(ctx context.Context, ID int64) ([]*BenchmarkConfiguration, error)
	FindBenchmark(ctx context.Context, URL string, description string) (*BenchmarkConfiguration, error)
	FindBenchmarkByURL(ctx context.Context, URL string) ([]*BenchmarkConfiguration, error)
	FindSummaryForBenchmark(ctx context.Context, bcID int64) ([]*BenchmarkSummary, error)
	DeleteBenchmark(ctx context.Context, bcID int64) error
	InsertBenchmarkSummary(ctx context.Context, summary *katyusha.Summary, bcId int64) error
	InsertBenchmarkConfiguration(ctx context.Context, benchParameters *katyusha.BenchmarkParameters, description string) (int64, error)
}

func NewInventory(dbType string, connString string) (Inventory, error) {
	var inventory Inventory
	var err error
	if dbType == strings.ToLower("sqlite") {
		inventory, err = NewSQLite(connString)
	}

	return inventory, err
}
