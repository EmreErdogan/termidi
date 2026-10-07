package main

import (
	"fmt"
	"os"
	"slices"
	"strings"

	"golang.org/x/term"
)

var familyNames = [16]string{
	"Piano", "Chromatic", "Organ", "Guitar", "Bass", "Strings", "Ensemble", "Brass",
	"Reed", "Pipe", "Synth Lead", "Synth Pad", "Synth FX", "Ethnic", "Percussive", "Sound FX",
}

// screen redraws a multi-line view in place using ANSI cursor movement.
type screen struct {
	song     *Song
	name     string
	channels []int // channels that play notes, in order
	lo, hi   int   // note range the song uses
	meter    [16]float64
	lines    int // lines drawn by the previous frame
	color    bool
	fancy    bool // stdout is a terminal; otherwise print a single status line
}

func newScreen(song *Song, name string) *screen {
	s := &screen{song: song, name: name, lo: 127, hi: 0}
	s.fancy = term.IsTerminal(int(os.Stdout.Fd()))
	s.color = s.fancy && os.Getenv("NO_COLOR") == ""
	var used [16]bool
	for _, e := range song.Events {
		if e.Status&0xF0 == 0x90 && e.Data2 > 0 {
			ch := int(e.Status & 0x0F)
			if !used[ch] {
				used[ch] = true
				s.channels = append(s.channels, ch)
			}
			if ch != 9 {
				s.lo, s.hi = min(s.lo, int(e.Data1)), max(s.hi, int(e.Data1))
			}
		}
	}
	slices.Sort(s.channels)
	if s.lo > s.hi {
		s.lo, s.hi = 60, 72
	}
	if s.fancy {
		fmt.Print("\x1b[?25l") // hide cursor
	}
	return s
}

func (s *screen) close() {
	if s.fancy {
		fmt.Print("\x1b[?25h")
	}
	fmt.Print("\r\n")
}

func (s *screen) paint(text string, ch int) string {
	if !s.color {
		return text
	}
	codes := [...]int{36, 33, 35, 32, 34, 31, 96, 93, 95, 92, 94, 91}
	return fmt.Sprintf("\x1b[%dm%s\x1b[0m", codes[ch%len(codes)], text)
}

func (s *screen) dim(text string) string {
	if !s.color {
		return text
	}
	return "\x1b[2m" + text + "\x1b[0m"
}

func (s *screen) draw(snap Snapshot) {
	width := 80
	if w, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil && w > 0 {
		width = w
	}
	status := progress(snap, s.song.Duration, width)
	if !s.fancy {
		fmt.Printf("\r%s", status)
		return
	}

	var out []string
	out = append(out, fit(fmt.Sprintf("♪ %s  (space: pause, ←/→: seek 5s, q: quit)", s.name), width))
	meterW := max(0, min(40, width-18))
	for _, ch := range s.channels {
		// Meters jump up instantly and fall back smoothly.
		s.meter[ch] = max(snap.Level[ch], s.meter[ch]*0.8)
		name := familyNames[snap.Program[ch]/8]
		if ch == 9 {
			name = "Drums"
		}
		n := int(s.meter[ch]*float64(meterW) + 0.5)
		bar := s.paint(strings.Repeat("█", n), ch) + s.dim(strings.Repeat("░", meterW-n))
		out = append(out, fmt.Sprintf("%2d %-11s %s", ch+1, name, bar))
	}
	out = append(out, s.keyboard(snap, width))
	out = append(out, status)

	var b strings.Builder
	if s.lines > 1 {
		fmt.Fprintf(&b, "\x1b[%dA", s.lines-1)
	}
	for i, line := range out {
		if i > 0 {
			b.WriteString("\r\n")
		}
		b.WriteString("\r\x1b[2K")
		b.WriteString(line)
	}
	fmt.Print(b.String())
	s.lines = len(out)
}

// keyboard draws one cell per note over the song's range: sounding notes in
// their channel's color, C notes as octave markers.
func (s *screen) keyboard(snap Snapshot, width int) string {
	lo, hi := s.lo, s.hi
	// Pad to whole octaves when there is room, otherwise trim around the middle.
	lo, hi = lo-lo%12, hi+11-hi%12
	if n := width - 1; hi-lo+1 > n {
		mid := (s.lo + s.hi) / 2
		lo = max(0, mid-n/2)
		hi = min(127, lo+n-1)
	}
	var b strings.Builder
	for n := lo; n <= hi; n++ {
		switch {
		case snap.Note[n] >= 0:
			b.WriteString(s.paint("█", int(snap.Note[n])))
		case n%12 == 0:
			b.WriteString(s.dim("|"))
		case isBlack(n):
			b.WriteString(" ")
		default:
			b.WriteString(s.dim("·"))
		}
	}
	return b.String()
}

func isBlack(n int) bool {
	switch n % 12 {
	case 1, 3, 6, 8, 10:
		return true
	}
	return false
}

func progress(snap Snapshot, dur float64, width int) string {
	state := "▶"
	if snap.Paused {
		state = "⏸"
	}
	w := max(10, min(50, width-20))
	frac := 0.0
	if dur > 0 {
		frac = min(snap.Pos/dur, 1)
	}
	n := int(frac * float64(w))
	return fmt.Sprintf("%s [%s%s] %s / %s", state, strings.Repeat("█", n), strings.Repeat("░", w-n), mmss(snap.Pos), mmss(dur))
}

// fit truncates plain text to width runes.
func fit(s string, width int) string {
	r := []rune(s)
	if len(r) > width {
		return string(r[:max(0, width-1)]) + "…"
	}
	return s
}

func mmss(s float64) string {
	return fmt.Sprintf("%02d:%02d", int(s)/60, int(s)%60)
}
