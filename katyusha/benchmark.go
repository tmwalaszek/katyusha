package katyusha

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"time"

	"code.cloudfoundry.org/bytefmt"
	"github.com/valyala/fasthttp"
)

// KatyushaName is set in fasthttp.Client
// I don't have better place for this variable now
const (
	KatyushaName = "Katyusha 1.0"
)

// RequestStat describe HTTP request status
type RequestStat struct {
	Start    time.Time
	End      time.Time
	Duration time.Duration

	BodySize int

	RetCode int
	Error   error
}

// Summary struct provides benchmark end results.
type Summary struct {
	URL string

	Start     time.Time
	End       time.Time
	TotalTime time.Duration

	ReqCount       int
	SuccessReq     int // Requests with return code 200
	FailReq        int // Requests with return code != 200
	DataTransfered int

	ReqPerSec float64 // Request per second

	AvgReqTime time.Duration // Average request time
	MinReqTime time.Duration // Min request time
	MaxReqTime time.Duration // Max request time

	Errors map[string]int // Errors map. Key is the HTTP response code.
}

func (s Summary) String() string {
	return fmt.Sprintf(`Benchmark summary:
  URL:					%s
  Start:				%v
  End:					%v
  Test Duration:			%v
  Total Requests:			%d
  Requests per Second:			%.2f
  Successful requests:			%d
  Failed requests:			%d
  Data transfered:			%s
  Average Request time:			%v
  Min Request time:			%v
  Max Request time:			%v
  Errors:				%v
	`, s.URL, s.Start, s.End, s.TotalTime, s.ReqCount, s.ReqPerSec, s.SuccessReq, s.FailReq, bytefmt.ByteSize(uint64(s.DataTransfered)),
		s.AvgReqTime, s.MinReqTime, s.MaxReqTime, s.Errors)
}

// BenchmarkParameters is used to configure HTTP requests
type BenchmarkParameters struct {
	URL    string
	Method string

	ReqCount        int
	AbortAfter      int
	ConcurrentConns int

	// TLS settings
	SkipVerify bool
	CA         string
	Cert       string
	Key        string

	Duration     time.Duration
	KeepAlive    time.Duration
	RequestDelay time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration

	Headers    map[string]string
	Parameters map[string]string

	Body []byte
}

// Benchmark is the main type.
// It stores the benchmark parameters and the fasthttp Client
// It provides methods to execute benchmark.
type Benchmark struct {
	BenchmarkParameters

	client *fasthttp.Client
}

// manageWorkers is run in separate goroutine
// It starts the workers goroutines and sends them signal to make a request via req channel
func (b *Benchmark) manageWorkers(ctx context.Context) (chan *RequestStat, chan struct{}) {
	statChan := make(chan *RequestStat, b.ConcurrentConns) // Workers will sends stats through this channel
	doneChan := make(chan struct{})

	go func() {
		doneChannels := make([]chan struct{}, b.ConcurrentConns)
		req := make(chan struct{})

		for i := 0; i < b.ConcurrentConns; i++ {
			doneChannels[i] = b.worker(req, statChan)
		}

		if b.Duration != time.Duration(0) {
			breakAfter := time.After(b.Duration)
		MAIN1:
			for {
				select {
				case <-ctx.Done():
					break MAIN1
				case <-breakAfter:
					break MAIN1
				default:
					req <- struct{}{}
				}
			}
		} else {
		MAIN2:
			for i := 0; i < b.ReqCount; i++ {
				select {
				case <-ctx.Done():
					break MAIN2
				default:
					req <- struct{}{}
				}
			}
		}

		for _, done := range doneChannels {
			done <- struct{}{}
		}

		doneChan <- struct{}{}
	}()

	return statChan, doneChan
}

// Worker make HTTP request when it gets notification on req channel
func (b *Benchmark) worker(req chan struct{}, statChan chan *RequestStat) chan struct{} {
	done := make(chan struct{})
	go func() {
		for {
			select {
			case <-done:
				return
			case <-req:
				stat := b.doRequest()
				statChan <- stat
				if b.RequestDelay != time.Duration(0) {
					time.Sleep(b.RequestDelay)
				}
			}
		}
	}()

	return done
}

// StartBenchmark runs the actual configured benchmark.
// It returns end results and can be start multiple times.
func (b *Benchmark) StartBenchmark(ctx context.Context) *Summary {
	var maxDuration, minDuration, avgDuration time.Duration
	var success, fail int
	var dataTransfered int
	var reqPerSecond float64

	errors := make(map[string]int)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	statChan, doneChan := b.manageWorkers(ctx)

	start := time.Now()
	// We are collecting results in this loop
MAIN:
	for {
		select {
		case stat := <-statChan:
			if maxDuration < stat.Duration {
				maxDuration = stat.Duration
			}

			if minDuration > stat.Duration || minDuration == 0 {
				minDuration = stat.Duration
			}

			if stat.RetCode == 200 && stat.Error == nil {
				success++
				dataTransfered += stat.BodySize
			} else {
				fail++
				var errString string
				if stat.Error != nil {
					errString = stat.Error.Error()
				} else {
					errString = fasthttp.StatusMessage(stat.RetCode)
				}

				if _, ok := errors[errString]; !ok {
					errors[errString] = 1
				} else {
					errors[errString]++
				}
			}

			avgDuration += stat.Duration // In the loop we will sum everything in this varaible. Ourside of the loop we will divde by reqs count

			if fail >= b.AbortAfter && b.AbortAfter != 0 {
				cancel()
				break MAIN
			}
		case <-doneChan:
			break MAIN
		case <-ctx.Done():
			break MAIN
		}
	}

	end := time.Now()
	totalTime := time.Since(start)

	if success != 0 {
		avgDuration = time.Duration(int64(avgDuration) / int64(success))
	} else {
		avgDuration = 0
	}

	reqCount := success + fail

	if totalTime > time.Duration(time.Second) {
		reqPerSecond = float64(success) / float64(totalTime/time.Second)

	} else {
		reqPerSecond = float64(success)
	}

	summary := &Summary{
		URL:            b.URL,
		Start:          start,
		End:            end,
		TotalTime:      totalTime,
		DataTransfered: dataTransfered,
		ReqPerSec:      reqPerSecond,
		ReqCount:       reqCount,
		SuccessReq:     success,
		FailReq:        fail,
		AvgReqTime:     avgDuration,
		MinReqTime:     minDuration,
		MaxReqTime:     maxDuration,
		Errors:         errors,
	}

	return summary
}

// NewBenchmark configure Benchmark and return its.
// It will setup fasthttp.Client and check benchmark requests parameters
func NewBenchmark(reqParams *BenchmarkParameters) (*Benchmark, error) {
	var tlsConfig tls.Config

	if reqParams.SkipVerify {
		tlsConfig.InsecureSkipVerify = reqParams.SkipVerify
	} else {
		if reqParams.CA != "" {
			caCert, err := ioutil.ReadFile(reqParams.CA)
			if err != nil {
				return nil, fmt.Errorf("Error reading CA file %s: %w", reqParams.CA, err)
			}

			caCertPool := x509.NewCertPool()
			caCertPool.AppendCertsFromPEM(caCert)

			tlsConfig.RootCAs = caCertPool

			if reqParams.Cert != "" && reqParams.Key != "" {
				cert, err := tls.LoadX509KeyPair(reqParams.Cert, reqParams.Key)
				if err != nil {
					return nil, fmt.Errorf("Could not load X509 key pair: %w", err)
				}

				tlsConfig.Certificates = []tls.Certificate{cert}
			}
		}
	}

	client := &fasthttp.Client{
		Name:                KatyushaName,
		MaxConnsPerHost:     reqParams.ConcurrentConns,
		ReadTimeout:         reqParams.ReadTimeout,
		WriteTimeout:        reqParams.WriteTimeout,
		MaxIdleConnDuration: reqParams.KeepAlive,
		TLSConfig:           &tlsConfig,
	}

	b := &Benchmark{
		BenchmarkParameters: *reqParams,
		client:              client,
	}

	return b, nil
}

// doRequest perform the HTTP request based on the paramters in BenchmarkParameters
func (b *Benchmark) doRequest() *RequestStat {
	req := fasthttp.AcquireRequest()
	resp := fasthttp.AcquireResponse()
	args := fasthttp.AcquireArgs()

	req.SetRequestURI(b.URL)
	req.Header.SetMethod(b.Method)

	// Set all Headers into Request
	for key, value := range b.Headers {
		req.Header.Add(key, value)
	}

	// Set args if any
	for key, value := range b.Parameters {
		args.Add(key, value)
	}

	if b.Method == fasthttp.MethodGet {
		reqArgs := req.URI().QueryArgs()
		args.CopyTo(reqArgs)
	} else {
		reqArgs := req.PostArgs()
		args.CopyTo(reqArgs)
	}

	if len(b.Body) != 0 && (b.Method == fasthttp.MethodPost || b.Method == fasthttp.MethodPut) {
		req.SetBody(b.Body)
	}

	start := time.Now()
	err := b.client.Do(req, resp)

	bodySize := len(resp.Body())

	end := time.Now()
	duration := time.Since(start)

	statusCode := resp.StatusCode()

	fasthttp.ReleaseRequest(req)
	fasthttp.ReleaseResponse(resp)
	fasthttp.ReleaseArgs(args)

	return &RequestStat{
		Start:    start,
		End:      end,
		Duration: duration,
		BodySize: bodySize,
		RetCode:  statusCode,
		Error:    err,
	}
}
