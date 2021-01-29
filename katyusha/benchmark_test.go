package katyusha

import (
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"net/http"
	"net/http/httptest"

	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttputil"
)

func TestHeader(t *testing.T) {
	tt := []struct {
		headers  []string
		expected headers
		match    bool
	}{
		{
			headers: []string{"Location: http://www.google.pl/"},
			expected: headers{
				"Location": "http://www.google.pl/",
			},
			match: true,
		},
		{
			headers: []string{"Location: http://www.google.pl/", "Content-Type: text/html; charset=UTF-8"},
			expected: headers{
				"Location":     "http://www.google.pl/",
				"Content-Type": "text/html; charset=UTF-8",
			},
			match: true,
		},
		{
			headers: []string{"Location http://www"},
			expected: headers{
				"Location": "http://www",
			},
			match: false,
		},
	}

	for _, tc := range tt {
		t.Run(fmt.Sprintf("Header test %s", tc.headers), func(t *testing.T) {
			h := NewHeader()

			for _, header := range tc.headers {
				err := h.Set(header)
				if tc.match && err != nil {
					t.Fatalf("Can't prase %s error %v", tc.headers, err)
				}

				if !tc.match && err == nil {
					t.Fatalf("Parameter %s should not be parsed", tc.headers)
				}
			}

			if !tc.match {
				return
			}

			if diff := cmp.Diff(tc.expected, h); diff != "" {
				t.Fatalf("Header mismatch (-want +got):\n %s", diff)
			}
		})
	}
}

func TestParameter(t *testing.T) {
	tt := []struct {
		parameters []string
		expected   parameters
		match      bool
	}{
		{
			parameters: []string{"key1=val1&key2=val2"},
			expected: parameters{
				{
					"key1": "val1",
					"key2": "val2",
				},
			},
			match: true,
		},
		{
			parameters: []string{"key1=val1"},
			expected: parameters{
				{
					"key1": "val1",
				},
			},
			match: true,
		},
		{
			parameters: []string{"key1=val1&key2=val2", "key3=val3&key4=val4"},
			expected: parameters{
				{
					"key1": "val1",
					"key2": "val2",
				},
				{
					"key3": "val3",
					"key4": "val4",
				},
			},
			match: true,
		},
		{
			parameters: []string{"key1"},
			expected: parameters{
				{
					"key1": "",
				},
			},
			match: false,
		},
	}

	for _, tc := range tt {
		t.Run(fmt.Sprintf("Parameter %s", tc.parameters), func(t *testing.T) {
			p := NewParameter()

			for _, parameter := range tc.parameters {
				err := p.Set(parameter)

				if tc.match && err != nil {
					t.Fatalf("Can't prase %s error %v", tc.parameters, err)
				}

				if !tc.match && err == nil {
					t.Fatalf("Parameter %s should not be parsed", tc.parameters)
				}
			}

			if !tc.match {
				return
			}

			if diff := cmp.Diff(tc.expected, p); diff != "" {
				t.Fatalf("Parameter mismatch (-want +got):\n %s", diff)
			}
		})
	}
}

func TestOneRequest(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		// We will sleep one second to later check duration of the request
		time.Sleep(time.Second)
		testHeader := r.Header.Get("TEST")
		if testHeader == "" {
			t.Error("TEST header is empty")
		}

		if testHeader != "TEST" {
			t.Errorf("TEST header value should be TEST but it is %s", testHeader)
		}
		fmt.Fprintf(w, "Test")
	}

	server := httptest.NewServer(http.HandlerFunc(handler))
	defer server.Close()

	headers := make(map[string]string)
	headers["TEST"] = "TEST"
	req := &BenchmarkParameters{
		URL:             server.URL,
		ConcurrentConns: 1,
		ReqCount:        1,
		Headers:         headers,
	}

	benchmark, err := NewBenchmark(req)
	if err != nil {
		t.Errorf("Can't create benchmark: %v", err)
	}

	summary := benchmark.StartBenchmark(context.Background())

	if summary.SuccessReq != 1 {
		t.Errorf("Success calls should be 1 but it is %d\n", summary.SuccessReq)
	}

	if summary.FailReq != 0 {
		t.Errorf("FailReq should be zero but it is %d", summary.FailReq)
	}

	if summary.MaxReqTime != summary.MinReqTime {
		t.Errorf("With only one request min and max time should be equal but max is %v and min is %v", summary.MaxReqTime, summary.MinReqTime)
	}

	if summary.ReqPerSec < 1 {
		t.Errorf("ReqPerSec should be more than one but it is %f", summary.ReqPerSec)
	}
}

func TestGetRequestWithArgs(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		// We will test Arguments here
		val := r.URL.Query()["key"]
		if len(val) == 0 {
			t.Errorf("No key value in query arguments")
		}
		if val[0] != "value" {
			t.Errorf("We should have query args key set to value but it is %s", val)
		}
		fmt.Fprintf(w, "Test")
	}

	server := httptest.NewServer(http.HandlerFunc(handler))
	defer server.Close()

	req := &BenchmarkParameters{
		URL:             server.URL,
		Method:          "GET",
		ConcurrentConns: 1,
		ReqCount:        1,
		Parameters: []map[string]string{
			{"key": "value"},
		},
	}

	benchmark, err := NewBenchmark(req)
	if err != nil {
		t.Errorf("Can't create benchmark: %v\n", err)
	}

	benchmark.StartBenchmark(context.Background())
}

func TestPostRequestWithArgs(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		// We will test Arguments here
		err := r.ParseForm()
		if err != nil {
			t.Errorf("Can't parse request form: %v", err)
		}

		value := r.Form.Get("key")
		if value != "value" {
			t.Errorf("We should have query args key set to value but it is %s", value)
		}
		fmt.Fprintf(w, "Test")
	}

	server := httptest.NewServer(http.HandlerFunc(handler))
	defer server.Close()

	req := &BenchmarkParameters{
		URL:             server.URL,
		Method:          "POST",
		ConcurrentConns: 1,
		ReqCount:        1,
		Parameters: []map[string]string{
			{"key": "value"},
		},
	}

	benchmark, err := NewBenchmark(req)
	if err != nil {
		t.Errorf("Can't create benchmark: %v\n", err)
	}

	benchmark.StartBenchmark(context.Background())
}

func TestBodyRequest(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		b, err := ioutil.ReadAll(r.Body)
		if err != nil {
			t.Errorf("Can't read from body: %v", err)
			return
		}

		if string(b) != "TEST BODY" {
			t.Errorf("Wrong body, it should be 'TEST BODY' but it is %s", string(b))
		}
	}

	server := httptest.NewServer(http.HandlerFunc(handler))
	defer server.Close()

	req := &BenchmarkParameters{
		URL:             server.URL,
		Method:          "POST",
		ConcurrentConns: 1,
		ReqCount:        1,
		Body:            []byte("TEST BODY"),
	}

	benchmark, err := NewBenchmark(req)
	if err != nil {
		t.Errorf("Can't create benchmark: %v", err)
	}

	benchmark.StartBenchmark(context.Background())
}

func TestMultipleRequests(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		// We will sleep one second to later check duration of the request
		time.Sleep(time.Second)
		fmt.Fprintf(w, "Test")
	}

	server := httptest.NewServer(http.HandlerFunc(handler))
	defer server.Close()

	req := &BenchmarkParameters{
		URL:             server.URL,
		ConcurrentConns: 1,
		ReqCount:        5,
	}

	benchmark, err := NewBenchmark(req)
	if err != nil {
		t.Errorf("Can't create benchmark: %v", err)
	}

	summary := benchmark.StartBenchmark(context.Background())

	if summary.SuccessReq != 5 {
		t.Errorf("Success calls should be 5 but it is %d\n", summary.SuccessReq)
	}

	if summary.FailReq != 0 {
		t.Errorf("FailReq should be 0 but it is %d", summary.FailReq)
	}

	if summary.AvgReqTime > 1 && summary.AvgReqTime < 2 {
		t.Errorf("Average time should be higher than one but lesser than two but avg time is %d", summary.AvgReqTime)
	}
}

func PrepareInmemoryListenerBenchmark(reqCount int, connections int) (*Benchmark, *fasthttp.Server, error) {
	ln := fasthttputil.NewInmemoryListener()
	s := &fasthttp.Server{
		Handler: func(ctx *fasthttp.RequestCtx) {
			body := ctx.Request.Body()
			ctx.Write(body)
		},
	}

	go s.Serve(ln)

	req := &BenchmarkParameters{
		ConcurrentConns: connections,
		ReqCount:        reqCount,
	}

	benchmark, err := NewBenchmark(req)
	if err != nil {
		return nil, nil, err
	}

	benchmark.client = &fasthttp.Client{
		Dial: func(addr string) (net.Conn, error) {
			return ln.Dial()
		},
	}

	return benchmark, s, nil
}

func BenchmarkKatyusha(b *testing.B) {
	benchmarks := []struct {
		name        string
		reqCount    int
		connections int
	}{
		{"10 requets 1 connection", 10, 1},
		{"10 requests 10 connections", 10, 10},
		{"100 requests 10 connections", 100, 10},
		{"1000 requests 10 connections", 1000, 10},
		{"1000 requests 100 connections", 1000, 100},
		{"1000 requests 1000 connections", 1000, 1000},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			benchmark, s, err := PrepareInmemoryListenerBenchmark(bm.reqCount, bm.connections)
			if err != nil {
				b.Fatalf("Can't create Benchmark: %v\n", err)
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				benchmark.StartBenchmark(context.Background())
			}

			s.Shutdown()
		})
	}
}
