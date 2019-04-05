.DEFAULT_GOAL := all

goland: solidity
	go get ./...
	go build -o bin/bc2server -i blockchainc2/cmd/bc2server
	go build -o bin/bc2agent -i blockchainc2/cmd/bc2agent

solidity:
	solc --bin --abi EventC2.sol --overwrite -o ./internal/pkg/BlockchainC2/
	abigen --abi ./internal/pkg/BlockchainC2/eventc2.abi --pkg BlockchainC2 --type EventC2 --out ./internal/pkg/BlockchainC2/eventc2.go --bin ./internal/pkg/BlockchainC2/EventC2.bin

clean:
	rm ./bin/*
	rm ./internal/pkg/BlockchainC2/eventc2.go
	rm ./internal/pkg/BlockchainC2/eventc2.bin
	
all: goland

