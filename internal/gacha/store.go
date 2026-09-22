package gacha

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
)

type Store struct {
	mu        sync.Mutex
	Directory string
	Game      string
	versions  map[string]uint64
}
type Summary struct {
	UID      string `json:"uid"`
	Region   string `json:"region"`
	Timezone int    `json:"timezone"`
	Language string `json:"lang"`
	Records  int    `json:"records"`
	Revision int64  `json:"revision"`
}

func (s *Store) filename(uid, region string) string {
	sum := sha256.Sum256([]byte(s.Game + "\x00" + uid + "\x00" + region))
	return filepath.Join(s.Directory, hex.EncodeToString(sum[:])+".json")
}
func readArchive(filename string) (Archive, error) {
	file, err := os.Open(filename)
	if err != nil {
		return Archive{}, err
	}
	defer file.Close()
	var archive Archive
	decoder := json.NewDecoder(io.LimitReader(file, 128*1024*1024))
	if decoder.Decode(&archive) != nil || decoder.Decode(new(any)) != io.EOF {
		return Archive{}, ErrInvalid
	}
	return archive, nil
}
func (s *Store) Read(uid, region string) (Archive, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return readArchive(s.filename(uid, region))
}
func (s *Store) Import(incoming Archive) (Archive, int, error) {
	return s.importVersion(incoming, nil)
}

// Snapshot also tracks removals of an absent archive during a pending sync.
func (s *Store) Snapshot(uid, region string) (Archive, uint64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	filename := s.filename(uid, region)
	archive, err := readArchive(filename)
	if errors.Is(err, os.ErrNotExist) {
		err = nil
	}
	return archive, s.versions[filename], err
}
func (s *Store) ImportIfUnchanged(incoming Archive, version uint64) (Archive, int, error) {
	return s.importVersion(incoming, &version)
}
func (s *Store) importVersion(incoming Archive, version *uint64) (Archive, int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	filename := s.filename(incoming.UID, incoming.Region)
	if version != nil && s.versions[filename] != *version {
		return Archive{}, 0, ErrConflict
	}
	existing, err := readArchive(filename)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return Archive{}, 0, err
	}
	merged, added, err := Merge(s.Game, existing, incoming)
	if err != nil {
		return Archive{}, 0, err
	}
	if err := os.MkdirAll(s.Directory, 0o700); err != nil {
		return Archive{}, 0, err
	}
	file, err := os.CreateTemp(s.Directory, ".import-*")
	if err != nil {
		return Archive{}, 0, err
	}
	defer os.Remove(file.Name())
	if err = json.NewEncoder(file).Encode(merged); err == nil {
		err = file.Sync()
	}
	closeErr := file.Close()
	if err != nil {
		return Archive{}, 0, err
	}
	if closeErr != nil {
		return Archive{}, 0, closeErr
	}
	if err = os.Rename(file.Name(), filename); err != nil {
		return Archive{}, 0, err
	}
	if s.versions == nil {
		s.versions = map[string]uint64{}
	}
	s.versions[filename]++
	return merged, added, nil
}
func (s *Store) Remove(uid, region string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	err := os.Remove(s.filename(uid, region))
	if errors.Is(err, os.ErrNotExist) {
		err = nil
	}
	if err == nil {
		if s.versions == nil {
			s.versions = map[string]uint64{}
		}
		s.versions[s.filename(uid, region)]++
	}
	return err
}
func (s *Store) List() ([]Summary, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	files, err := os.ReadDir(s.Directory)
	if errors.Is(err, os.ErrNotExist) {
		return []Summary{}, nil
	}
	if err != nil {
		return nil, err
	}
	result := []Summary{}
	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".json") {
			continue
		}
		archive, err := readArchive(filepath.Join(s.Directory, file.Name()))
		if err != nil {
			return nil, err
		}
		result = append(result, Summary{UID: archive.UID, Region: archive.Region, Timezone: archive.Timezone, Language: archive.Language, Records: len(archive.Records), Revision: archive.Revision})
	}
	slices.SortFunc(result, func(a, b Summary) int {
		if a.UID != b.UID {
			return strings.Compare(a.UID, b.UID)
		}
		return strings.Compare(a.Region, b.Region)
	})
	return result, nil
}
