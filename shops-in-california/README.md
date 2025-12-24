# California Quilt Shops Scraper

A Go application that scrapes California quilt shop listings from ronatheribbiter.com and stores them in a SQLite database for proximity analysis.

## Features

- Scrapes quilt shop data from ronatheribbiter.com
- Extracts shop name, address, city, phone, and email
- Stores data in a SQLite database for easy querying
- Indexes on city and shop name for fast lookups

## Prerequisites

- Go 1.21 or later
- Internet connection to fetch the quilt shop listings

## Quick Start with Just Recipes

From the repository root, use these `just` recipes:

```bash
# Download Go dependencies
just deps-ca

# Run the scraper to fetch and store quilt shop data
just scrape-ca

# View statistics - count shops by city
just stats-ca

# Query shops in a specific city
just city-ca "San Francisco"

# Build the binary
just build-ca

# Clean build artifacts and database
just clean-ca
```

## Manual Usage

Alternatively, you can run commands directly from this directory:

```bash
# Download dependencies
go mod download

# Run the scraper
go run main.go
```

This will:

1. Fetch the California quilt shops listing
2. Parse the HTML content to extract shop information
3. Create a SQLite database file named `quilt_shops.db`
4. Insert all shop records into the database

## Database Schema

The `quilt_shops` table contains:

- `id` - Auto-incrementing primary key
- `name` - Shop name (required)
- `address` - Street address
- `city` - City name (required)
- `phone` - Phone number
- `email` - Email address
- `created_at` - Timestamp of when the record was created

Indexes are created on `city` and `name` fields for efficient querying.

## Querying the Database

### Using Just Recipes (Recommended)

From the repository root:

```bash
# View shop count by city
just stats-ca

# Find shops in a specific city
just city-ca "Berkeley"
```

### Using SQLite Directly

You can also query the database using any SQLite client:

```bash
sqlite3 quilt_shops.db "SELECT * FROM quilt_shops WHERE city = 'berkeley';"
```

Or use the SQLite CLI interactively:

```bash
sqlite3 quilt_shops.db
```

Example queries:

```sql
-- Count shops by city
SELECT city, COUNT(*) as shop_count FROM quilt_shops GROUP BY city ORDER BY shop_count DESC;

-- Find shops with email addresses
SELECT name, city, email FROM quilt_shops WHERE email IS NOT NULL AND email != '';

-- Search for shops by name
SELECT * FROM quilt_shops WHERE name LIKE '%Quilt%';
```

## Development

The application uses:

- [goquery](https://github.com/PuerkitoBio/goquery) for HTML parsing
- [modernc.org/sqlite](https://gitlab.com/cznic/sqlite) for pure Go SQLite database

## License

This project follows the license of the parent template repository.
