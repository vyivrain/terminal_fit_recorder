package commands

import (
	"context"
	"fmt"
	"time"

	"terminal_fit_recorder/internal/api"
	"terminal_fit_recorder/internal/db"
	"terminal_fit_recorder/internal/youtube"
)

const videosFetchTimeout = 10 * time.Minute

// VideosCommand backfills technique videos for exercises created outside the
// TUI (e.g. via `exercise save` or `exercise plan --save`), where there is no
// long-lived process for a background lookup to run in. It is the explicit,
// user-invoked counterpart to the TUI's automatic background lookup.
type VideosCommand struct {
	args []string
}

func NewVideosCommand(args ...string) *VideosCommand {
	return &VideosCommand{args: args}
}

func (cmd *VideosCommand) Name() string { return "fetch exercise videos" }

func (cmd *VideosCommand) Validate() error {
	if len(cmd.args) != 0 {
		return fmt.Errorf("usage: terminal_fit_recorder exercise videos")
	}
	return nil
}

func (cmd *VideosCommand) HelpManual() string {
	return "terminal_fit_recorder exercise videos\n    Look up a short (<=60s) technique video for every exercise in the active profile that doesn't have one yet. Requires yt-dlp."
}

func (cmd *VideosCommand) Execute(database *db.DB, _ api.OllamaClient) error {
	refs, err := database.GetExercisesMissingYoutubeURL()
	if err != nil {
		return fmt.Errorf("could not list exercises: %w", err)
	}
	if len(refs) == 0 {
		fmt.Println("Every exercise already has a technique video.")
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), videosFetchTimeout)
	defer cancel()

	finder := youtube.NewFinder(3)
	found := 0
	for _, ref := range refs {
		url, err := finder.Find(ctx, ref.Name)
		if err != nil {
			return fmt.Errorf("looking up %q: %w", ref.Name, err)
		}
		if url == "" {
			fmt.Printf("· %s: no video under a minute found\n", ref.Name)
			continue
		}
		if err := database.SetExerciseYoutubeURL(ref.ID, url); err != nil {
			return fmt.Errorf("saving video for %q: %w", ref.Name, err)
		}
		found++
		fmt.Printf("✓ %s: %s\n", ref.Name, url)
	}

	fmt.Printf("\n%d of %d exercises now have a technique video.\n", found, len(refs))
	return nil
}
