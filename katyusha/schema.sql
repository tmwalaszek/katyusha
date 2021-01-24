CREATE TABLE benchmark_configuration (
    id INTEGER PRIMARY KEY,
    target_endpoint TEXT,
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
    header TEXT,
    benchmark_configuration INTEGER,    

    FOREIGN KEY(benchmark_configuration) REFERENCES benchmark_configuration(id)
    ON DELETE CASCADE
);

CREATE TABLE parameters (
    id INTEGER PRIMARY KEY,
    parameter TEXT,
    benchmark_configuration INTEGER,

    FOREIGN KEY(benchmark_configuration) REFERENCES benchmark_configuration(id) 
    ON DELETE CASCADE
);

CREATE TABLE benchmark_summary (
    id INTEGER PRIMARY KEY,
    description TEXT,
    target_version TEXT,
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
    p50_req_time TEXT,
    p75_req_time TEXT,
    p90_req_time TEXT,
    p99_req_time TEXT,
    benchmark_configuration INTEGER,

    FOREIGN KEY(benchmark_configuration) REFERENCES benchmark_configuration(id)
    ON DELETE CASCADE
);

CREATE TABLE errors (
    id INTEGER PRIMARY KEY,
    name TEXT,
    count INTEGER,
    benchmark_summary INTEGER,

    FOREIGN KEY(benchmark_summary) REFERENCES benchmark_summary(id) 
    ON DELETE CASCADE
);
