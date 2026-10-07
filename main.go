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
	cleanupOldBinary()
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
	paths, err := collectFiles(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	var songs []*Song
	var names []string
	for _, path := range paths {
		song, err := LoadMIDI(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "skipping %s: %v\n", path, err)
			continue
		}
		songs = append(songs, song)
		names = append(names, filepath.Base(path))
	}
	if len(songs) == 0 {
		fmt.Fprintln(os.Stderr, "error: nothing to play")
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

	cur := 0
	p := NewPlayer(songs[cur])
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

	scr := newScreen()
	defer scr.close()
	play := func(i int) {
		cur = i
		p.Load(songs[cur])
		title := names[cur]
		if len(songs) > 1 {
			title = fmt.Sprintf("%s  [%d/%d]", title, cur+1, len(songs))
		}
		scr.setSong(songs[cur], title)
	}
	play(0)
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
			case 'n', 'N':
				if cur+1 < len(songs) {
					play(cur + 1)
				}
			case 'p', 'P':
				// Like most players: restart the song unless it just began.
				if p.Position() < 3 && cur > 0 {
					play(cur - 1)
				} else {
					p.Seek(0)
				}
			case 'q', 'Q', 3:
				return
			}
		case <-sig:
			return
		case <-tick.C:
			scr.draw(p.Snapshot())
			if p.IsDone() {
				if cur+1 == len(songs) {
					return
				}
				play(cur + 1)
			}
		}
	}
}

// collectFiles expands directories into the MIDI files they contain.
func collectFiles(args []string) ([]string, error) {
	var out []string
	for _, a := range args {
		info, err := os.Stat(a)
		if err != nil {
			return nil, err
		}
		if !info.IsDir() {
			out = append(out, a)
			continue
		}
		entries, err := os.ReadDir(a)
		if err != nil {
			return nil, err
		}
		for _, e := range entries {
			ext := strings.ToLower(filepath.Ext(e.Name()))
			if !e.IsDir() && (ext == ".mid" || ext == ".midi") {
				out = append(out, filepath.Join(a, e.Name()))
			}
		}
	}
	return out, nil
}
