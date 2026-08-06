# shrink-my-photos

`shrink-my-photos` recursively converts selected image types to WebP without moving ordinary files out of their source folders. It writes converted files below `VOLUME_PATH/processed`, preserving the original directory layout, and keeps its manifest and logs in the operating system's user configuration folder.

## What it does

```text
VOLUME_PATH/
├── event-a/photo.jpg              → processed/event-a/photo.webp → event-a/photo.webp
├── event-a/discarded/             → emptied by sync, removed after conversion
└── event-b/photo.png              → processed/event-b/photo.webp
```

- Files with the allowed extensions are discovered recursively.
- The first copy of a byte-identical image remains in place; later copies are moved to a `discarded` folder beside their original directory.
- You may move any tracked image into its folder's `discarded` directory yourself.
- `sync` permanently removes files in `discarded` and updates their manifest records.
- Conversion leaves source files untouched unless `DELETE_ORIGINALS` is enabled. With the default processed workspace, verified WebPs move beside deleted originals; an explicit `PROCESSED_PATH` remains the final output location.
- `processed` and `discarded` directories are excluded from future scans.

## Quick start

```shell
go build -o shrinker ./cmd/shrinker

# Discover files, pause for review, then convert. Original deletion requires confirmation.
./shrinker -mode=all -volume=/Volumes/ExternalSSD/Photos
```

`all` pauses after discovery so you can move unwanted images into `discarded` folders, then always asks before deleting converted originals. When using the default workspace, verified WebP files then move into the original folders.

For an unattended, destructive workflow:

```shell
./shrinker -mode=auto -volume=/Volumes/ExternalSSD/Photos
```

`auto` runs discovery, synchronization, and conversion without review prompts. Set `DELETE_ORIGINALS=true` to also delete converted originals. If its temporary workspace is too small, it requests a validated fallback directory before converting. Use it only after validating the volume and configuration with `all`.

To review manual discards before conversion:

```shell
./shrinker -mode=stage -volume=/Volumes/ExternalSSD/Photos

# Move unwanted tracked files to a sibling discarded directory, for example:
# /Volumes/ExternalSSD/Photos/Trip/discarded/image.jpg

./shrinker -mode=sync -volume=/Volumes/ExternalSSD/Photos
./shrinker -mode=convert -volume=/Volumes/ExternalSSD/Photos
```

`sync` permanently removes the files in `discarded`. After a conversion run, empty `discarded` directories are removed.

## Modes

| Mode      | Description                                                                                                                                                                                          |
| --------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `all`     | Discover images, pause for manual review, optionally delete listed hidden files, sync, convert, and always ask before deleting originals. This is the default.                                       |
| `auto`    | Run `stage → sync → convert`; delete originals only when `DELETE_ORIGINALS=true`.                                                                                                                    |
| `stage`   | Recursively discover candidates, list hidden files, ignore camera-photo folders, and move byte-identical duplicates to per-folder `discarded` directories. No ordinary image is copied or converted. |
| `sync`    | Permanently delete files in every `discarded` directory and update the manifest.                                                                                                                     |
| `convert` | Convert pending images to `VOLUME_PATH/processed`, preserving relative paths. Source images remain unchanged.                                                                                        |
| `delete`  | Permanently delete originals whose manifest status is `converted`, then move verified WebP files into their original folders.                                                                        |

## Configuration

Flags override values in a `.env` file in the current directory.

| Flag                   | Environment variable  | Default                 | Description                                                                                                                              |
| ---------------------- | --------------------- | ----------------------- | ---------------------------------------------------------------------------------------------------------------------------------------- |
| `-mode`                | `MODE`                | `all`                   | One of `auto`, `all`, `stage`, `sync`, `convert`, or `delete`.                                                                           |
| `-volume`              | `VOLUME_PATH`         | —                       | Source root; required for every mode.                                                                                                    |
| `-processed`           | `PROCESSED_PATH`      | `VOLUME_PATH/processed` | Output workspace; when explicitly supplied, it remains the final WebP destination.                                                       |
| `-state`               | `STATE_DIR`           | OS user config folder   | Override the state location, useful for testing or portable runs.                                                                        |
| `-types`               | `ALLOWED_TYPES`       | `png,jpg,jpeg`          | Comma-separated extensions to process.                                                                                                   |
| `-quality`             | `QUALITY`             | automatic               | Preferred WebP quality from 55 through 90. The app uses 80–55 automatically when omitted, or evaluates the supplied value and ±5 points. |
| `-target-size`         | `TARGET_SIZE_KB`      | `200` KiB               | Ideal output size for files larger than the small-file threshold.                                                                        |
| `-small-file-size`     | `SMALL_FILE_SIZE_KB`  | `150` KiB               | Files at or below this size are not resized further.                                                                                     |
| `-workers`             | `WORKERS`             | CPU count               | Concurrent conversion workers; must be at least 1.                                                                                       |
| `-clean`               | `CLEAN_MANIFEST`      | `false`                 | Start a new manifest during discovery; existing output files remain.                                                                     |
| `-delete-hidden-files` | `DELETE_HIDDEN_FILES` | `false`                 | In `auto` mode, delete discovered hidden regular files without prompting.                                                                |
| `-delete-originals`    | `DELETE_ORIGINALS`    | `false`                 | Allow unattended `auto` mode to delete originals.  |
| `-hidden-file-list`    | —                     | `false`                 | Print paths, sizes, and status for hidden files stored in the current volume's manifest. |

For images larger than the small-file threshold, the encoder aims for the ideal 150–200 KiB range and accepts up to 400 KiB before resizing dimensions. It never lowers quality below 55. When `QUALITY` is omitted, it chooses the best quality from 80 down to 55. When you set `QUALITY`, it evaluates that value plus and minus 5 quality points, choosing the best result that meets the size policy. If needed, it progressively reduces dimensions while retaining the permitted quality range.

Example `.env`:

```dotenv
MODE=all
VOLUME_PATH=/Volumes/ExternalSSD/Photos
ALLOWED_TYPES=png,jpg,jpeg
# Optional: omit for automatic 80–55 quality selection.
QUALITY=80
TARGET_SIZE_KB=200
SMALL_FILE_SIZE_KB=150
WORKERS=4
DELETE_HIDDEN_FILES=false
DELETE_ORIGINALS=false
```

## Low-space conversion

The default temporary workspace is `VOLUME_PATH/processed`. Before conversion, the app estimates the required output capacity from the pending image count and target size, including a 10% safety reserve.

When that workspace lacks space, `convert`, `all`, and `auto` ask for an existing empty directory with the required free capacity, then validate it before any conversion starts. You can avoid this prompt by setting `PROCESSED_PATH` or passing `-processed` to an appropriately sized external directory. An explicit `PROCESSED_PATH` is retained as the output location; it is not automatically moved back into the source tree.

## Manifest and logs

By default, state is stored outside the repository and source volume:

- macOS: `~/Library/Application Support/shrink-my-photos/`
- Windows: `%AppData%\shrink-my-photos\`
- Linux: `$XDG_CONFIG_HOME/shrink-my-photos/` (normally `~/.config/shrink-my-photos/`)

The app creates a separate hashed manifest and log for each absolute source-volume path, so separate photo roots do not share history.

## Camera-photo filtering

The scanner excludes a directory when it detects camera EXIF metadata in one of its allowed image files. It recognizes common DSLR/mirrorless makes (for example Canon, Nikon, Sony, Fujifilm, Leica, Panasonic, Olympus, Pentax, Ricoh, and Hasselblad) and iPhone/iPad camera metadata. Files with unreadable or no EXIF metadata are treated as ordinary images, deliberately avoiding false positives for screenshots and downloaded images.

EXIF is not a guarantee: stripped metadata, unsupported formats, or a nonstandard camera make cannot be reliably identified. Review the first run on a copy if camera-photo exclusion is important.

## Progress and logs

Scanning and conversion progress display both completed and total file counts, for example `Scanning (42/120)`. Operational errors and images that remain above the target size are written to the per-volume log file in the state directory. Conversion reports potential savings after replacement; deletion reports source space reclaimed, using KB, MB, GB, or TB as appropriate.

## Hidden Apple metadata files

Discovery records every dot-prefixed regular file it skips in the per-volume manifest, including its path, size, and `present` or `deleted` status. It prints the path list and total hidden-file size at the end of discovery. In `all` or `stage` mode, you can explicitly approve their deletion. In unattended `auto` mode, opt in with `DELETE_HIDDEN_FILES=true` or `-delete-hidden-files`.

To print the recorded list later, using `VOLUME_PATH` from `.env` or an explicit `-volume` flag:

```shell
./shrinker -hidden-file-list
./shrinker -volume=/Volumes/ExternalSSD/Photos -hidden-file-list
```

## Manifest format: JSON or SQLite?

JSON remains a good fit for the current single-process, per-volume manifest: it is portable, inspectable, and the state is written after each phase. SQLite would be worthwhile only if the manifest grows very large, partial/resumable updates during conversion need transaction-level durability, or multiple app processes must access the same volume concurrently. The current workflow does not need that complexity.

## Verification

```shell
go test ./...
go test -race ./...
go vet ./...
```

## License

[MIT License](LICENSE)
