package main

import (
	"math"
	"testing"
)

func TestSustainPedal(t *testing.T) {
	p := NewPlayer(&Song{})
	p.apply(Event{Status: 0xB0, Data1: 64, Data2: 127}) // pedal down
	p.apply(Event{Status: 0x90, Data1: 60, Data2: 100})
	p.apply(Event{Status: 0x80, Data1: 60})
	v := p.voices[0]
	if v.released || !v.held {
		t.Fatalf("note-off with pedal down: released=%v held=%v, want held", v.released, v.held)
	}

	p.apply(Event{Status: 0xB1, Data1: 64, Data2: 0}) // other channel's pedal
	if v.released {
		t.Fatal("pedal up on another channel released the note")
	}

	p.apply(Event{Status: 0xB0, Data1: 64, Data2: 0}) // pedal up
	if !v.released || v.held {
		t.Fatalf("after pedal up: released=%v held=%v, want released", v.released, v.held)
	}
}

func TestNoteOffWithoutPedal(t *testing.T) {
	p := NewPlayer(&Song{})
	p.apply(Event{Status: 0x90, Data1: 60, Data2: 100})
	p.apply(Event{Status: 0x90, Data1: 60, Data2: 0}) // note-on vel 0 = note-off
	if v := p.voices[0]; !v.released || v.held {
		t.Fatalf("released=%v held=%v, want released", v.released, v.held)
	}
}

func TestSeekRestoresSustain(t *testing.T) {
	p := NewPlayer(&Song{Duration: 2, Events: []Event{
		{Time: 0, Status: 0xB0, Data1: 64, Data2: 127},
		{Time: 1.5, Status: 0xB0, Data1: 64, Data2: 0},
	}})
	p.Seek(1)
	if !p.chans[0].sustain {
		t.Error("sustain not restored by seek")
	}
	p.Seek(1.8)
	if p.chans[0].sustain {
		t.Error("sustain still on after seeking past pedal up")
	}
}

func TestStealingFadesInsteadOfCutting(t *testing.T) {
	p := NewPlayer(&Song{})
	for n := range byte(maxVoices) {
		p.apply(Event{Status: 0x90, Data1: n, Data2: 100})
	}
	p.apply(Event{Status: 0x80, Data1: 10}) // released voices are stolen first
	first, released := p.voices[0], p.voices[10]
	p.apply(Event{Status: 0x90, Data1: 100, Data2: 100})

	if len(p.voices) != maxVoices+1 {
		t.Fatalf("got %d voices, want %d (stolen voice kept while fading)", len(p.voices), maxVoices+1)
	}
	if !released.stolen || first.stolen {
		t.Fatalf("stolen: released=%v first=%v, want the released voice stolen", released.stolen, first.stolen)
	}

	// The stolen voice must fade smoothly and be gone within ~50 ms.
	prev := released.gain
	for range sampleRate / 20 {
		released.sample()
		if released.gain > prev {
			t.Fatal("gain increased while fading")
		}
		prev = released.gain
	}
	if !released.dead() {
		t.Errorf("stolen voice still alive after 50 ms (gain %v)", released.gain)
	}
}

func TestEnvelopeIsSampleRateIndependent(t *testing.T) {
	// After releaseTau seconds a released note should be at 1/e of its level.
	v := &voice{freq: 440, amp: 1, gain: 1, env: 1, released: true}
	for range int(math.Round(sampleRate * releaseTau)) {
		v.render()
	}
	if got, want := v.env, math.Exp(-1); math.Abs(got-want) > 0.01 {
		t.Errorf("env after releaseTau = %v, want ≈ %v", got, want)
	}
}
