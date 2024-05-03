.PHONY=lint
lint: 
	golangci-lint run --issues-exit-code=0
