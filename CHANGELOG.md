# Changelog

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Sustain pedal (CC64): notes keep ringing until the pedal is released.

## [0.1.0] - 2026-10-07

### Added
- Standard MIDI File (type 0/1) parsing with tempo map support.
- Software synth: simple timbres chosen by GM program family, drum channel.
- Terminal UI: progress bar, space to pause, ←/→ to seek 5 s, q to quit.
- `version` and `update` commands; one-line install script.
