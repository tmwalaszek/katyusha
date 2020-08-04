package katyusha

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"math"
	"math/rand"
	"regexp"
	"sort"
	"strings"
	"time"

	"code.cloudfoundry.org/bytefmt"
	"github.com/valyala/fasthttp"
)

// KatyushaName is set in fasthttp.Client
// I don't have better place for this variable now
const (
	KatyushaName = "Katyusha 1.0"
	headerRegexp = `^([\w-]+):\s*(.+)`
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

type ReqTimes []time.Duration

func (r ReqTimes) Len() int           { return len(r) }
func (r ReqTimes) Swap(i, j int)      { r[i], r[j] = r[j], r[i] }
func (r ReqTimes) Less(i, j int) bool { return r[i] < r[j] }

// a ReqTimes needs to be sorted
func percentil(a ReqTimes, q float64) time.Duration {
	n := (q / 100) * float64(len(a))
	p := int(math.Ceil(n))

	return a[p-1]
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

	P50ReqTime time.Duration // 50th percentile
	P75ReqTime time.Duration // 75th percentile
	P90ReqTime time.Duration // 90th percentile
	P99ReqTime time.Duration // 99th percentile

	StdDeviation float64 // Standard deviation

	Errors map[string]int // Errors map. Key is the HTTP response code.

	requestsTimes ReqTimes
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
  P50 Request time:			%v
  P75 Request time:			%v
  P90 Request time:			%v
  P99 Request time:			%v
  Errors:				%v
	`, s.URL, s.Start, s.End, s.TotalTime, s.ReqCount, s.ReqPerSec, s.SuccessReq, s.FailReq, bytefmt.ByteSize(uint64(s.DataTransfered)),
		s.AvgReqTime, s.MinReqTime, s.MaxReqTime, s.P50ReqTime, s.P75ReqTime, s.P90ReqTime, s.P99ReqTime, s.Errors)
}

type headers map[string]string

func NewHeader() headers {
	h := make(headers)
	return h
}

func (h headers) Set(value string) error {
	r := regexp.MustCompile(headerRegexp)
	matches := r.FindStringSubmatch(value)

	if len(matches) < 3 {
		return fmt.Errorf("Can't parse header %s", value)
	}

	h[matches[1]] = matches[2]
	return nil
}

type parameters []map[string]string

func NewParameter() parameters {
	p := make(parameters, 0)
	return p
}

// value needs to in format "key1=value2&key2=value2
func (p *parameters) Set(value string) error {
	paramsMap := make(map[string]string)
	parameters := strings.Split(value, "&")

	for _, param := range parameters {
		keyValue := strings.Split(param, "=")
		if len(keyValue) != 2 {
			return fmt.Errorf("Can't parse parameter %s", value)
		}

		paramsMap[keyValue[0]] = keyValue[1]
	}

	*p = append(*p, paramsMap)
	return nil
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
	Parameters []map[string]string

	Body []byte
}

// Benchmark is the main type.
// It stores the benchmark parameters and the fasthttp Client
// It provides methods to execute benchmark.
type Benchmark struct {
	BenchmarkParameters

	client *fasthttp.Client
}

// manageWorkers runs in a separate goroutine
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

	requestTimes := make(ReqTimes, 0)
	start := time.Now()
	// We are collecting results in this loop
MAIN:
	for {
		select {
		case stat := <-statChan:
			requestTimes = append(requestTimes, stat.Duration)

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

	sort.Sort(requestTimes)

	minDuration = requestTimes[0]
	maxDuration = requestTimes[len(requestTimes)-1]

	for _, reqTime := range requestTimes {
		avgDuration += reqTime
	}

	p50 := percentil(requestTimes, 50)
	p75 := percentil(requestTimes, 75)
	p90 := percentil(requestTimes, 90)
	p99 := percentil(requestTimes, 99)

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
		P50ReqTime:     p50,
		P75ReqTime:     p75,
		P90ReqTime:     p90,
		P99ReqTime:     p99,
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

	if len(b.Parameters) > 0 {
		rand.Seed(time.Now().Unix())
		r := rand.Intn(len(b.Parameters))

		// Set args if any
		for key, value := range b.Parameters[r] {
			args.Add(key, value)
		}
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
