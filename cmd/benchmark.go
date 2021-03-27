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

// benchmarkCmd represents the benchmark command
func NewBenchmarkCmd(bench Benchmark, inv Inventory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "benchmark",
		Short: "Run HTTP benchmark",
		Run: func(cmd *cobra.Command, args []string) {
			logger := log.New(cmd.ErrOrStderr(), "", log.LstdFlags)
			viper.BindPFlag("save", cmd.Flags().Lookup("save"))
			var err error

			if viper.GetString("benchmark_config") != "" {
				viper.SetConfigFile(viper.GetString("benchmark_config"))

				if err := viper.MergeInConfig(); err != nil {
					logger.Fatalf("Error on loading benchmark configuration %s: %v\n", viper.GetString("benchmark_config"), err)
				}
			}

			var benchmarkParams *katyusha.BenchmarkParameters

			benchmarkParams, err = benchmarkOptionsToStruct()
			if err != nil {
				cmd.Usage()
				logger.Fatalf("Benchmark configuration error: %v", err)
			}

			//runBenchmark(benchmarkParams)
			description := viper.GetString("description")

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			c := make(chan os.Signal, 1)
			signal.Notify(c, os.Interrupt)

			go func() {
				<-c
				logger.Print("Received signal and will stop benchmark")
				cancel()
			}()

			var inv *katyusha.Inventory
			if viper.GetBool("save") {
				inv, err = katyusha.NewInventory(viper.GetString("db"))
				if err != nil {
					logger.Fatalf("Could not create inventory: %v", err)
				}
			}

			benchmark, err := katyusha.NewBenchmark(benchmarkParams)
			if err != nil {
				logger.Fatalf("Error while creating benchmark: %v", err)
			}

			var bcID int64
			if viper.GetBool("save") {
				bcID, err = inv.InsertBenchmarkConfiguration(ctx, benchmarkParams, description)
				if err != nil {
					logger.Fatalf("Error inserting benchmark configuration: %v", err)
				}
			}

			summary := benchmark.StartBenchmark(ctx)
			fmt.Println(summary)

			if viper.GetBool("save") {
				err = inv.InsertBenchmarkSummary(ctx, summary, "", bcID)
				if err != nil {
					logger.Fatalf("Error saving summary: %v", err)
				}
			}
		},
	}

	cmd.Flags().StringP("benchmark_config", "b", "", "Benchmark configuration file")
	cmd.Flags().String("description", "Default benchmark description", "Benchmark description used in database")
	cmd.Flags().String("host", "", "Host")
	cmd.Flags().StringP("method", "m", "", "HTTP Method")
	cmd.Flags().StringP("ca", "c", "", "CA path")
	cmd.Flags().StringP("cert", "F", "", "Cert path")
	cmd.Flags().StringP("key", "K", "", "Key path")
	cmd.Flags().StringP("version_endpoint", "E", "", "Version endpoint, the results from this endpoint will be saved in test summary")
	cmd.Flags().BoolP("save", "s", false, "Save benchmark configuration and result")
	cmd.Flags().BoolP("insecure", "i", false, "TLS Skip verify")
	cmd.Flags().DurationP("duration", "d", time.Duration(0), "Benchmark duration")
	cmd.Flags().DurationP("keep_alive", "k", time.Duration(0), "HTTP Keep Alive")
	cmd.Flags().DurationP("request_delay", "D", time.Duration(0), "Request delay")
	cmd.Flags().DurationP("read_timeout", "R", time.Duration(0), "Read Timeout")
	cmd.Flags().DurationP("write_timeout", "W", time.Duration(0), "Write Timeout")
	cmd.Flags().IntP("requests", "r", 0, "Requests count")
	cmd.Flags().IntP("connections", "C", 0, "Concurrent connections")
	cmd.Flags().IntP("abort", "a", 0, "Number of connections after which benchmark will be aborted")
	cmd.Flags().StringSliceP("header", "H", nil, "Header, can be used multiple times")
	cmd.Flags().StringSliceP("parameter", "P", nil, "HTTP parameters, can be used multiple times")

	viper.BindPFlags(cmd.Flags())

	return cmd
}

func benchmarkOptionsToStruct() (*katyusha.BenchmarkParameters, error) {
	host := viper.GetString("host")
	if host == "" {
		return nil, fmt.Errorf("host not provided")
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

