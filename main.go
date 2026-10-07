package main

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
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

	scr := newScreen(song, filepath.Base(os.Args[1]))
	defer scr.close()
	const seekStep = 5.0
	esc := 0
	tick := time.NewTicker(50 * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case k := <-keys:
			// Arrow keys arrive as ESC [ A/B/C/D (up/down/right/left).
			if esc == 2 {
				esc = 0
				switch k {
				case 'C':
					p.Seek(p.Position() + seekStep)
				case 'D':
					p.Seek(p.Position() - seekStep)
				case 'A':
					p.SetVolume(+0.1)
				case 'B':
					p.SetVolume(-0.1)
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
			case '+', '=':
				p.SetSpeed(+0.1)
			case '-', '_':
				p.SetSpeed(-0.1)
			case 'q', 'Q', 3:
				return
			}
		case <-sig:
			return
		case <-tick.C:
			scr.draw(p.Snapshot())
			if p.IsDone() {
				return
			}
		}
	}
}
