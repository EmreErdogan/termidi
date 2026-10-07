package main

import (
	"encoding/binary"
	"math"
	"math/rand"
	"sync"
)

const sampleRate = 44100

type voice struct {
	ch, note byte
	freq     float64
	phase    float64
	amp      float64
	env      float64
	released bool
	drum     bool
	wave     int
	age      float64
}

type channel struct {
	program byte
	volume  float64
	pitch   float64 // semitones
}

// Player is an io.Reader that renders the song as float32 stereo PCM.
type Player struct {
	mu     sync.Mutex
	song   *Song
	next   int
	t      float64
	voices []*voice
	chans  [16]channel
	Paused bool
	Done   bool
}

func NewPlayer(s *Song) *Player {
	p := &Player{song: s}
	for i := range p.chans {
		p.chans[i].volume = 100.0 / 127
	}
	return p
}

func (p *Player) Position() float64 {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.t
}

func (p *Player) TogglePause() {
	p.mu.Lock()
	p.Paused = !p.Paused
	p.mu.Unlock()
}

func (p *Player) IsDone() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.Done
}

func (p *Player) apply(e Event) {
	ch := e.Status & 0x0F
	c := &p.chans[ch]
	switch e.Status & 0xF0 {
	case 0x90:
		if e.Data2 > 0 {
			p.noteOn(ch, e.Data1, e.Data2)
			return
		}
		fallthrough
	case 0x80:
		for _, v := range p.voices {
			if v.ch == ch && v.note == e.Data1 && !v.drum {
				v.released = true
			}
		}
	case 0xB0:
		switch e.Data1 {
		case 7:
			c.volume = float64(e.Data2) / 127
		case 120, 123:
			for _, v := range p.voices {
				if v.ch == ch {
					v.released = true
				}
			}
		}
	case 0xC0:
		c.program = e.Data1
	case 0xE0:
		bend := int(e.Data2)<<7 | int(e.Data1) - 8192
		c.pitch = float64(bend) / 8192 * 2
		for _, v := range p.voices {
			if v.ch == ch && !v.drum {
				v.freq = noteFreq(v.note, c.pitch)
			}
		}
	}
}

func noteFreq(n byte, bend float64) float64 {
	return 440 * math.Pow(2, (float64(n)-69+bend)/12)
}

func (p *Player) noteOn(ch, note, vel byte) {
	c := &p.chans[ch]
	v := &voice{ch: ch, note: note, amp: float64(vel) / 127 * c.volume, drum: ch == 9}
	if !v.drum {
		v.freq = noteFreq(note, c.pitch)
		// Pick a rough timbre from the GM program family.
		switch fam := c.program / 8; {
		case fam == 0 || fam == 1: // piano, chromatic perc
			v.wave = 0
		case fam == 4 || fam == 5: // guitar, bass
			v.wave = 1
		case fam >= 10 && fam <= 11: // brass, reed
			v.wave = 2
		default:
			v.wave = 1
		}
	}
	if len(p.voices) > 64 {
		p.voices = p.voices[1:]
	}
	p.voices = append(p.voices, v)
}

func (v *voice) sample() float64 {
	dt := 1.0 / sampleRate
	v.age += dt
	if v.drum {
		// Short noise burst; low notes (kicks/toms) get a pitched thump.
		decay := 0.15
		if v.note < 40 {
			v.env = math.Exp(-v.age / 0.12)
			v.phase += (60 + 100*v.env) * dt
			return math.Sin(2*math.Pi*v.phase) * v.env * v.amp * 1.2
		}
		if v.note == 42 || v.note == 44 || v.note == 46 {
			decay = 0.05
		}
		v.env = math.Exp(-v.age / decay)
		return (rand.Float64()*2 - 1) * v.env * v.amp * 0.4
	}
	if v.released {
		v.env *= 0.9995
	} else if v.age < 0.01 {
		v.env = v.age / 0.01
	} else {
		v.env = math.Max(0.6, v.env*0.99997)
	}
	v.phase += v.freq * dt
	v.phase -= math.Floor(v.phase)
	var s float64
	switch v.wave {
	case 0: // sine + harmonic
		s = math.Sin(2*math.Pi*v.phase) + 0.3*math.Sin(4*math.Pi*v.phase)
	case 1: // triangle
		s = 4*math.Abs(v.phase-0.5) - 1
	case 2: // soft square
		if v.phase < 0.5 {
			s = 0.5
		} else {
			s = -0.5
		}
	}
	return s * v.env * v.amp
}

func (v *voice) dead() bool {
	if v.drum {
		return v.age > 0.05 && v.env < 0.001
	}
	return v.released && v.env < 0.001
}

func (p *Player) Read(buf []byte) (int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	frames := len(buf) / 8
	for f := 0; f < frames; f++ {
		var s float64
		if !p.Paused && !p.Done {
			for p.next < len(p.song.Events) && p.song.Events[p.next].Time <= p.t {
				p.apply(p.song.Events[p.next])
				p.next++
			}
			alive := p.voices[:0]
			for _, v := range p.voices {
				s += v.sample()
				if !v.dead() {
					alive = append(alive, v)
				}
			}
			p.voices = alive
			p.t += 1.0 / sampleRate
			if p.next >= len(p.song.Events) && len(p.voices) == 0 {
				p.Done = true
			}
		}
		s = math.Tanh(s * 0.35) // soft clip for polyphony
		bits := math.Float32bits(float32(s))
		binary.LittleEndian.PutUint32(buf[f*8:], bits)
		binary.LittleEndian.PutUint32(buf[f*8+4:], bits)
	}
	return frames * 8, nil
}

// Seek jumps to t seconds. Notes are cut; program/controller/pitch state is
// rebuilt by replaying non-note events up to the target.
func (p *Player) Seek(t float64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	t = math.Max(0, math.Min(t, p.song.Duration))
	p.voices = nil
	for i := range p.chans {
		p.chans[i] = channel{volume: 100.0 / 127}
	}
	p.next = 0
	for p.next < len(p.song.Events) && p.song.Events[p.next].Time < t {
		if e := p.song.Events[p.next]; e.Status&0xF0 != 0x90 && e.Status&0xF0 != 0x80 {
			p.apply(e)
		}
		p.next++
	}
	p.t = t
	p.Done = false
}
