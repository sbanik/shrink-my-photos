# shrink-my-photos
Shrink screenshots for Low Disk space

## Setup

### Environment Variables
Setup .env file

```
# Execution mode
MODE=stage

# Path to your target screenshots directory
VOLUME_PATH=

# Path to your target output directory
OUT_DIR=

# Compression Quality (0 - 100)
QUALITY=80

# Maximum concurrent workers (leave blank to default to CPU core count)
WORKERS= 

# Comma-separated extensions to scan (e.g. png,jpg,jpeg)
ALLOWED_TYPES=png,jpg,jpeg

# Clean staged folder and manifest before staging
CLEAN_STAGED=true

# Minimum space savings percentage threshold
MIN_SAVINGS=10.0
```

## Build

```shell
go build -o shrinker ./cmd/shrinker
```

## Run Command

```shell
./shrinker -mode=stage -volume=/Volumes/ExternalSSD -out=/Users/yourusername/Desktop/OutputFolder
```

### Command Line Flags

|     Flag     | Flag Type | .env Key     |	Default  | Description                                                          |
|--------------|-----------|--------------|-----------|----------------------------------------------------------------------|
| -mode        | string    | MODE         | stage     | Execution mode: all (stage + convert), stage (scan & copy), convert (process staged files), delete (remove original files) |
| -volume      | string    | VOLUME_PATH  | ""        | Source directory/external SSD volume path to scan.                   |
| -out         | string    | OUT_DIR      | ""        | Destination folder for manifest.json, error.log, and `screenshots/`. |
| -quality     | float64   | QUALITY      | 80.0      | WebP image encoding quality (range 1.0 to 100.0).                    |
| -workers     | int       | WORKERS      | CPU Count | Number of concurrent goroutines used during conversion.              |
| -clean       | bool      | CLEAN_STAGED | false     | Clean staged folder and manifest before staging.                     |
| -min-savings | float64   | MIN_SAVINGS  | 10.0      | Minimum space savings percentage threshold (e.g. 5.0 for 5%)         |

## Running Unit Tests:

Run the test suite across all files:

```shell
go test -v ./...
```

To include race condition checks:

```shell
go test -v -race ./...
```

Running benchmarks

```shell
go test -bench=. ./...
```

Cleaning Go Test cache

```shell
go clean -testcache
```