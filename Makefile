build:
	go build -o bin/kt main.go

test:
	go test -cover -v ./...

benchmark:
	go test -bench=. ./katyusha
