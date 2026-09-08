all:
	main/version.sh
	go build -o goclaw.out main/main.go main/version.go
	
run: all
	mkdir -p ./data/logs
	./goclaw.out -c ./main/goclaw.yaml
	
install: all
	cp goclaw.out /opt/claw/bin/claw
	
clean:
	@go clean -cache
	@rm -f goclaw.out
	

.PHONY: clean test