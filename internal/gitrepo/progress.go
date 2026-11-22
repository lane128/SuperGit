package gitrepo

import (
	"bytes"
	"context"
	"math"
	"regexp"
	"strconv"
	"strings"
)

// OperationProgress reports parsed progress from a long-running Git command
// such as fetch, pull, push, or clone.
type OperationProgress struct {
	Active        bool
	Phase         string
	Percent       int     // 0..100 within the current phase when Indeterminate is false
	Fraction      float32 // 0..1 overall monotonic progress for a single progress bar
	Indeterminate bool
}

// ProgressFunc receives live progress updates from Git's stderr stream.
type ProgressFunc func(OperationProgress)

type progressKey struct{}

// WithProgress attaches a progress callback to ctx. Network Git operations
// enable `--progress` when a callback is present.
func WithProgress(ctx context.Context, fn ProgressFunc) context.Context {
	if fn == nil {
		return ctx
	}
	return context.WithValue(ctx, progressKey{}, fn)
}

// WithScaledProgress maps progress fractions reported on ctx into [start, end],
// keeping a multi-step operation (such as fetching several remotes) monotonic.
func WithScaledProgress(ctx context.Context, start, end float32) context.Context {
	parent := progressFromContext(ctx)
	if parent == nil || end <= start {
		return ctx
	}
	start = clamp01(start)
	end = clamp01(end)
	return WithProgress(ctx, func(p OperationProgress) {
		if !p.Indeterminate {
			p.Fraction = clamp01(start + (end-start)*clamp01(p.Fraction))
		}
		parent(p)
	})
}

func progressFromContext(ctx context.Context) ProgressFunc {
	if ctx == nil {
		return nil
	}
	fn, _ := ctx.Value(progressKey{}).(ProgressFunc)
	return fn
}

func hasProgress(ctx context.Context) bool {
	return progressFromContext(ctx) != nil
}

var (
	progressPercentRE = regexp.MustCompile(`^(?:remote:\s+)?([A-Za-z][A-Za-z0-9 /-]{1,40}?):\s+(\d{1,3})%\s*\(`)
	progressCountRE   = regexp.MustCompile(`^(?:remote:\s+)?(Enumerating objects|Counting objects|Finding sources):\s+\d+`)
)

// progressStderrWriter streams Git stderr through a progress parser while
// collapsing `\r`-overwritten progress frames before forwarding to sink so a
// long transfer never trips the bounded stderr error buffer.
type progressStderrWriter struct {
	sink    *limitedBuffer
	fn      ProgressFunc
	command string

	lineBuf []byte
	maxFrac float32
	hasEmit bool
	last    OperationProgress
}

func newProgressStderrWriter(sink *limitedBuffer, command string, fn ProgressFunc) *progressStderrWriter {
	return &progressStderrWriter{sink: sink, command: command, fn: fn}
}

func (w *progressStderrWriter) Write(p []byte) (int, error) {
	for _, b := range p {
		switch b {
		case '\r':
			w.handleSegment(w.lineBuf)
			// Overwrite the pending line just like a terminal so repeated `\r`
			// ticks do not bloat the error buffer.
			w.lineBuf = w.lineBuf[:0]
		case '\n':
			w.handleSegment(w.lineBuf)
			if len(w.lineBuf) > 0 {
				_, _ = w.sink.Write(w.lineBuf)
			}
			_, _ = w.sink.Write([]byte{'\n'})
			w.lineBuf = w.lineBuf[:0]
		default:
			if len(w.lineBuf) < 4096 {
				w.lineBuf = append(w.lineBuf, b)
			}
		}
	}
	return len(p), nil
}

func (w *progressStderrWriter) Flush() {
	if len(w.lineBuf) == 0 {
		return
	}
	w.handleSegment(w.lineBuf)
	_, _ = w.sink.Write(w.lineBuf)
	w.lineBuf = w.lineBuf[:0]
}

func (w *progressStderrWriter) handleSegment(raw []byte) {
	line := strings.TrimSpace(string(bytes.TrimRight(raw, "\x00")))
	if line == "" {
		return
	}
	update, ok := parseGitProgressLine(w.command, line)
	if !ok {
		return
	}
	if !update.Indeterminate {
		if update.Fraction < w.maxFrac {
			update.Fraction = w.maxFrac
		} else {
			w.maxFrac = update.Fraction
		}
	}
	if w.hasEmit && sameProgress(w.last, update) {
		return
	}
	w.hasEmit = true
	w.last = update
	w.fn(update)
}

func sameProgress(a, b OperationProgress) bool {
	if a.Active != b.Active || a.Indeterminate != b.Indeterminate || a.Phase != b.Phase || a.Percent != b.Percent {
		return false
	}
	return math.Round(float64(a.Fraction*200)) == math.Round(float64(b.Fraction*200))
}

func parseGitProgressLine(command, line string) (OperationProgress, bool) {
	remotePrefix := strings.HasPrefix(line, "remote:")
	if m := progressPercentRE.FindStringSubmatch(line); len(m) == 3 {
		pct, err := strconv.Atoi(m[2])
		if err != nil {
			return OperationProgress{}, false
		}
		if pct < 0 {
			pct = 0
		} else if pct > 100 {
			pct = 100
		}
		phase := strings.TrimSpace(m[1])
		frac := phaseFraction(command, phase, remotePrefix, float32(pct)/100)
		return OperationProgress{
			Active:   true,
			Phase:    phase,
			Percent:  pct,
			Fraction: clamp01(frac),
		}, true
	}
	if m := progressCountRE.FindStringSubmatch(line); len(m) == 2 {
		return OperationProgress{
			Active:        true,
			Phase:         strings.TrimSpace(m[1]),
			Indeterminate: true,
		}, true
	}
	return OperationProgress{}, false
}

// phaseFraction maps a single Git progress phase into a smooth overall fraction
// so a single thin bar advances steadily instead of resetting at each phase.
func phaseFraction(command, phase string, remotePrefix bool, ratio float32) float32 {
	ratio = clamp01(ratio)
	start, end := float32(0), float32(1)
	switch command {
	case "push":
		switch {
		case !remotePrefix && (phase == "Enumerating objects" || phase == "Counting objects"):
			start, end = 0.00, 0.12
		case !remotePrefix && phase == "Compressing objects":
			start, end = 0.12, 0.30
		case !remotePrefix && phase == "Writing objects":
			start, end = 0.30, 0.90
		case remotePrefix && phase == "Resolving deltas":
			start, end = 0.90, 1.00
		}
	default: // fetch, pull, clone, and other transfers
		switch {
		case remotePrefix && (phase == "Enumerating objects" || phase == "Counting objects"):
			start, end = 0.00, 0.10
		case remotePrefix && phase == "Compressing objects":
			start, end = 0.10, 0.20
		case phase == "Receiving objects":
			start, end = 0.20, 0.85
		case phase == "Resolving deltas":
			start, end = 0.85, 0.96
		case phase == "Updating files" || phase == "Checking out files" || phase == "Filtering content":
			start, end = 0.96, 1.00
		}
	}
	return start + (end-start)*ratio
}

func clamp01(v float32) float32 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
