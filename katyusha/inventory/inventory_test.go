package inventory

import (
	"context"
	"github.com/tmwalaszek/katyusha/katyusha"
	"os"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestInventory(t *testing.T) {
	inv, err := NewInventory("pliczek.db")
	if err != nil {
		t.Fatalf("Can't create database file: %v", err)
	}

	defer os.Remove("pliczek.db")

	b := &katyusha.BenchmarkParameters{
		URL:             "http://katyusha.text",
		ConcurrentConns: 1,
		ReqCount:        1,
	}

	b.Headers = map[string]string{}
	b.Parameters = []map[string]string{}

	bcID, err := inv.InsertBenchmarkConfiguration(context.Background(), b, "Test description")
	if err != nil {
		t.Fatalf("Error inserting benchmark configuration: %v", err)
	}

	bcs, err := inv.FindBenchmarkByURL(context.Background(), "http://katyusha.text")
	if err != nil {
		t.Errorf("Error searching for URL http://katyusha.text: %v", err)
	}

	if diff := cmp.Diff(*b, bcs[0].BenchmarkParameters); diff != "" {
		t.Errorf("Benchmark parameters mismatch (-want +got):\n%s", diff)
	}

	start, err := time.Parse(time.RFC3339, "2020-03-07T18:57:46+01:00")
	if err != nil {
		t.Fatalf("Could not parse time: %v", err)
	}

	end, err := time.Parse(time.RFC3339, "2020-03-07T18:58:10+01:00")
	if err != nil {
		t.Fatalf("Could not parse time: %v", err)
	}

	summary := &katyusha.Summary{
		Start:          start,
		End:            end,
		TotalTime:      time.Duration(30 * time.Second),
		ReqCount:       8547,
		SuccessReq:     8547,
		FailReq:        0,
		DataTransfered: 5230764,
		ReqPerSec:      284.2,
		AvgReqTime:     time.Duration(352 * time.Millisecond),
		MinReqTime:     time.Duration(81 * time.Millisecond),
		MaxReqTime:     time.Duration(1 * time.Second),
		P50ReqTime:     time.Duration(50 * time.Second),
		P75ReqTime:     time.Duration(75 * time.Second),
		P90ReqTime:     time.Duration(90 * time.Second),
		P99ReqTime:     time.Duration(99 * time.Second),
		Errors:         make(map[string]int),
	}

	err = inv.InsertBenchmarkSummary(context.Background(), summary, bcID)
	if err != nil {
		t.Errorf("Error inserting benchmark summary: %v", err)
	}

	sm, err := inv.FindSummaryForBenchmark(context.Background(), bcs[0].ID)
	if err != nil {
		t.Fatalf("Coould not receive benchmark summary; %v", err)
	}

	var r katyusha.ReqTimes
	if diff := cmp.Diff(*summary, sm[0].Summary, cmpopts.IgnoreTypes(r)); diff != "" {
		t.Errorf("Benchmark summary mismatch (-want +got):\n%s", diff)
	}

}
