package katyusha

var summaryFields = "start,end,duration,requests_count,success_req,fail_req,data_transfered,req_per_sec,avg_req_time,min_req_time,max_req_time"
var benchmarkFields = "description,url,method,requests_count,concurrent_conns,skip_verify,abort_after,ca,cert,key,duration,keep_alive,request_delay,read_timeout,write_timeout,body"

var schema = `CREATE TABLE benchmark_configuration (
    id INTEGER PRIMARY KEY,
    description TEXT,
    url TEXT,
    method TEXT,
    requests_count INTEGER,
    concurrent_conns INTEGER,
    skip_verify INTEGER,
    abort_after INTEGER,
    ca TEXT, 
    cert TEXT,
    key TEXT,
    duration TEXT,
    keep_alive TEXT,
    request_delay TEXT,
    read_timeout TEXT,
    write_timeout TEXT,
    body BLOB,
    UNIQUE(description,url)
);

CREATE TABLE headers (
    id INTEGER PRIMARY KEY,
    name TEXT,
    value TEXT,
    benchmark_configuration INTEGER,    

    FOREIGN KEY(benchmark_configuration) REFERENCES benchmark_configuration(id) 
);

CREATE TABLE parameters (
    id INTEGER PRIMARY KEY,
    name TEXT,
    value TEXT,
    benchmark_configuration INTEGER,

    FOREIGN KEY(benchmark_configuration) REFERENCES benchmark_configuration(id) 
);

CREATE TABLE benchmark_summary (
    id INTEGER PRIMARY KEY,
    start TEXT,
    end TEXT,
    duration TEXT,
    requests_count INTEGER,
    success_req INTEGER,
    fail_req INTEGER,
    data_transfered INTEGER,
    req_per_sec REAL,
    avg_req_time TEXT,
    min_req_time TEXT,
    max_req_time TEXT,
    benchmark_configuration INTEGER,

    FOREIGN KEY(benchmark_configuration) REFERENCES benchmark_configuration(id)
);

CREATE TABLE errors (
    id INTEGER PRIMARY KEY,
    name TEXT,
    count INTEGER,
    benchmark_summary INTEGER,

    FOREIGN KEY(benchmark_summary) REFERENCES benchmark_summary(id) ON DELETE CASCADE
);`
