package main

import (
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"sort"
)

// Event is a channel event placed on an absolute timeline (seconds).
type Event struct {
	Time   float64
	Status byte // high nibble: type, low nibble: channel
	Data1  byte
	Data2  byte
}

type Song struct {
	Events   []Event
	Duration float64
}

type rawEvent struct {
	tick  uint64
	order int
	tempo uint32 // >0 for tempo meta events
	ev    Event
}

func LoadMIDI(path string) (*Song, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(b) < 14 || string(b[:4]) != "MThd" {
		return nil, errors.New("not a valid MIDI file")
	}
	hlen := binary.BigEndian.Uint32(b[4:8])
	ntracks := int(binary.BigEndian.Uint16(b[10:12]))
	division := binary.BigEndian.Uint16(b[12:14])
	if division&0x8000 != 0 {
		return nil, errors.New("SMPTE time division is not supported")
	}
	pos := 8 + int(hlen)

	var raws []rawEvent
	order := 0
	for t := 0; t < ntracks && pos+8 <= len(b); t++ {
		id := string(b[pos : pos+4])
		l := int(binary.BigEndian.Uint32(b[pos+4 : pos+8]))
		pos += 8
		end := pos + l
		if end > len(b) {
			end = len(b)
		}
		if id == "MTrk" {
			evs, err := parseTrack(b[pos:end], &order)
			if err != nil {
				return nil, fmt.Errorf("track %d: %w", t, err)
			}
			raws = append(raws, evs...)
		}
		pos = end
	}

	sort.SliceStable(raws, func(i, j int) bool {
		if raws[i].tick != raws[j].tick {
			return raws[i].tick < raws[j].tick
		}
		return raws[i].order < raws[j].order
	})

	// Convert ticks to seconds using the tempo map.
	song := &Song{}
	tempo := 500000.0 // µs per quarter note
	var lastTick uint64
	var sec float64
	for _, r := range raws {
		sec += float64(r.tick-lastTick) * tempo / 1e6 / float64(division)
		lastTick = r.tick
		if r.tempo > 0 {
			tempo = float64(r.tempo)
			continue
		}
		r.ev.Time = sec
		song.Events = append(song.Events, r.ev)
	}
	song.Duration = sec
	return song, nil
}

func parseTrack(d []byte, order *int) ([]rawEvent, error) {
	var out []rawEvent
	var tick uint64
	var running byte
	i := 0
	readVar := func() (uint64, error) {
		var v uint64
		for n := 0; n < 4; n++ {
			if i >= len(d) {
				return 0, errors.New("unexpected end of file")
			}
			c := d[i]
			i++
			v = v<<7 | uint64(c&0x7f)
			if c&0x80 == 0 {
				return v, nil
			}
		}
		return v, nil
	}
	for i < len(d) {
		delta, err := readVar()
		if err != nil {
			return nil, err
		}
		tick += delta
		if i >= len(d) {
			break
		}
		st := d[i]
		if st&0x80 != 0 {
			i++
		} else {
			if running == 0 {
				return nil, errors.New("invalid running status")
			}
			st = running
		}
		switch {
		case st == 0xFF:
			// The spec says meta and sysex events cancel running status, but
			// keeping it is harmless for valid files and lets sloppy ones play.
			if i >= len(d) {
				return out, nil
			}
			typ := d[i]
			i++
			l, err := readVar()
			if err != nil {
				return nil, err
			}
			if i+int(l) > len(d) {
				return out, nil
			}
			if typ == 0x51 && l == 3 {
				tp := uint32(d[i])<<16 | uint32(d[i+1])<<8 | uint32(d[i+2])
				*order++
				out = append(out, rawEvent{tick: tick, order: *order, tempo: tp})
			}
			i += int(l)
			if typ == 0x2F {
				return out, nil
			}
		case st == 0xF0 || st == 0xF7:
			l, err := readVar()
			if err != nil {
				return nil, err
			}
			i += int(l)
		case st > 0xF0:
			// System common/real-time messages don't belong in a file; skip
			// them along with their data bytes.
			switch st {
			case 0xF1, 0xF3:
				i++
			case 0xF2:
				i += 2
			}
		default:
			running = st
			n := 2
			if t := st & 0xF0; t == 0xC0 || t == 0xD0 {
				n = 1
			}
			if i+n > len(d) {
				return out, nil
			}
			ev := Event{Status: st, Data1: d[i]}
			if n == 2 {
				ev.Data2 = d[i+1]
			}
			i += n
			*order++
			out = append(out, rawEvent{tick: tick, order: *order, ev: ev})
		}
	}
	return out, nil
}
