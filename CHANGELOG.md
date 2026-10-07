# Changelog

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Live view: a level meter per channel (with instrument family) and a keyboard that lights up sounding notes in channel colors. Set `NO_COLOR` to disable colors.
- Master volume (↑/↓) and playback speed (+/-), shown in the status line.
- Playlists: pass several files or folders; `n`/`p` jump to the next/previous song. Files that fail to load are skipped with a warning.
- SMPTE time division (files timed in frames instead of beats).
- Windows support (amd64, arm64) with a PowerShell installer; `update` works on Windows too.

## [0.3.0] - 2026-10-07

### Added
- Stereo output with pan (CC10).
- Expression (CC11).
- Pitch bend range is read from RPN 0 instead of being fixed at ±2 semitones.

### Fixed
- Volume (CC7) changes now apply to notes that are already sounding.
- System messages (0xF1–0xFE) inside a track no longer corrupt the following events.

## [0.2.0] - 2026-10-07

### Added
- Sustain pedal (CC64): notes keep ringing until the pedal is released.

### Fixed
- No more clicks when more than 64 notes sound at once: the oldest voice fades out instead of being cut.
- Envelope timing is defined in seconds, so it no longer depends on the sample rate.

## [0.1.0] - 2026-10-07

### Added
- Standard MIDI File (type 0/1) parsing with tempo map support.
- Software synth: simple timbres chosen by GM program family, drum channel.
- Terminal UI: progress bar, space to pause, ←/→ to seek 5 s, q to quit.
- `version` and `update` commands; one-line install script.
