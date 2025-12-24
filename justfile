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

# build the Go application for California
[group('build')]
build-ca:
	cd shops-in-california && go build -o quilt-shop-scraper main.go

# download Go dependencies for California
[group('build')]
deps-ca:
	cd shops-in-california && go mod download && go mod tidy

# download Go dependencies for Virginia
[group('build')]
deps-va:
	cd shops-in-virginia && go mod download && go mod tidy

# run the scraper to fetch and store California quilt shop data
[group('run')]
scrape-ca:
	cd shops-in-california && go run main.go

# run the scraper to fetch and store Virginia quilt shop data
[group('run')]
scrape-va:
	cd shops-in-virginia && go run main.go

# clean build artifacts and database for California
[group('clean')]
clean-ca:
	rm -f shops-in-california/quilt-shop-scraper shops-in-california/quilt_shops.db

# clean build artifacts and database for Virginia
[group('clean')]
clean-va:
	rm -f shops-in-virginia/quilt-shop-scraper shops-in-virginia/quilt_shops.db shops-in-virginia/virginia-quilt-shops.pdf

# query the California database to show shop count by city
[group('query')]
stats-ca:
	@echo "{{BLUE}}Quilt shops by city (California):{{NORMAL}}"
	@sqlite3 shops-in-california/quilt_shops.db "SELECT city, COUNT(*) as count FROM quilt_shops GROUP BY city ORDER BY count DESC LIMIT 20;" -header -column

# query the Virginia database to show shop count by city
[group('query')]
stats-va:
	@echo "{{BLUE}}Quilt shops by city (Virginia):{{NORMAL}}"
	@sqlite3 shops-in-virginia/quilt_shops.db "SELECT city, COUNT(*) as count FROM quilt_shops GROUP BY city ORDER BY count DESC LIMIT 20;" -header -column

# show all shops in a specific city (California)
[group('query')]
city-ca CITY:
	@echo "{{BLUE}}Quilt shops in {{CITY}} (California):{{NORMAL}}"
	@sqlite3 shops-in-california/quilt_shops.db "SELECT name, address, phone, email FROM quilt_shops WHERE city = '{{CITY}}';" -header -column

# show all shops in a specific city (Virginia)
[group('query')]
city-va CITY:
	@echo "{{BLUE}}Quilt shops in {{CITY}} (Virginia):{{NORMAL}}"
	@sqlite3 shops-in-virginia/quilt_shops.db "SELECT name, address, phone, email, website FROM quilt_shops WHERE city = '{{CITY}}';" -header -column
