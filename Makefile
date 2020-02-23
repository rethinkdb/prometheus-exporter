build:
	go build -o rethinkdb-exporter

lint:
	goimports -w .
	go fmt ./...
	golangci-lint run ./...

dockerbuild:
	docker build --tag rethinkdb-exporter .

dockerrun:
	docker run --rm -it --name rethinkdb-exporter -p 9055:9055 rethinkdb-exporter --log.debug --stats.table-estimates
