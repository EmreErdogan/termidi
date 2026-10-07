package main

import (
	"encoding/binary"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// smf builds a Standard MIDI File from raw track bodies.
func smf(division uint16, tracks ...[]byte) []byte {
	b := []byte("MThd")
	b = binary.BigEndian.AppendUint32(b, 6)
	b = binary.BigEndian.AppendUint16(b, 1)
	b = binary.BigEndian.AppendUint16(b, uint16(len(tracks)))
	b = binary.BigEndian.AppendUint16(b, division)
	for _, t := range tracks {
		b = append(b, "MTrk"...)
		b = binary.BigEndian.AppendUint32(b, uint32(len(t)))
		b = append(b, t...)
	}
	return b
}

func load(t *testing.T, data []byte) *Song {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.mid")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	s, err := LoadMIDI(path)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func approx(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

func TestExampleFile(t *testing.T) {
	s, err := LoadMIDI("example.mid")
	if err != nil {
		t.Fatal(err)
	}
	// Eight eighth notes (240 ticks at 480 PPQ, 120 BPM).
	if len(s.Events) != 16 {
		t.Fatalf("got %d events, want 16", len(s.Events))
	}
	if !approx(s.Duration, 2.0) {
		t.Errorf("duration = %v, want 2.0", s.Duration)
	}
	if e := s.Events[1]; e.Status != 0x80 || e.Data1 != 0x3C || !approx(e.Time, 0.25) {
		t.Errorf("second event = %+v, want note-off C4 at 0.25s", e)
	}
}

func TestMultiByteDelta(t *testing.T) {
	// Delta 0x81 0x80 0x00 = 16384 ticks; at 96 PPQ and 120 BPM that is 85.333 s.
	s := load(t, smf(96, []byte{
		0x00, 0x90, 60, 100,
		0x81, 0x80, 0x00, 0x80, 60, 0,
		0x00, 0xFF, 0x2F, 0x00,
	}))
	want := 16384.0 / 96 * 0.5
	if got := s.Events[1].Time; !approx(got, want) {
		t.Errorf("time = %v, want %v", got, want)
	}
}

func TestRunningStatus(t *testing.T) {
	s := load(t, smf(96, []byte{
		0x00, 0x90, 60, 100,
		0x00, 64, 100, // running status: note-on E4
		0x60, 60, 0, // running status: note-on vel 0 (note-off)
		0x00, 0xC0, 5, // program change: one data byte
		0x00, 7, // running status with one data byte
		0x00, 0xFF, 0x2F, 0x00,
	}))
	want := []Event{
		{0, 0x90, 60, 100},
		{0, 0x90, 64, 100},
		{0.5, 0x90, 60, 0},
		{0.5, 0xC0, 5, 0},
		{0.5, 0xC0, 7, 0},
	}
	if len(s.Events) != len(want) {
		t.Fatalf("got %d events, want %d: %+v", len(s.Events), len(want), s.Events)
	}
	for i, w := range want {
		if g := s.Events[i]; g.Status != w.Status || g.Data1 != w.Data1 || g.Data2 != w.Data2 || !approx(g.Time, w.Time) {
			t.Errorf("event %d = %+v, want %+v", i, g, w)
		}
	}
}

func TestTempoMapAcrossTracks(t *testing.T) {
	// Tempo track: 120 BPM, then 60 BPM at tick 96.
	tempo := []byte{
		0x00, 0xFF, 0x51, 0x03, 0x07, 0xA1, 0x20,
		0x60, 0xFF, 0x51, 0x03, 0x0F, 0x42, 0x40,
		0x00, 0xFF, 0x2F, 0x00,
	}
	// Note track: events at tick 96 and 192.
	notes := []byte{
		0x60, 0x90, 60, 100,
		0x60, 0x80, 60, 0,
		0x00, 0xFF, 0x2F, 0x00,
	}
	s := load(t, smf(96, tempo, notes))
	if got := s.Events[0].Time; !approx(got, 0.5) {
		t.Errorf("first note at %v, want 0.5", got)
	}
	if got := s.Events[1].Time; !approx(got, 1.5) {
		t.Errorf("second note at %v, want 1.5", got)
	}
	if !approx(s.Duration, 1.5) {
		t.Errorf("duration = %v, want 1.5", s.Duration)
	}
}

func TestSkipsSysExAndMeta(t *testing.T) {
	s := load(t, smf(96, []byte{
		0x00, 0xF0, 0x03, 0x7E, 0x09, 0xF7, // GM reset sysex
		0x00, 0xFF, 0x03, 0x02, 'h', 'i', // track name
		0x00, 0x90, 60, 100,
		0x00, 0xFF, 0x2F, 0x00,
	}))
	if len(s.Events) != 1 || s.Events[0].Status != 0x90 {
		t.Errorf("events = %+v, want a single note-on", s.Events)
	}
}

func TestInvalidFiles(t *testing.T) {
	cases := map[string]struct {
		data []byte
		err  string
	}{
		"not midi":      {[]byte("RIFF0000000000"), "not a valid MIDI file"},
		"smpte":         {smf(0xE728), "SMPTE"},
		"no status":     {smf(96, []byte{0x00, 60, 100}), "running status"},
		"truncated vlq": {smf(96, []byte{0x81}), "unexpected end of file"},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "bad.mid")
			os.WriteFile(path, c.data, 0o644)
			_, err := LoadMIDI(path)
			if err == nil || !strings.Contains(err.Error(), c.err) {
				t.Errorf("err = %v, want containing %q", err, c.err)
			}
		})
	}
}
