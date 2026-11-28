package delivery

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var (
	ErrNoReleaseWindow         = errors.New("no release window is held")
	ErrReleaseWindowRequired   = errors.New("release window ownership is required")
	ErrReleaseWindowBlocked    = errors.New("blocked release window requires recovery")
	ErrRecoveryDetailsRequired = errors.New("release window recovery requires -by and -reason")
)

const releaseWindowRecovered = "recover_release_window"

type ReleaseWindow struct {
	TaskID string
	Since  time.Time
}

type releaseWindowRecord struct {
	ReleaseWindow
	TaskHistoryLength int `json:",omitempty"`
}

func (s *Store) AcquireReleaseWindow(id string, now time.Time) error {
	if err := validateTaskID(id); err != nil {
		return err
	}
	lock, err := s.lockWrites()
	if err != nil {
		return err
	}
	defer lock.Close()

	task, err := s.Load(id)
	if err != nil {
		return err
	}
	if task.Phase == PhaseComplete || task.Status == StatusCancelled {
		return fmt.Errorf("task %q is complete or cancelled and cannot acquire the release window", id)
	}
	window, err := s.CurrentReleaseWindow()
	if err == nil {
		if window.TaskID == id {
			return nil
		}
		return fmt.Errorf("release window is held by task %q", window.TaskID)
	}
	if !errors.Is(err, ErrNoReleaseWindow) {
		return err
	}
	return s.writeReleaseWindow(releaseWindowRecord{
		ReleaseWindow:     ReleaseWindow{TaskID: id, Since: now},
		TaskHistoryLength: len(task.History),
	})
}

func (s *Store) ReleaseReleaseWindow(id string) error {
	if err := validateTaskID(id); err != nil {
		return err
	}
	lock, err := s.lockWrites()
	if err != nil {
		return err
	}
	defer lock.Close()

	window, err := s.CurrentReleaseWindow()
	if err != nil {
		return err
	}
	if window.TaskID != id {
		return fmt.Errorf("release window is held by task %q", window.TaskID)
	}
	task, err := s.Load(id)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if task.Status == StatusBlocked {
		return ErrReleaseWindowBlocked
	}
	if isReleasePhase(task.Phase) && task.Phase != PhaseComplete {
		return fmt.Errorf("task %q cannot release its window during phase %q", id, task.Phase)
	}
	return os.Remove(s.releaseWindowPath())
}

// RecoverReleaseWindow records a maintainer's recovery after verifying that
// main stayed at or was returned to its release baseline. The task stays blocked.
func (s *Store) RecoverReleaseWindow(id, by, reason string) error {
	by, reason = strings.TrimSpace(by), strings.TrimSpace(reason)
	if by == "" || reason == "" {
		return ErrRecoveryDetailsRequired
	}
	path, err := s.taskPath(id)
	if err != nil {
		return err
	}
	lock, err := s.lockWrites()
	if err != nil {
		return err
	}
	defer lock.Close()

	task, err := s.Load(id)
	if err != nil {
		return err
	}
	if task.Status != StatusBlocked {
		return fmt.Errorf("task %q must be blocked to recover its window", id)
	}
	if err := s.requireReleaseWindow(task); err != nil {
		return err
	}
	task.recordEvent(releaseWindowRecovered, fmt.Sprintf("%s: %s", by, reason), time.Now().UTC())
	task.StoreRevision++
	if err := writeTask(path, task); err != nil {
		return err
	}

	// The atomic task write already expires the hold. A leftover queue record
	// is replaced at the next acquisition, just as when a task completes.
	_ = os.Remove(s.releaseWindowPath())
	return nil
}

func (s *Store) CurrentReleaseWindow() (ReleaseWindow, error) {
	data, err := os.ReadFile(s.releaseWindowPath())
	if errors.Is(err, os.ErrNotExist) {
		return ReleaseWindow{}, ErrNoReleaseWindow
	}
	if err != nil {
		return ReleaseWindow{}, err
	}
	var record releaseWindowRecord
	if err := json.Unmarshal(data, &record); err != nil {
		return ReleaseWindow{}, fmt.Errorf("decode release window: %w", err)
	}
	if err := validateTaskID(record.TaskID); err != nil {
		return ReleaseWindow{}, fmt.Errorf("decode release window: %w", err)
	}

	// Publishing completion or a recovery event atomically expires this hold.
	// The history offset distinguishes recovery of older windows from this one.
	task, err := s.Load(record.TaskID)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return ReleaseWindow{}, err
	}
	if task.Phase == PhaseComplete {
		return ReleaseWindow{}, ErrNoReleaseWindow
	}
	if record.TaskHistoryLength < 0 {
		return ReleaseWindow{}, errors.New("invalid release window history offset")
	}
	for i := record.TaskHistoryLength; i < len(task.History); i++ {
		if task.History[i].Action == releaseWindowRecovered {
			return ReleaseWindow{}, ErrNoReleaseWindow
		}
	}
	return record.ReleaseWindow, nil
}

// AdvanceTask persists a candidate change and phase transition together.
func (s *Store) AdvanceTask(id string, to Phase, candidate string, now time.Time) (Task, error) {
	return s.Update(id, func(task *Task) error {
		if candidate != "" && candidate != task.Candidate {
			if err := task.SetCandidate(candidate, now); err != nil {
				return err
			}
		}
		return task.Advance(to, now)
	})
}

func (s *Store) requireReleaseWindow(task Task) error {
	window, err := s.CurrentReleaseWindow()
	if errors.Is(err, ErrNoReleaseWindow) {
		return fmt.Errorf("task %q at phase %q: %w: %w", task.ID, task.Phase, ErrReleaseWindowRequired, err)
	}
	if err != nil {
		return err
	}
	if window.TaskID != task.ID {
		return fmt.Errorf("task %q at phase %q: %w: held by task %q", task.ID, task.Phase, ErrReleaseWindowRequired, window.TaskID)
	}
	return nil
}

func (s *Store) releaseWindowPath() string {
	return filepath.Join(s.Root, "queue.json")
}

// writeReleaseWindow publishes a replacement while the caller holds the writer lock.
func (s *Store) writeReleaseWindow(record releaseWindowRecord) error {
	file, err := os.CreateTemp(s.Root, ".queue-*")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	defer file.Close()

	if err := file.Chmod(0600); err != nil {
		return err
	}
	if err := json.NewEncoder(file).Encode(record); err != nil {
		return fmt.Errorf("write release window: %w", err)
	}
	if err := file.Close(); err != nil {
		return err
	}
	return os.Rename(file.Name(), s.releaseWindowPath())
}
