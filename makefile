all:
	go fmt *.go
	go build

run:
	sudo ./backdoor
