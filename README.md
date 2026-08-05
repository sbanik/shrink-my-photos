# shrink-my-photos
Shrink screenshots for Low Disk space

## Build
```shell
GOOS=darwin GOARCH=arm64 go build -o shrinker main.go detector.go
```

## Mode: Stage

```shell
./shrinker -mode=stage -volume=/Volumes/ExternalSSD -out=/Users/yourusername/Desktop/OutputFolder
```

## Mode: Convert

```shell
./shrinker -mode=convert -out=/Users/yourusername/Desktop/OutputFolder
```

## Command Line Flags

|     Flag          | Flag Type | .env Key    |	Default   | Description                                                          |
|-------------------|-----------|-------------|-----------|----------------------------------------------------------------------|
| -volume           | string    | VOLUME_PATH | ""        | Source directory/external SSD volume path to scan.                   |
| -out              | string    | OUT_DIR     | ""        | Destination folder for manifest.json, error.log, and `screenshots/`. |
| -quality          | float64   | QUALITY     | 80.0      | WebP image encoding quality (range 1.0 to 100.0).                    |
| -workers          | int       | WORKERS     | CPU Count | Number of concurrent goroutines used during conversion.              |
| -delete-originals | bool      | —           | false     | Standalone mode: Deletes source files on SSD marked as `converted`.  |