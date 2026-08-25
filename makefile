all:
	main/version.sh
	go build -o goclaw.out main/main.go main/version.go
	
run: all
	mkdir -p ./data/log
	./goclaw.out -c ./main/goclaw.yaml
clean:
	@go clean -cache
	@rm -f goclaw.out
	

.PHONY: clean test