package delivery

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Store persists tasks under the tasks directory in Root. Writers sharing a
// root take an advisory file lock across processes. Load and List stay
// lock-free because each task is published with an atomic rename.
type Store struct {
	Root string
}

func NewStore(root string) *Store {
	return &Store{Root: root}
}

func (s *Store) Create(t Task) error {
	path, err := s.taskPath(t.ID)
	if err != nil {
		return err
	}
	lock, err := s.lockWrites()
	if err != nil {
		return err
	}
	defer lock.Close()

	if _, err := os.Lstat(path); err == nil {
		return fmt.Errorf("task %q already exists: %w", t.ID, os.ErrExist)
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return writeTask(path, t)
}

// Load returns an error matching os.ErrNotExist when there is no task for id.
func (s *Store) Load(id string) (Task, error) {
	path, err := s.taskPath(id)
	if err != nil {
		return Task{}, err
	}
	return readTask(path, id)
}

func (s *Store) Save(t Task) error {
	path, err := s.taskPath(t.ID)
	if err != nil {
		return err
	}
	lock, err := s.lockWrites()
	if err != nil {
		return err
	}
	defer lock.Close()
	return writeTask(path, t)
}

// Update holds the write lock while fn runs. The callback can change the
// supplied task and read the store, but must not call other store writers.
func (s *Store) Update(id string, fn func(*Task) error) (Task, error) {
	path, err := s.taskPath(id)
	if err != nil {
		return Task{}, err
	}
	lock, err := s.lockWrites()
	if err != nil {
		return Task{}, err
	}
	defer lock.Close()

	t, err := readTask(path, id)
	if err != nil {
		return Task{}, err
	}
	if err := fn(&t); err != nil {
		return Task{}, err
	}
	if t.ID != id {
		return Task{}, fmt.Errorf("cannot change task ID from %q to %q", id, t.ID)
	}
	if err := writeTask(path, t); err != nil {
		return Task{}, err
	}
	return t, nil
}

func (s *Store) List() ([]Task, error) {
	dir := s.tasksDir()
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return []Task{}, nil
	}
	if err != nil {
		return nil, err
	}

	tasks := make([]Task, 0, len(entries))
	for _, entry := range entries {
		id, ok := strings.CutSuffix(entry.Name(), ".json")
		if !ok || entry.IsDir() {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		if err := validateTaskID(id); err != nil {
			return nil, fmt.Errorf("read task %s: %w", path, err)
		}
		task, err := readTask(path, id)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}
	sort.Slice(tasks, func(i, j int) bool {
		if order := tasks[i].CreatedAt.Compare(tasks[j].CreatedAt); order != 0 {
			return order < 0
		}
		return tasks[i].ID < tasks[j].ID
	})
	return tasks, nil
}

// Closing the returned file releases the lock. Keep the lock file on disk so
// every process locking this root uses the same file.
func (s *Store) lockWrites() (*os.File, error) {
	path := filepath.Join(s.Root, ".tasks.lock")
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	if err := lockTaskStoreFile(file); err != nil {
		file.Close()
		return nil, fmt.Errorf("lock task store %s: %w", path, err)
	}
	return file, nil
}

func (s *Store) tasksDir() string {
	return filepath.Join(s.Root, "tasks")
}

func (s *Store) taskPath(id string) (string, error) {
	if err := validateTaskID(id); err != nil {
		return "", err
	}
	return filepath.Join(s.tasksDir(), id+".json"), nil
}

func readTask(path, id string) (Task, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Task{}, err
	}
	var task Task
	if err := json.Unmarshal(data, &task); err != nil {
		return Task{}, fmt.Errorf("decode task %s: %w", path, err)
	}
	if task.ID != id {
		return Task{}, fmt.Errorf("decode task %s: task ID %q does not match filename", path, task.ID)
	}
	return task, nil
}

func writeTask(path string, task Task) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	info, err := os.Stat(dir)
	if err != nil {
		return err
	}
	if info.Mode().Perm() != 0700 {
		if err := os.Chmod(dir, 0700); err != nil {
			return err
		}
	}
	f, err := os.CreateTemp(dir, ".task-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())

	encoder := json.NewEncoder(f)
	encoder.SetIndent("", "  ")
	encodeErr := encoder.Encode(task)
	closeErr := f.Close()
	if encodeErr != nil {
		return fmt.Errorf("write task %s: %w", path, encodeErr)
	}
	if closeErr != nil {
		return closeErr
	}
	return os.Rename(f.Name(), path)
}
