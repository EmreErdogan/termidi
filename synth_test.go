package main

import "testing"

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
