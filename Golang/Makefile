clean:
	go mod tidy

local-run:
	go run main.go

test:
	go test -v ./... -cover 2>&1 | grep -v "no test files"



run: clean local-run
