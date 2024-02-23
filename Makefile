.PHONY: update build all clean

update:
	go get -u github.com/scroll-tech/go-ethereum@more_api && go mod tidy

build:
	go build

clean:
	rm main
