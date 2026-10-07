# termidi

A tiny MIDI player for your terminal. It ships with a built-in software synth, so no SoundFont or external synthesizer is needed.

## Install

```sh
curl -fsSL https://raw.githubusercontent.com/EmreErdogan/termidi/main/install.sh | sh
```

The binary is installed to `~/.local/bin` (override with `INSTALL_DIR=/path`). If that directory is not on your `PATH`, it is added to your shell config automatically.

Supported platforms: macOS and Linux (amd64, arm64).

## Usage

```sh
termidi song.mid
```

| Key | Action |
| --- | --- |
| `space` | pause / resume |
| `←` / `→` | seek 5 s back / forward |
| `q` | quit |

## Updating

```sh
termidi update     # update to the latest release
termidi version    # print the installed version
```

## Building from source

```sh
make build   # Go 1.27+; Linux also needs libasound2-dev
```

## Releasing

1. In `CHANGELOG.md`, move the `[Unreleased]` entries under a new version heading.
2. Commit, push, then run `make release V=x.y.z`. It pushes the tag; GitHub Actions builds the binaries and attaches them to the release.
