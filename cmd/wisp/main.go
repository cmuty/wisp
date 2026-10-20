package main

import (
	"flag"
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"wisp/internal/app"
	"wisp/internal/library"
	"wisp/internal/player"
	"wisp/internal/terminal"
)

var version = "0.4.8-dev"

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "wisp:", err)
		os.Exit(1)
	}
}
func run() error {
	showVersion := flag.Bool("version", false, "Print version")
	preview := flag.String("preview", "", "Print a static UI frame (welcome, username, theme, sources, ready, disconnected, credential, completion, settings, playlists, lyrics, butterfly-flight)")
	previewFrame := flag.Int("preview-frame", -1, "Animation frame for a static preview")
	previewLanguage := flag.String("preview-language", "en", "Language for a static preview: en or ru")
	previewTheme := flag.String("preview-theme", "mist", "Theme for a static preview: mist, ember, haze, aurora or afterglow")
	plain := flag.Bool("ascii", false, "Use ASCII art and borders")
	flag.Parse()
	if *showVersion {
		fmt.Println("wisp", version)
		return nil
	}
	path := library.DefaultPath()
	s, err := library.Load(path)
	if err != nil {
		return err
	}
	engine := player.New()
	defer engine.Close()
	m := app.New(s, path, engine)
	m.Plain = m.Plain || *plain
	if *preview != "" {
		fmt.Print(app.PreviewLanguage(*preview, *previewTheme, m.Plain, *previewFrame, *previewLanguage))
		return nil
	}
	restoreTitle := terminal.SetTitle("wisp")
	defer restoreTitle()
	_, err = tea.NewProgram(m).Run()
	return err
}
