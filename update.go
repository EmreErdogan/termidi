package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// Set at build time: -ldflags "-X main.version=v1.2.3".
var version = "dev"

const (
	appName = "termidi"
	repo    = "EmreErdogan/termidi" // GitHub owner/repo; release assets live here.
)

func assetName() string {
	return fmt.Sprintf("%s-%s-%s", appName, runtime.GOOS, runtime.GOARCH)
}

// runUpdate replaces the running binary with the latest GitHub release.
func runUpdate() error {
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Get("https://api.github.com/repos/" + repo + "/releases/latest")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("cannot fetch latest release: %s", resp.Status)
	}
	var rel struct {
		Tag    string `json:"tag_name"`
		Assets []struct {
			Name string `json:"name"`
			URL  string `json:"browser_download_url"`
		} `json:"assets"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return err
	}
	if version != "dev" && !newer(rel.Tag, version) {
		fmt.Printf("already up to date (%s)\n", version)
		return nil
	}
	url := ""
	for _, a := range rel.Assets {
		if a.Name == assetName() {
			url = a.URL
		}
	}
	if url == "" {
		return fmt.Errorf("release %s has no %s binary", rel.Tag, assetName())
	}

	exe, err := os.Executable()
	if err != nil {
		return err
	}
	if exe, err = filepath.EvalSymlinks(exe); err != nil {
		return err
	}
	bin, err := client.Get(url)
	if err != nil {
		return err
	}
	defer bin.Body.Close()
	if bin.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed: %s", bin.Status)
	}
	// Write next to the target and rename, so the swap is atomic.
	tmp, err := os.CreateTemp(filepath.Dir(exe), "."+appName+"-update-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := io.Copy(tmp, bin.Body); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmp.Name(), 0o755); err != nil {
		return err
	}
	if err := os.Rename(tmp.Name(), exe); err != nil {
		return err
	}
	fmt.Printf("updated %s → %s\n", version, rel.Tag)
	return nil
}

// newer reports whether semver tag a is greater than b ("v1.2.3" form).
func newer(a, b string) bool {
	pa, pb := semver(a), semver(b)
	if pa == nil || pb == nil {
		return a != b
	}
	for i := range 3 {
		if pa[i] != pb[i] {
			return pa[i] > pb[i]
		}
	}
	return false
}

func semver(s string) []int {
	parts := strings.SplitN(strings.TrimPrefix(s, "v"), ".", 3)
	if len(parts) != 3 {
		return nil
	}
	out := make([]int, 3)
	for i, p := range parts {
		n, err := strconv.Atoi(strings.SplitN(p, "-", 2)[0])
		if err != nil {
			return nil
		}
		out[i] = n
	}
	return out
}

var errUsage = errors.New("usage: " + appName + " <file.mid> | update | version")
