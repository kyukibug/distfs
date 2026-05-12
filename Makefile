.PHONY: proto build test clean

proto:
	protec --go_out=. --go-gprc_out=. proto/distfs.proto

build:
	go build ./...

test:
	go test ./...

clean:
	rm -f proto/*.pb.go