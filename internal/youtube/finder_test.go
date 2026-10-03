package youtube

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

// fakeRunner records every call it receives and answers them in order from
// outputs (the last entry is reused once exhausted), unless err is set, in
// which case every call fails with it.
type fakeRunner struct {
	outputs []string
	err     error
	calls   [][]string
}

func (f *fakeRunner) Run(_ context.Context, name string, args []string) (string, error) {
	f.calls = append(f.calls, append([]string{name}, args...))
	if f.err != nil {
		return "", f.err
	}
	if len(f.outputs) == 0 {
		return "", nil
	}
	index := len(f.calls) - 1
	if index >= len(f.outputs) {
		index = len(f.outputs) - 1
	}
	return f.outputs[index], nil
}

func TestFindPicksTheLongestClipAtOrUnderMaxDuration(t *testing.T) {
	runner := &fakeRunner{outputs: []string{"abc123\t45\nshort1\t5\ntoolong\t90\nnodur\tnotanumber\n"}}
	finder := &Finder{binary: "yt-dlp", runner: runner, sem: make(chan struct{}, 1)}

	url, err := finder.Find(context.Background(), "Bulgarian split squat")
	require.NoError(t, err)
	require.Equal(t, "https://www.youtube.com/watch?v=abc123", url)
	require.Len(t, runner.calls, 2, "Find must search both generally and for Shorts")
	require.Contains(t, runner.calls[0], "ytsearch8:Bulgarian split squat exercise technique")
	require.Contains(t, runner.calls[1], "ytsearch8:Bulgarian split squat exercise technique shorts")
}

func TestFindPrefersAShortsCandidateOnADurationTie(t *testing.T) {
	runner := &fakeRunner{outputs: []string{
		"general1\t50\n", // general search: 50s
		"shorts1\t50\n",  // Shorts search: also 50s — should win the tie
	}}
	finder := &Finder{binary: "yt-dlp", runner: runner, sem: make(chan struct{}, 1)}

	url, err := finder.Find(context.Background(), "Deadlift")
	require.NoError(t, err)
	require.Equal(t, "https://www.youtube.com/watch?v=shorts1", url)
}

func TestFindSurvivesAFailedShortsSearch(t *testing.T) {
	runner := &sequencedRunner{
		responses: []response{
			{output: "abc123\t45\n"},
			{err: errors.New("temporary failure")},
		},
	}
	finder := &Finder{binary: "yt-dlp", runner: runner, sem: make(chan struct{}, 1)}

	url, err := finder.Find(context.Background(), "Row")
	require.NoError(t, err, "a failed Shorts search must not fail the whole lookup")
	require.Equal(t, "https://www.youtube.com/watch?v=abc123", url)
}

// sequencedRunner answers calls one at a time from responses, each of which
// can independently be an output or an error — fakeRunner can't express a
// run where only some calls fail.
type sequencedRunner struct {
	responses []response
	calls     int
}

type response struct {
	output string
	err    error
}

func (r *sequencedRunner) Run(context.Context, string, []string) (string, error) {
	resp := r.responses[r.calls]
	r.calls++
	return resp.output, resp.err
}

func TestFindReturnsEmptyWhenNothingQualifies(t *testing.T) {
	runner := &fakeRunner{outputs: []string{"toolong1\t75\ntoolong2\t600\n"}}
	finder := &Finder{binary: "yt-dlp", runner: runner, sem: make(chan struct{}, 1)}

	url, err := finder.Find(context.Background(), "Deadlift")
	require.NoError(t, err)
	require.Empty(t, url)
}

func TestFindReturnsEmptyOnNoOutput(t *testing.T) {
	runner := &fakeRunner{}
	finder := &Finder{binary: "yt-dlp", runner: runner, sem: make(chan struct{}, 1)}

	url, err := finder.Find(context.Background(), "Plank")
	require.NoError(t, err)
	require.Empty(t, url)
}

func TestFindPropagatesGeneralSearchErrors(t *testing.T) {
	runner := &fakeRunner{err: errors.New("exec: \"yt-dlp\": executable file not found in $PATH")}
	finder := &Finder{binary: "yt-dlp", runner: runner, sem: make(chan struct{}, 1)}

	_, err := finder.Find(context.Background(), "Push-up")
	require.ErrorContains(t, err, "yt-dlp search failed")
	require.ErrorContains(t, err, "executable file not found")
	require.Len(t, runner.calls, 1, "a failed general search should not be retried as the Shorts search")
}

func TestFindRejectsEmptyExerciseName(t *testing.T) {
	finder := NewFinder(2)
	_, err := finder.Find(context.Background(), "   ")
	require.ErrorContains(t, err, "exercise name is required")
}

func TestBestCandidatePrefersLongerDuration(t *testing.T) {
	best := bestCandidate([]candidate{
		{id: "a", duration: 10 * 1e9},
		{id: "b", duration: 55 * 1e9},
		{id: "c", duration: 30 * 1e9},
	})
	require.NotNil(t, best)
	require.Equal(t, "b", best.id)
}

func TestBestCandidateNilWhenEmpty(t *testing.T) {
	require.Nil(t, bestCandidate(nil))
}
