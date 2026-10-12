package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/KitsuneForgering/FileOrganizer/internal/daemon"
	"github.com/KitsuneForgering/FileOrganizer/internal/organizer"
)

func main() {
	config := flag.String("config", defaultProfilePath(), "path to the JSON rules profile")
	apply := flag.Bool("apply", false, "move matching files; otherwise report previews only")
	settle := flag.Duration("settle", 750*time.Millisecond, "delay after an event before scanning")
	flag.Parse()
	file, err := os.Open(*config)
	if err != nil {
		fatal(err)
	}
	defer file.Close()
	var request organizer.Request
	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		fatal(err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		fatal(fmt.Errorf("expected exactly one JSON profile"))
	}
	watch, err := daemon.New(request, *apply, *settle)
	if err != nil {
		fatal(err)
	}
	defer watch.Close()
	mode := "preview"
	if *apply {
		mode = "apply"
	}
	fmt.Fprintf(os.Stderr, "watching %s (%s mode)\n", request.Source, mode)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	encoder := json.NewEncoder(os.Stdout)
	if err := watch.Run(ctx, func(event daemon.Event) { _ = encoder.Encode(event) }); err != nil {
		fatal(err)
	}
}

func defaultProfilePath() string {
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		configDir, err := os.UserConfigDir()
		if err != nil {
			return ""
		}
		base = configDir
	}
	return filepath.Join(base, "fileorganizer", "profile.json")
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
