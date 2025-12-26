# quilt-shop-proximity

![GitHub Issues](https://img.shields.io/github/issues/fini-net/quilt-shop-proximity)
![GitHub Pull Requests](https://img.shields.io/github/issues-pr/fini-net/quilt-shop-proximity)
![GitHub License](https://img.shields.io/github/license/fini-net/quilt-shop-proximity)
![GitHub watchers](https://img.shields.io/github/watchers/fini-net/quilt-shop-proximity)

Tools for finding and analyzing quilt shops by proximity.

## Projects

### California Quilt Shops

See [shops-in-california/](shops-in-california/) for a Go application that scrapes California quilt shop data and stores it in a SQLite database.

Quick start:

```bash
just deps-ca
just scrape-ca
just stats-ca
```

### Virginia Quilt Shops

See [shops-in-virginia/](shops-in-virginia/) for a Go application that parses Virginia quilt shop data from VCQ PDF and stores it in a SQLite database.

Quick start:

```bash
just deps-va
just scrape-va
just stats-va
```

### Production Database

The production database at `data/quilt_shops.db` contains the merged data from California and Virginia, ready for use in the Godot application.

#### Database Schema

**quilt_shops table:**

- `id` - INTEGER PRIMARY KEY AUTOINCREMENT
- `name` - TEXT NOT NULL
- `address` - TEXT
- `city` - TEXT NOT NULL
- `state` - TEXT NOT NULL
- `phone` - TEXT
- `email` - TEXT
- `website` - TEXT
- `latitude` - REAL NOT NULL
- `longitude` - REAL NOT NULL
- `created_at` - DATETIME DEFAULT CURRENT_TIMESTAMP
- `geocode_attempted_at` - DATETIME

Indexes: `idx_city`, `idx_state`, `idx_coordinates`

**metadata table:**

- `key` - TEXT PRIMARY KEY
- `value` - TEXT
- `updated_at` - DATETIME DEFAULT CURRENT_TIMESTAMP

Current metadata:

- `version`: 1.0.0
- `total_shops`: 60

#### Database Verification

SHA256 checksum: `4fa4c3b80043de442a787dbe48e677883d6b18346c5412095325bc921eb4fc21`

Verify with:

```bash
shasum -a 256 -c data/quilt_shops.db.sha256
```

## Features

- Web scraping of quilt shop listings
- SQLite database storage
- Geographic proximity analysis (planned)
- Command-line tools via `just` recipes

## Contributing

- [Code of Conduct](.github/CODE_OF_CONDUCT.md)
- [Contributing Guide](.github/CONTRIBUTING.md) includes a step-by-step guide to our
  [development process](.github/CONTRIBUTING.md#development-process).

## Support & Security

- [Getting Support](.github/SUPPORT.md)
- [Security Policy](.github/SECURITY.md)
