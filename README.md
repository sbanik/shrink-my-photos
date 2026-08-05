# shrink-my-photos
Shrink screenshots for Low Disk space

## Build

```shell
go build -o shrinker ./cmd/shrinker
```

## Run Command

```shell
./shrinker -mode=stage -volume=/Volumes/ExternalSSD -out=/Users/yourusername/Desktop/OutputFolder
```

### Command Line Flags

|     Flag   | Flag Type | .env Key         |	Default | Description                                                          |
|------------|-----------|------------------|-----------|----------------------------------------------------------------------|
| -mode      | string    | VOLUME_PATH      | stage     | Execution mode: all (stage + convert), stage (scan & copy), convert (process staged files), delete (remove original files) |
| -volume    | string    | VOLUME_PATH      | ""        | Source directory/external SSD volume path to scan.                   |
| -out       | string    | OUT_DIR          | ""        | Destination folder for manifest.json, error.log, and `screenshots/`. |
| -quality   | float64   | QUALITY          | 80.0      | WebP image encoding quality (range 1.0 to 100.0).                    |
| -workers   | int       | WORKERS          | CPU Count | Number of concurrent goroutines used during conversion.              |

## Running Unit Tests:

Run the test suite across all files:

```shell
go test -v ./...
```

To include race condition checks:

```shell
go test -v -race ./...
```
