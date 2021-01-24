package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/tmwalaszek/katyusha/katyusha"
)

var benchmarkConf string

// benchmarkCmd represents the benchmark command
var benchmarkCmd = &cobra.Command{
	Use:   "benchmark",
	Short: "Run HTTP benchmark",
	Run: func(cmd *cobra.Command, args []string) {
		var err error

		if viper.GetString("benchmark_config") != "" {
			viper.SetConfigFile(viper.GetString("benchmark_config"))

			if err := viper.MergeInConfig(); err != nil {
				log.Fatalf("Error on loading benchmark configuration %s: %v\n", viper.GetString("benchmark_config"), err)
			}
		}

		var benchmarkParams *katyusha.BenchmarkParameters

		benchmarkParams, err = benchmarkOptionsToStruct()
		if err != nil {
			cmd.Usage()
			log.Fatalf("Benchmark configuration error: %v", err)
		}

		runBenchmark(benchmarkParams)
	},
}

func runBenchmark(benchmarkParams *katyusha.BenchmarkParameters) {
	var err error
	description := viper.GetString("description")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	go func() {
		for {
			select {
			case <-c:
				log.Print("Received signal and will stop benchmark")
				cancel()
				return
			}
		}
	}()

	var inv *katyusha.Inventory
	if viper.GetBool("save") {
		inv, err = katyusha.NewInventory(viper.GetString("db"))
		if err != nil {
			log.Fatalf("Could not create inventory: %v", err)
		}
	}

	benchmark, err := katyusha.NewBenchmark(benchmarkParams)
	if err != nil {
		log.Fatalf("Error while creating benchmark: %v", err)
	}

	var bcID int64
	if viper.GetBool("save") {
		bcID, err = inv.InsertBenchmarkConfiguration(ctx, benchmarkParams, description)
		if err != nil {
			log.Fatalf("Error inserting benchmark configuration: %v", err)
		}
	}

	summary := benchmark.StartBenchmark(ctx)
	fmt.Println(summary)

	if viper.GetBool("save") {
		err = inv.InsertBenchmarkSummary(ctx, summary, "", bcID)
		if err != nil {
			log.Fatalf("Error saving summary: %v", err)
		}
	}
}

func benchmarkOptionsToStruct() (*katyusha.BenchmarkParameters, error) {
	host := viper.GetString("host")
	if host == "" {
		return nil, fmt.Errorf("Host not provided")
	}

	headers := katyusha.NewHeader()
	params := katyusha.NewParameter()

	for _, value := range viper.GetStringSlice("header") {
		err := headers.Set(value)
		if err != nil {
			return nil, err
		}
	}

	for _, value := range viper.GetStringSlice("parameter") {
		err := params.Set(value)
		if err != nil {
			return nil, err
		}
	}

	return &katyusha.BenchmarkParameters{
		URL:             host,
		TargetEndpoint:  viper.GetString("version_endpoint"),
		Method:          viper.GetString("method"),
		ReqCount:        viper.GetInt("requests"),
		AbortAfter:      viper.GetInt("abort"),
		ConcurrentConns: viper.GetInt("connections"),
		SkipVerify:      viper.GetBool("insecure"),
		CA:              viper.GetString("ca"),
		Cert:            viper.GetString("cert"),
		Key:             viper.GetString("key"),
		Duration:        viper.GetDuration("duration"),
		KeepAlive:       viper.GetDuration("keep_alive"),
		RequestDelay:    viper.GetDuration("request_delay"),
		ReadTimeout:     viper.GetDuration("read_timeout"),
		WriteTimeout:    viper.GetDuration("write_timeout"),
		Headers:         headers,
		Parameters:      params,
	}, nil
}

func init() {
	benchmarkCmd.Flags().StringP("benchmark_config", "b", "", "Benchmark configuration file")
	benchmarkCmd.Flags().String("description", "Default benchmark description", "Benchmark description used in database")
	benchmarkCmd.Flags().String("host", "", "Host")
	benchmarkCmd.Flags().StringP("method", "m", "", "HTTP Method")
	benchmarkCmd.Flags().StringP("ca", "c", "", "CA path")
	benchmarkCmd.Flags().StringP("cert", "F", "", "Cert path")
	benchmarkCmd.Flags().StringP("key", "K", "", "Key path")
	benchmarkCmd.Flags().StringP("version_endpoint", "E", "", "Version endpoint, the results from this endpoint will be saved in test summary")
	benchmarkCmd.Flags().BoolP("save", "s", false, "Save benchmark configuration and result")
	benchmarkCmd.Flags().BoolP("insecure", "i", false, "TLS Skip verify")
	benchmarkCmd.Flags().DurationP("duration", "d", time.Duration(0), "Benchmark duration")
	benchmarkCmd.Flags().DurationP("keep_alive", "k", time.Duration(0), "HTTP Keep Alive")
	benchmarkCmd.Flags().DurationP("request_delay", "D", time.Duration(0), "Request delay")
	benchmarkCmd.Flags().DurationP("read_timeout", "R", time.Duration(0), "Read Timeout")
	benchmarkCmd.Flags().DurationP("write_timeout", "W", time.Duration(0), "Write Timeout")
	benchmarkCmd.Flags().IntP("requests", "r", 0, "Requests count")
	benchmarkCmd.Flags().IntP("connections", "C", 0, "Concurrent connections")
	benchmarkCmd.Flags().IntP("abort", "a", 0, "Number of connections after which benchmark will be aborted")
	benchmarkCmd.Flags().StringSliceP("header", "H", nil, "Header, can be used multiple times")
	benchmarkCmd.Flags().StringSliceP("parameter", "P", nil, "HTTP parameters, can be used multiple times")

	viper.BindPFlags(benchmarkCmd.Flags())

	rootCmd.AddCommand(benchmarkCmd)
}
