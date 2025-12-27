# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Repository Purpose

A multi-state quilt shop proximity finder that scrapes quilt shop data, geocodes it, and provides it to a Godot mobile application for proximity analysis.

## Architecture Overview

This is a monorepo with three main components:

1. **Go scrapers** - Separate scraping applications for California and Virginia quilt shops
2. **Geocoding and merging** - Shared geocoding package and database merge tool
3. **Godot application** - Mobile app (skeleton, planned for future development)

### Data Flow

```text
shops-in-california/     shops-in-virginia/
       ↓                        ↓
    scrape                   scrape
       ↓                        ↓
   local DB                 local DB
       ↓                        ↓
    geocode                  geocode
       ↓                        ↓
       └────────┬───────────────┘
                ↓
            merge/
                ↓
          production DB
        (data/quilt_shops.db)
                ↓
          Godot app
```

### Directory Structure

- `shops-in-california/` - Go scraper for ronatheribbiter.com (web scraping with goquery)
- `shops-in-virginia/` - Go scraper for VCQ PDF (requires `pdftotext` CLI tool)
- `geocode/` - Shared Go package for OpenStreetMap Nominatim geocoding
- `merge/` - Go tool that combines CA/VA databases into unified production database
- `data/` - Production database with SHA256 checksum for verification
- `.godot/` - Godot 4.5 mobile application project (skeleton only)

### Key Dependencies

- **Go modules**: Each scraper has its own `go.mod` (no workspace configuration)
- **External tools**: `pdftotext` (from poppler-utils) required for Virginia scraper
- **Go libraries**:
  - `modernc.org/sqlite` - Pure Go SQLite (used in all Go tools)
  - `github.com/PuerkitoBio/goquery` - HTML parsing for California scraper
  - `github.com/chicks-net/quilt-shop-proximity/geocode` - Shared geocoding package

### Database Schema

All databases share the same schema with these key fields:

- **quilt_shops table**: `id`, `name`, `address`, `city`, `state`, `phone`, `email`, `website`, `latitude`, `longitude`, `created_at`, `geocode_attempted_at`
- **metadata table**: `key`, `value`, `updated_at` (only in merged database)
- **Indexes**: `idx_city`, `idx_state`, `idx_coordinates`

The production database (`data/quilt_shops.db`) only contains shops with successful geocoding (100% geocoded).

## Development Workflow

All development uses `just` recipes from the repository root.

### California Shops Workflow

```bash
just deps-ca              # Download Go dependencies
just scrape-ca            # Scrape ronatheribbiter.com
just geocode-ca           # Add GPS coordinates via Nominatim
just stats-ca             # Show shop count by city
just city-ca "Berkeley"   # Query specific city
just clean-ca             # Remove binary and database
```

### Virginia Shops Workflow

```bash
just deps-va              # Download Go dependencies
just scrape-va            # Download and parse VCQ PDF
just geocode-va           # Add GPS coordinates via Nominatim
just stats-va             # Show shop count by city
just city-va "Alexandria" # Query specific city
just clean-va             # Remove artifacts and database
```

### Geocoding Workflow

```bash
just geocode-all          # Geocode both CA and VA (runs geocode-ca and geocode-va)
just geocode-stats-ca     # Show geocoding success/failure counts
just geocode-stats-va     # Show geocoding success/failure counts
```

### Database Merging

```bash
just merge-databases      # Merge CA/VA into production database (only geocoded shops)
just stats-merged         # Show shop counts by state in merged database
just city-merged "City"   # Query specific city in merged database
```

### Testing

```bash
cd shops-in-california && go test -v  # Run California scraper tests
```

Virginia scraper does not currently have tests.

### Production Database Verification

```bash
shasum -a 256 -c data/quilt_shops.db.sha256  # Verify database integrity
```

## Important Implementation Details

### Geocoding

- Uses OpenStreetMap Nominatim API with 1-second rate limiting (enforced via ticker)
- Requires User-Agent header: `quilt-shop-proximity/1.0 (github.com/chicks-net/quilt-shop-proximity)`
- Shared package at `geocode/geocode.go` used by both scrapers
- Both scrapers support `geocode` subcommand: `go run main.go geocode`
- Failed geocoding attempts are tracked with `geocode_attempted_at` timestamp

### California Scraper

- Parses HTML using goquery to extract shop listings from pre-formatted text blocks
- Deduplicates shops using in-memory map (`seenShops`)
- Contact info pattern matching: phone numbers, email addresses
- Run from `shops-in-california/` directory

### Virginia Scraper

- Downloads PDF from vcq.org if not present locally
- Uses `pdftotext -layout` to preserve formatting
- State machine parser to identify city headers, shop names, addresses, and contact info
- More complex parsing due to PDF text extraction challenges
- Run from `shops-in-virginia/` directory

### Database Merge Tool

- Located in `merge/` directory
- Reads from `../shops-in-california/quilt_shops.db` and `../shops-in-virginia/quilt_shops.db`
- **Only includes shops with valid coordinates** (WHERE latitude IS NOT NULL AND longitude IS NOT NULL)
- Adds `state` column to each record (CA or VA)
- Creates production database at `merge/quilt_shops.db`
- Includes metadata table with version and total shop count
- After successful merge, copy `merge/quilt_shops.db` to `data/quilt_shops.db` and regenerate checksum

### Godot Application

- Godot 4.5 project configured for mobile (1125x2000 viewport)
- Currently skeleton only - no implementation yet
- Intended to read from `data/quilt_shops.db` for proximity analysis
- Project file: `project.godot`

## Common Development Patterns

### Adding a New State

1. Create new directory `shops-in-<state>/`
2. Implement Go scraper with same database schema
3. Add geocoding support via `geocode` package
4. Add `just` recipes for deps, scrape, geocode, stats, city, clean
5. Update `merge/main.go` to include new state database
6. Update documentation

### Updating Production Database

After scraping and geocoding changes:

```bash
just scrape-ca
just geocode-ca
just scrape-va
just geocode-va
just merge-databases
cp merge/quilt_shops.db data/quilt_shops.db
shasum -a 256 data/quilt_shops.db > data/quilt_shops.db.sha256
```

### Working with Individual Scrapers

Each scraper is a standalone Go application. Navigate to its directory to work directly:

```bash
cd shops-in-california
go run main.go           # scrape
go run main.go geocode   # geocode
go test -v               # test
```
