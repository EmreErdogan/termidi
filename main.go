package main

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"time"

	"github.com/ebitengine/oto/v3"
	"golang.org/x/term"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, errUsage)
		os.Exit(1)
	}
	switch os.Args[1] {
	case "version", "--version", "-v":
		fmt.Println(appName, version)
		return
	case "update":
		if err := runUpdate(); err != nil {
			fmt.Fprintln(os.Stderr, "update failed:", err)
			os.Exit(1)
		}
		return
	}
	song, err := LoadMIDI(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}

	ctx, ready, err := oto.NewContext(&oto.NewContextOptions{
		SampleRate:   sampleRate,
		ChannelCount: 2,
		Format:       oto.FormatFloat32LE,
		BufferSize:   50 * time.Millisecond,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "cannot open audio device:", err)
		os.Exit(1)
	}
	<-ready

	p := NewPlayer(song)
	out := ctx.NewPlayer(p)
	// Keep a live reference: oto closes players via a finalizer once they are
	// garbage collected, which silently stops playback.
	defer out.Close()
	out.Play()

	// Raw terminal so single keypresses work (space = pause, q = quit).
	keys := make(chan byte, 4)
	if old, err := term.MakeRaw(int(os.Stdin.Fd())); err == nil {
		defer term.Restore(int(os.Stdin.Fd()), old)
		go func() {
			b := make([]byte, 1)
			for {
				if n, err := os.Stdin.Read(b); err != nil || n == 0 {
					return
				}
				keys <- b[0]
			}
		}()
	}
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)

	name := filepath.Base(os.Args[1])
	fmt.Printf("♪ %s  (space: pause, ←/→: seek 5s, q: quit)\r\n", name)
	const seekStep = 5.0
	esc := 0
	tick := time.NewTicker(100 * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case k := <-keys:
			// Arrow keys arrive as ESC [ C (right) and ESC [ D (left).
			if esc == 2 {
				esc = 0
				switch k {
				case 'C':
					p.Seek(p.Position() + seekStep)
				case 'D':
					p.Seek(p.Position() - seekStep)
				}
				continue
			}
			if esc == 1 && k == '[' {
				esc = 2
				continue
			}
			esc = 0
			switch k {
			case 27:
				esc = 1
			case ' ':
				p.TogglePause()
			case 'q', 'Q', 3:
				fmt.Print("\r\n")
				return
			}
		case <-sig:
			fmt.Print("\r\n")
			return
		case <-tick.C:
			draw(p, song.Duration)
			if p.IsDone() {
				fmt.Print("\r\n")
				return
			}
		}
	}
}

func draw(p *Player, dur float64) {
	pos := p.Position()
	const w = 30
	frac := 0.0
	if dur > 0 {
		frac = min(pos/dur, 1)
	}
	n := int(frac * w)
	state := "▶"
	p.mu.Lock()
	if p.Paused {
		state = "⏸"
	}
	p.mu.Unlock()
	fmt.Printf("\r%s [%s%s] %s / %s ", state, strings.Repeat("█", n), strings.Repeat("░", w-n), mmss(pos), mmss(dur))
}

func mmss(s float64) string {
	return fmt.Sprintf("%02d:%02d", int(s)/60, int(s)%60)
}
