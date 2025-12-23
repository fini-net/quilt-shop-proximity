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
just deps
just scrape
just stats
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
