.PHONY: bmrcl irctc all clean install

# Install dependencies
install:
	go mod download

# Scrape BMRCL data
bmrcl:
	go run cmd/bmrcl-scraper/main.go -output output/bmrcl

# Scrape IRCTC data
irctc:
	go run cmd/irctc-scraper/main.go -output output/irctc

# Scrape all data
all: bmrcl irctc

# Clean output directories
clean:
	rm -rf output/

# Build binaries
build:
	go build -o bin/bmrcl-scraper cmd/bmrcl-scraper/main.go
	go build -o bin/irctc-scraper cmd/irctc-scraper/main.go


