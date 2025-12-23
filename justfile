# project justfile

import? '.just/compliance.just'
import? '.just/gh-process.just'
import? '.just/pr-hook.just'
import? '.just/shellcheck.just'

# list recipes (default works without naming it)
[group('info')]
list:
	just --list
	@echo "{{GREEN}}Your justfile is waiting for more scripts and snippets{{NORMAL}}"

# build the Go application
[group('build')]
build:
	cd shops-in-california && go build -o quilt-shop-scraper main.go

# download Go dependencies
[group('build')]
deps:
	cd shops-in-california && go mod download && go mod tidy

# run the scraper to fetch and store quilt shop data
[group('run')]
scrape:
	cd shops-in-california && go run main.go

# clean build artifacts and database
[group('clean')]
clean:
	rm -f shops-in-california/quilt-shop-scraper shops-in-california/quilt_shops.db

# query the database to show shop count by city
[group('query')]
stats:
	@echo "{{BLUE}}Quilt shops by city:{{NORMAL}}"
	@sqlite3 shops-in-california/quilt_shops.db "SELECT city, COUNT(*) as count FROM quilt_shops GROUP BY city ORDER BY count DESC LIMIT 20;" -header -column

# show all shops in a specific city
[group('query')]
city CITY:
	@echo "{{BLUE}}Quilt shops in {{CITY}}:{{NORMAL}}"
	@sqlite3 shops-in-california/quilt_shops.db "SELECT name, address, phone, email FROM quilt_shops WHERE city = '{{CITY}}';" -header -column
