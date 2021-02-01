package katyusha

import (
	"context"
	_ "embed"
	"os"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

//go:embed test_files/summary.txt
var summaryText string

func TestInventory(t *testing.T) {
	inv, err := NewInventory("pliczek.db")
	if err != nil {
		t.Fatalf("Can't create database file: %v", err)
	}

	t.Cleanup(func() {
		err := inv.Close()
		if err != nil {
			t.Errorf("Error closing the inventory: %v", err)
		}
		err = os.Remove("pliczek.db")
		if err != nil {
			t.Errorf("Error removing inventory file: %v", err)
		}
	})

	header := NewHeader()
	header.Set("Content-Type: application/json")

	if header["Content-Type"] != "application/json" {
		t.Fatalf("Header Content-Type should be application/json but it is %s", header["Content-Type"])
	}

	param := NewParameter()
	param.Set("key=value")

	if param[0]["key"] != "value" {
		t.Fatalf("Param value for key should be value but it is %s\n", param[0]["key"])
	}

	b := &BenchmarkParameters{
		URL:             "http://katyusha.text",
		ConcurrentConns: 1,
		ReqCount:        1,
		Method:          "GET",
		Headers:         header,
		Parameters:      param,
	}

	bcID, err := inv.InsertBenchmarkConfiguration(context.Background(), b, "Test description")
	if err != nil {
		t.Fatalf("Error inserting benchmark configuration: %v", err)
	}

	// This should return an error
	_, err = inv.InsertBenchmarkConfiguration(context.Background(), b, "Test description")
	if err == nil {
		t.Fatalf("Second insert of benchmark with equal URL and description have to return error")
	}

	// Check parameters table
	params, err := inv.queryParametersTable(context.Background(), bcID)
	if err != nil {
		t.Fatalf("Error query parameters table: %v", err)
	}

	if diff := cmp.Diff(param, params); diff != "" {
		t.Fatalf("Params table mismatch (-want +got):\n%s", diff)
	}

	bcs, err := inv.FindBenchmarkByURL(context.Background(), "http://katyusha.text")
	if err != nil {
		t.Errorf("Error searching for URL http://katyusha.text: %v", err)
	}

	if diff := cmp.Diff(*b, bcs[0].BenchmarkParameters); diff != "" {
		t.Errorf("Benchmark parameters mismatch (-want +got):\n%s", diff)
	}

	bcs, err = inv.FindAllBenchmarks(context.Background())
	if err != nil {
		t.Errorf("Error via find all benchmarks: %v", err)
	}

	if diff := cmp.Diff(*b, bcs[0].BenchmarkParameters); diff != "" {
		t.Errorf("Benchmark parameters mismatch (-want +got):\n%s", diff)
	}

	bc, err := inv.FindBenchmarkByID(context.Background(), bcID)
	if err != nil {
		t.Fatalf("Error find benchmark via id")
	}

	if diff := cmp.Diff(*b, bc.BenchmarkParameters); diff != "" {
		t.Errorf("Benchmark parameters mismatch (-want +got):\n%s", diff)
	}

	bc, err = inv.FindBenchmark(context.Background(), "http://katyusha.text", "Test description")
	if err != nil {
		t.Fatalf("Error find benchmark via url and description: %v", err)
	}

	if diff := cmp.Diff(*b, bc.BenchmarkParameters); diff != "" {
		t.Errorf("Benchmark parameters mismatch (-want +got):\n%s", diff)
	}

	if bc.String() != summaryText {
		t.Errorf("BenchmarkConfiguration string mismatch: %s \n %s", bc.String(), summaryText)
	}

	start, err := time.Parse(time.RFC3339, "2020-03-07T18:57:46+01:00")
	if err != nil {
		t.Fatalf("Could not parse time: %v", err)
	}

	end, err := time.Parse(time.RFC3339, "2020-03-07T18:58:10+01:00")
	if err != nil {
		t.Fatalf("Could not parse time: %v", err)
	}

	errors := map[string]int{
		"error": 15,
	}
	summary := &Summary{
		Start:          start,
		End:            end,
		TotalTime:      time.Duration(30 * time.Second),
		ReqCount:       8562,
		SuccessReq:     8547,
		FailReq:        15,
		DataTransfered: 5230764,
		ReqPerSec:      284.2,
		AvgReqTime:     time.Duration(352 * time.Millisecond),
		MinReqTime:     time.Duration(81 * time.Millisecond),
		MaxReqTime:     time.Duration(1 * time.Second),
		P50ReqTime:     time.Duration(50 * time.Second),
		P75ReqTime:     time.Duration(75 * time.Second),
		P90ReqTime:     time.Duration(90 * time.Second),
		P99ReqTime:     time.Duration(99 * time.Second),
		Errors:         errors,
	}

	err = inv.InsertBenchmarkSummary(context.Background(), summary, "", bcID)
	if err != nil {
		t.Errorf("Error inserting benchmark summary: %v", err)
	}

	sm, err := inv.FindSummaryForBenchmark(context.Background(), bcs[0].ID)
	if err != nil {
		t.Fatalf("Coould not receive benchmark summary; %v", err)
	}

	var r ReqTimes
	if diff := cmp.Diff(*summary, sm[0].Summary, cmpopts.IgnoreTypes(r)); diff != "" {
		t.Errorf("Benchmark summary mismatch (-want +got):\n%s", diff)
	}

	err = inv.DeleteBenchmark(context.Background(), bcID)
	if err != nil {
		t.Fatalf("Error deleting benchmark: %v", err)
	}

}
