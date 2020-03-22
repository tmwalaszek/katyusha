# Katyusha
Another HTTP Benchmarking tool

Katyusha is a HTTP benchmarking tool written in Golang.
It uses fasthttp as a HTTP library to make requests. It can basically do the same thing as wrk, siege and all other benchmarking tools out there. What is interesting in Katyusha is the possibility to save benchmark configuration in the database with the benchmarking results.
User can start a benchmark and tell Katyusha to save the configuration with the benchmark results. Benchmark parameters can be provided from command line options or from yaml files.
The only supported database is Sqlite3 at the moment.

## Build Katyusha
To compile simply type make build
```
tmw@MacBook-Pro-Tomasz katyusha % make build
go build -o bin/kt main.go
```

## Basic usage
```
% kt
Katyusha CLI is a HTTP benchmarking tool written in Golang.
It uses fasthttp library to make HTTP requests.

Usage:
  example [command]

Available Commands:
  benchmark   Run HTTP benchmark
  help        Help about any command
  inventory   Testcases and test summary inventory management

Flags:
      --config string   config file (default "/Users/tmw/.katyusha/katyusha.yaml")
      --db string       Inventory file location (default "/Users/tmw/.katyusha/inventory.db")
  -h, --help            help for example

Use "example [command] --help" for more information about a command.
subcommand is required
```

We are provided with two subcommands benchmark and inventory. Benchmark subcommand runs the benchmark and inventory provides basic options to view and add new benchmark. You can also save current benchmark with results in benchmark subcommand.

There are two global flags config and db. Config provides default benchmark options like default HTTP method or how many requests needs to be perform by default. Option db provides default location of sqlite3 database file, it can be overwritten by command line option or db entry in config file.
Example katyusha.yaml
```
---
method: "GET"
request_count: 1000
connections: 10
abort: 1000
db: "/Users/tmw/.katyusha/katyusha.db"
```

## Benchmark
Benchmark subcommand provides options to start and customize benchmark.
```
kt benchmark --help
Run HTTP benchmark

Usage:
  example benchmark [flags]

Flags:
  -a, --abort int                 Number of connections after which benchmark will be aborted
  -b, --benchmark_config string   Benchmark configuration file
  -c, --ca string                 CA path
  -F, --cert string               Cert path
  -C, --connections int           Concurrent connections
      --description string        Benchmark description used in database (default "Default benchmark description")
  -d, --duration duration         Benchmark duration
  -H, --header strings            Header, can be used multiple times
  -h, --help                      help for benchmark
      --host string               Host
  -I, --id int                    Benchmark configuration ID from database
  -i, --insecure                  TLS Skip verify
  -k, --keep_alive duration       HTTP Keep Alive
  -K, --key string                Key path
  -m, --method string             HTTP Method
  -P, --parameter strings         HTTP parameters, can be used multiple times
  -R, --read_timeout duration     Read Timeout
  -D, --request_delay duration    Request delay
  -r, --requests int              Requests count
  -S, --save                      Save benchamrk configuration and result
  -W, --write_timeout duration    Write Timeout

Global Flags:
      --config string   config file (default "/Users/tmw/.katyusha/katyusha.yaml")
      --db string       Inventory file location (default "/Users/tmw/.katyusha/inventory.db")
```

Host option is mandatory and does not have short version. You can also provide yaml files with the same parameters names.
As an example we will perform benchmark with 10 concurrent connections that will be 1m long.
```
kt benchmark --host http://127.0.0.1 -C 10 -d 1m
Benchmark summary:
  URL:					http://127.0.0.1
  Start:				2020-03-16 21:31:56.656238 +0100 CET m=+0.002339920
  End:					2020-03-16 21:32:56.674784 +0100 CET m=+60.019084360
  Test Duration:			1m0.016744532s
  Total Requests:			25541
  Requests per Second:			425.68
  Successful requests:			25541
  Failed requests:			0
  Data transfered:			14.9M
  Average Request time:			23.475697ms
  Min Request time:			4.145188ms
  Max Request time:			1.035877612s
  Errors:				map[]
```

We can also save the configuration of our benchmark with results. 
To do that we need --save flag and optional --description option describing benchmark.

```
kt benchmark --host http://127.0.0.1 -C 10 -d 1m --save --description "NGINX in docker benchmark"
Benchmark summary:
  URL:					http://127.0.0.1
  Start:				2020-03-16 21:36:25.138042 +0100 CET m=+0.009301094
  End:					2020-03-16 21:37:25.159564 +0100 CET m=+60.029022031
  Test Duration:			1m0.019720985s
  Total Requests:			25880
  Requests per Second:			431.33
  Successful requests:			25880
  Failed requests:			0
  Data transfered:			15.1M
  Average Request time:			23.169194ms
  Min Request time:			4.633608ms
  Max Request time:			1.036283466s
  Errors:				map[]
```

## Inventory
Inventory lets you view benchmark configurations along with benchmark summaries.
Benchmark configuration has one constraint URL and Description needs to be unique.

Lets search for our NGINX in docker benchmark
```
 kt inventory show benchmark 
Found 1 benchmarks
Benchmark [0]
ID:		 1
Description:	 NGINX in docker benchmark
Url:		 http://127.0.0.1
```

If we know the benchmark ID we can see the benchmark summaries

```
kt inventory show summary -i 1
Found 1 summaries for given benchmark
Summary 0
Benchmark summary:
  URL:					
  Start:				2020-03-16 21:36:25 +0100 CET
  End:					2020-03-16 21:37:25 +0100 CET
  Test Duration:			1m0.019720985s
  Total Requests:			25880
  Requests per Second:			431.33
  Successful requests:			25880
  Failed requests:			0
  Data transfered:			15.1M
  Average Request time:			23.169194ms
  Min Request time:			4.633608ms
  Max Request time:			1.036283466s
  Errors:				map[]
```

We can also view full benchmark configuration options
```
 kt inventory show benchmark -f
Found 1 benchmarks
Benchmark [0]
Benchmark configuration:
ID:				1
Description: 			NGINX in docker benchmark
URL:				http://127.0.0.1
Method:				GET
Request count:			1000
Abort:				1000
Concurrent connections:		10
SkipVerify:			false
CA:				
Cert:			
Key:			
Duration:			1m0s
Keep Alive: 			0s
Request Delay:			0s
Read Timeout:			0s
Write Timeout:			0s
Headers: 			map[]
Query args: 			map[]
Body: 		
```

And run this benchmark again and save results. After that we can check how many results we have for NGINX docker benchmark.
```
% kt benchmark -I 1 --save            
Benchmark summary:
  URL:					http://127.0.0.1
  Start:				2020-03-16 21:58:12.978071 +0100 CET m=+0.004683460
  End:					2020-03-16 21:59:13.009893 +0100 CET m=+60.034703830
  Test Duration:			1m0.030020448s
  Total Requests:			23290
  Requests per Second:			388.17
  Successful requests:			23290
  Failed requests:			0
  Data transfered:			13.6M
  Average Request time:			25.750586ms
  Min Request time:			4.400291ms
  Max Request time:			467.382393ms
  Errors:				map[]
	
tmw@MacBook-Pro-Tomasz katyusha % kt inventory show summary -i 1 
Found 2 summaries for given benchmark
Summary 0
Benchmark summary:
  URL:					
  Start:				2020-03-16 21:36:25 +0100 CET
  End:					2020-03-16 21:37:25 +0100 CET
  Test Duration:			1m0.019720985s
  Total Requests:			25880
  Requests per Second:			431.33
  Successful requests:			25880
  Failed requests:			0
  Data transfered:			15.1M
  Average Request time:			23.169194ms
  Min Request time:			4.633608ms
  Max Request time:			1.036283466s
  Errors:				map[]
	
Summary 1
Benchmark summary:
  URL:					
  Start:				2020-03-16 21:58:12 +0100 CET
  End:					2020-03-16 21:59:13 +0100 CET
  Test Duration:			1m0.030020448s
  Total Requests:			23290
  Requests per Second:			388.17
  Successful requests:			23290
  Failed requests:			0
  Data transfered:			13.6M
  Average Request time:			25.750586ms
  Min Request time:			4.400291ms
  Max Request time:			467.382393ms
  Errors:				map[]
```
