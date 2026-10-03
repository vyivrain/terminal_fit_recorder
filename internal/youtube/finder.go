// Package youtube looks up a short technique video for an exercise name,
// using the locally installed yt-dlp binary to search YouTube. It never
// invents a URL: a search that finds nothing suitable returns an empty
// string, not an error.
package youtube

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// MaxDuration is the longest clip Find will accept — a quick technique demo
// rather than a full-length tutorial.
const MaxDuration = 60 * time.Second

// searchResultCount bounds how many candidates yt-dlp inspects per search, so
// one lookup can't run away fetching metadata for dozens of videos.
const searchResultCount = 8

type commandRunner interface {
	Run(ctx context.Context, name string, args []string) (string, error)
}

type execRunner struct{}

func (execRunner) Run(ctx context.Context, name string, args []string) (string, error) {
	output, err := exec.CommandContext(ctx, name, args...).Output()
	return string(output), err
}

// Finder searches YouTube for exercise technique videos. Build one with
// NewFinder; the zero value is not usable.
type Finder struct {
	binary string
	runner commandRunner
	sem    chan struct{}
}

// NewFinder returns a Finder backed by the yt-dlp binary, running at most
// maxConcurrent searches at a time so a workout with many exercises doesn't
// launch a pile of simultaneous subprocesses.
func NewFinder(maxConcurrent int) *Finder {
	if maxConcurrent < 1 {
		maxConcurrent = 1
	}
	return &Finder{
		binary: "yt-dlp",
		runner: execRunner{},
		sem:    make(chan struct{}, maxConcurrent),
	}
}

type candidate struct {
	id       string
	duration time.Duration
	// short marks a candidate found via the Shorts-biased search, so ties on
	// duration can favor it — Shorts are the format this lookup specifically
	// wants when available.
	short bool
}

// Find searches YouTube for a short technique video of exerciseName and
// returns its watch URL, or "" if nothing at or under MaxDuration turned up.
// A non-nil error means the search itself failed (e.g. yt-dlp is not
// installed), not that no video was found.
//
// It runs two searches: a general one, and one biased toward YouTube Shorts
// (yt-dlp has no dedicated "search only Shorts" operator, so this nudges
// YouTube's own search ranking toward that format instead). The Shorts
// search is additive — if it fails, the general search's result still
// stands; only a failure of the general search is treated as fatal.
func (f *Finder) Find(ctx context.Context, exerciseName string) (string, error) {
	exerciseName = strings.TrimSpace(exerciseName)
	if exerciseName == "" {
		return "", fmt.Errorf("exercise name is required")
	}

	select {
	case f.sem <- struct{}{}:
	case <-ctx.Done():
		return "", ctx.Err()
	}
	defer func() { <-f.sem }()

	candidates, err := f.search(ctx, exerciseName, "exercise technique", false)
	if err != nil {
		return "", err
	}
	if shortsCandidates, shortsErr := f.search(ctx, exerciseName, "exercise technique shorts", true); shortsErr == nil {
		candidates = append(candidates, shortsCandidates...)
	}

	best := bestCandidate(candidates)
	if best == nil {
		return "", nil
	}
	return "https://www.youtube.com/watch?v=" + best.id, nil
}

// search runs one yt-dlp ytsearch query, built from exerciseName plus a
// query suffix, and returns the candidates it found.
func (f *Finder) search(ctx context.Context, exerciseName, querySuffix string, short bool) ([]candidate, error) {
	query := fmt.Sprintf("ytsearch%d:%s %s", searchResultCount, exerciseName, querySuffix)
	output, err := f.runner.Run(ctx, f.binary, []string{
		"--no-warnings", "--skip-download", "--ignore-errors",
		"--print", "%(id)s\t%(duration)s",
		query,
	})
	if err != nil {
		return nil, fmt.Errorf("yt-dlp search failed: %w", err)
	}
	return parseCandidates(output, short), nil
}

// parseCandidates reads yt-dlp's "<id>\t<duration>" lines, keeping only
// entries with a valid id and a duration at or under MaxDuration.
func parseCandidates(output string, short bool) []candidate {
	var candidates []candidate
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		fields := strings.SplitN(scanner.Text(), "\t", 2)
		if len(fields) != 2 {
			continue
		}
		id := strings.TrimSpace(fields[0])
		seconds, err := strconv.ParseFloat(strings.TrimSpace(fields[1]), 64)
		if id == "" || err != nil {
			continue
		}
		duration := time.Duration(seconds * float64(time.Second))
		if duration <= 0 || duration > MaxDuration {
			continue
		}
		candidates = append(candidates, candidate{id: id, duration: duration, short: short})
	}
	return candidates
}

// bestCandidate favors the longest clip within MaxDuration — a fuller demo
// over a near-instant one — among the entries that qualify at all, breaking
// a duration tie in favor of a Shorts-sourced candidate.
func bestCandidate(candidates []candidate) *candidate {
	var best *candidate
	for index := range candidates {
		current := &candidates[index]
		switch {
		case best == nil:
			best = current
		case current.duration > best.duration:
			best = current
		case current.duration == best.duration && current.short && !best.short:
			best = current
		}
	}
	return best
}
