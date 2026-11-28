package delivery

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

var (
	ErrNoReleaseWindow       = errors.New("no release window is held")
	ErrReleaseWindowRequired = errors.New("release window ownership is required")
)

type ReleaseWindow struct {
	TaskID string
	Since  time.Time
}

type releaseWindowRecord struct {
	ReleaseWindow
	AcquiredAfterCompletion bool `json:",omitempty"`
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
	complete, err := s.releaseTaskComplete(id)
	if err != nil {
		return err
	}
	return s.writeReleaseWindow(releaseWindowRecord{
		ReleaseWindow:           ReleaseWindow{TaskID: id, Since: now},
		AcquiredAfterCompletion: complete,
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
	if isReleasePhase(task.Phase) && task.Phase != PhaseComplete {
		return fmt.Errorf("task %q cannot release its window during phase %q", id, task.Phase)
	}
	return os.Remove(s.releaseWindowPath())
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

	// The atomic task write that records completion also expires the hold.
	// Writers can replace the expired queue record on the next acquisition.
	if !record.AcquiredAfterCompletion {
		complete, err := s.releaseTaskComplete(record.TaskID)
		if err != nil {
			return ReleaseWindow{}, err
		}
		if complete {
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

func (s *Store) releaseTaskComplete(id string) (bool, error) {
	task, err := s.Load(id)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return task.Phase == PhaseComplete, err
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
