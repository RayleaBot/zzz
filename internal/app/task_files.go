package app

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/RayleaBot/plugin-zzz/internal/localdata"
)

// storedTask is a scheduled task kept in its own file.
type storedTask interface {
	taskRef() string
}

func (r Reminder) taskRef() string { return r.Ref }

// errTaskChanged stops a trigger whose task was edited, paused or removed
// while it ran; its results are not written back.
var errTaskChanged = errors.New("task changed during its trigger")

// errTaskMissing is a trigger's claim of a task that is not stored, whose
// job is left over.
var errTaskMissing = errors.New("scheduled task is not stored")

// taskFiles keeps each scheduled task in its own file under Directory. A
// trigger claims its task, makes its network requests without the store's
// lock, and writes back only that task's file, and only while nobody else
// changed it. Legacy is the single file earlier versions kept every task
// in, moved into Directory on first use.
type taskFiles[T storedTask] struct {
	mu        sync.Mutex
	Directory string
	Legacy    string
	// claimed holds each running trigger's last written copy of its task.
	claimed  map[string]T
	migrated bool
}

func (s *taskFiles[T]) file(ref string) string {
	sum := sha256.Sum256([]byte(ref))
	return filepath.Join(s.Directory, hex.EncodeToString(sum[:16])+".json")
}

// migrate moves the legacy file's tasks into their own files and keeps the
// old file renamed beside them.
func (s *taskFiles[T]) migrate() error {
	if s.migrated || s.Legacy == "" {
		return nil
	}
	items := []T{}
	if err := localdata.Read(s.Legacy, &items); err != nil {
		return err
	}
	for _, task := range items {
		if err := localdata.Write(s.file(task.taskRef()), task); err != nil {
			return err
		}
	}
	if len(items) > 0 {
		if err := os.Rename(s.Legacy, s.Legacy+".migrated"); err != nil {
			return err
		}
	}
	s.migrated = true
	return nil
}

func (s *taskFiles[T]) read() ([]T, error) {
	if err := s.migrate(); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(s.Directory)
	if errors.Is(err, os.ErrNotExist) {
		return []T{}, nil
	}
	if err != nil {
		return nil, err
	}
	items := []T{}
	for _, entry := range entries {
		if !entry.Type().IsRegular() || strings.HasPrefix(entry.Name(), ".") || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		var task T
		if err := localdata.Read(filepath.Join(s.Directory, entry.Name()), &task); err != nil {
			return nil, err
		}
		items = append(items, task)
	}
	slices.SortFunc(items, func(a, b T) int { return strings.Compare(a.taskRef(), b.taskRef()) })
	return items, nil
}

func (s *taskFiles[T]) List() ([]T, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.read()
}

// edit changes the task list under the store's lock, index being ref's
// position or -1, and writes only the tasks that changed.
func (s *taskFiles[T]) edit(ref string, fn func(*[]T, int) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	before, err := s.read()
	if err != nil {
		return err
	}
	items := slices.Clone(before)
	if err = fn(&items, slices.IndexFunc(items, func(task T) bool { return task.taskRef() == ref })); err != nil {
		return err
	}
	kept := map[string]bool{}
	for _, task := range items {
		kept[task.taskRef()] = true
		old := slices.IndexFunc(before, func(item T) bool { return item.taskRef() == task.taskRef() })
		if old >= 0 && sameTask(before[old], task) {
			continue
		}
		if err = localdata.Write(s.file(task.taskRef()), task); err != nil {
			return err
		}
	}
	for _, task := range before {
		if !kept[task.taskRef()] {
			if err = os.Remove(s.file(task.taskRef())); err != nil && !errors.Is(err, os.ErrNotExist) {
				return err
			}
		}
	}
	return nil
}

func sameTask[T any](a, b T) bool {
	left, _ := json.Marshal(a)
	right, _ := json.Marshal(b)
	return string(left) == string(right)
}

// claim reads a task for a trigger, unless another trigger of it is still
// running; errTaskMissing when the task is not stored.
func (s *taskFiles[T]) claim(ref string) (T, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var task T
	if err := s.migrate(); err != nil {
		return task, false, err
	}
	if _, running := s.claimed[ref]; running {
		return task, false, nil
	}
	if err := localdata.Read(s.file(ref), &task); err != nil {
		return task, false, err
	}
	if ref == "" || task.taskRef() != ref {
		return task, false, errTaskMissing
	}
	if s.claimed == nil {
		s.claimed = map[string]T{}
	}
	s.claimed[ref] = task
	return task, true, nil
}

func (s *taskFiles[T]) release(ref string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.claimed, ref)
}

// save writes a claimed task back while its file is still what this trigger
// last wrote, and returns errTaskChanged otherwise.
func (s *taskFiles[T]) save(task T) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	var current T
	if err := localdata.Read(s.file(task.taskRef()), &current); err != nil {
		return err
	}
	if base, ok := s.claimed[task.taskRef()]; !ok || !sameTask(current, base) {
		return errTaskChanged
	}
	if err := localdata.Write(s.file(task.taskRef()), task); err != nil {
		return err
	}
	s.claimed[task.taskRef()] = task
	return nil
}

// ReminderStore keeps reminders and account tasks, each in its own file.
type ReminderStore struct {
	taskFiles[Reminder]
}

func reminderStore(directory string) *ReminderStore {
	return &ReminderStore{taskFiles[Reminder]{Directory: filepath.Join(directory, "reminders"), Legacy: filepath.Join(directory, "reminders.json")}}
}
