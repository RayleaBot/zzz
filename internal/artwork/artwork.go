// Package artwork serves the upstream images the templates use. The plugin
// package ships those files, taken from the pinned upstream commits, under
// assets/<source>/; an administrator may download a source's repository into
// the data directory, whose files then take precedence, the way upstream
// plugins update their cloned resources. Templates read the files through
// render.image path resources, which the host looks up in the data directory
// first and the package second, and fall back to plain styling when a file is
// missing.
package artwork

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	maxFileBytes   int64 = 64 << 20
	maxSourceBytes int64 = 4 << 30
	stallTimeout         = time.Minute
	// Gitee serves repository archives only to download tools and answers
	// other clients with a web page.
	userAgent = "RayleaBot-game-plugin (compatible; curl)"
)

// Source is one upstream repository. Archives are tar.gz snapshots of the
// repository tried in order; only files under an Include prefix with one of
// the Extensions are kept, at their repository-relative paths.
//
// A source with Mirrors is fetched file by file instead, the first time a
// reply needs a file, as ZZZ-Plugin does with its image mirrors.
type Source struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	Archives   []string `json:"archives,omitempty"`
	Include    []string `json:"include,omitempty"`
	Extensions []string `json:"extensions,omitempty"`
	Mirrors    []string `json:"mirrors,omitempty"`
}

// Status describes one source for the chat and management views. State is
// missing, downloading, ready (downloaded) or bundled (served from the files
// the package ships), or on_demand for a source fetched file by file; Error
// keeps the last failed attempt, which leaves earlier files in place.
type Status struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	State        string `json:"state"`
	Commit       string `json:"commit,omitempty"`
	Files        int    `json:"files"`
	Bytes        int64  `json:"bytes"`
	DownloadedMS int64  `json:"downloaded_ms,omitempty"`
	Received     int64  `json:"received,omitempty"`
	Error        string `json:"error,omitempty"`
}

type record struct {
	Commit       string `json:"commit,omitempty"`
	Archive      string `json:"archive"`
	Files        int    `json:"files"`
	Bytes        int64  `json:"bytes"`
	DownloadedMS int64  `json:"downloaded_ms"`
}

type job struct {
	received atomic.Int64
}

// Store keeps each downloaded source under Root/<id>, beside the files the
// package ships under Package/<id>. Start runs downloads in the background for
// the life of the store; Close stops them.
type Store struct {
	Root string
	// Package is <package directory>/assets, empty when unknown.
	Package string
	Sources []Source
	Client  *http.Client

	mu     sync.Mutex
	jobs   map[string]*job
	errors map[string]string
	// fetching and failed serve on-demand sources: one download per file at
	// a time, and a pause of fetchPause after a file could not be fetched.
	fetching map[string]*sync.Mutex
	failed   map[string]time.Time
	ctx      context.Context
	cancel   context.CancelFunc
	running  sync.WaitGroup
}

func (s *Store) source(id string) (Source, bool) {
	for _, source := range s.Sources {
		if source.ID == id {
			return source, true
		}
	}
	return Source{}, false
}

// Start begins downloading a source, or reports the download already running.
func (s *Store) Start(id string) (Status, error) {
	source, ok := s.source(id)
	if !ok {
		return Status{}, fmt.Errorf("unknown artwork source %q", id)
	}
	if len(source.Mirrors) > 0 {
		return Status{}, errors.New("artwork source downloads files on demand")
	}
	s.mu.Lock()
	if s.ctx == nil {
		s.ctx, s.cancel = context.WithCancel(context.Background())
		s.jobs, s.errors = map[string]*job{}, map[string]string{}
	}
	if s.ctx.Err() != nil {
		s.mu.Unlock()
		return Status{}, errors.New("artwork store is closed")
	}
	if s.jobs[id] == nil {
		current := &job{}
		s.jobs[id] = current
		delete(s.errors, id)
		s.running.Add(1)
		go func() {
			defer s.running.Done()
			err := s.download(s.ctx, source, current)
			s.mu.Lock()
			delete(s.jobs, id)
			if err != nil {
				s.errors[id] = err.Error()
			}
			s.mu.Unlock()
		}()
	}
	s.mu.Unlock()
	return s.status(source), nil
}

// Close stops running downloads and waits for them to clean up.
func (s *Store) Close() {
	s.mu.Lock()
	if s.cancel == nil {
		s.ctx, s.cancel = context.WithCancel(context.Background())
	}
	s.cancel()
	s.mu.Unlock()
	s.running.Wait()
}

// Statuses lists every source in declared order.
func (s *Store) Statuses() []Status {
	result := make([]Status, 0, len(s.Sources))
	for _, source := range s.Sources {
		result = append(result, s.status(source))
	}
	return result
}

func (s *Store) status(source Source) Status {
	status := Status{ID: source.ID, Name: source.Name, State: "missing"}
	if len(source.Mirrors) > 0 {
		status.State = "on_demand"
		_ = filepath.WalkDir(filepath.Join(s.Root, source.ID), func(_ string, entry os.DirEntry, err error) error {
			if err == nil && entry.Type().IsRegular() {
				if info, infoErr := entry.Info(); infoErr == nil {
					status.Files++
					status.Bytes += info.Size()
				}
			}
			return nil
		})
		return status
	}
	// A download moves its directory in and writes its record before its job
	// ends under the lock, and Start and Delete change them under it too, so
	// reading them under the lock never sees a finished download without its
	// record.
	s.mu.Lock()
	defer s.mu.Unlock()
	if raw, err := os.ReadFile(s.recordPath(source.ID)); err == nil {
		var saved record
		if json.Unmarshal(raw, &saved) == nil {
			status.Commit, status.Files, status.Bytes, status.DownloadedMS = saved.Commit, saved.Files, saved.Bytes, saved.DownloadedMS
		}
	}
	if info, err := os.Stat(filepath.Join(s.Root, source.ID)); err == nil && info.IsDir() {
		status.State = "ready"
	} else if s.bundled(source.ID) {
		// The shipped files serve until a download replaces them.
		status.State = "bundled"
		if raw, err := os.ReadFile(filepath.Join(s.Package, source.ID+".json")); err == nil {
			var shipped record
			if json.Unmarshal(raw, &shipped) == nil {
				status.Commit, status.Files, status.Bytes = shipped.Commit, shipped.Files, shipped.Bytes
			}
		}
	}
	if current := s.jobs[source.ID]; current != nil {
		status.State, status.Received = "downloading", current.received.Load()
	}
	status.Error = s.errors[source.ID]
	return status
}

// Delete removes a downloaded source. A running download must finish first.
func (s *Store) Delete(id string) error {
	if _, ok := s.source(id); !ok {
		return fmt.Errorf("unknown artwork source %q", id)
	}
	// Move the directory aside under the lock so a download cannot start into
	// it, then remove it without holding the lock.
	discarded := filepath.Join(s.Root, "."+id+".deleted")
	s.mu.Lock()
	if s.jobs[id] != nil {
		s.mu.Unlock()
		return errors.New("artwork source is downloading")
	}
	err := os.RemoveAll(discarded)
	if err == nil {
		if err = renameRetry(filepath.Join(s.Root, id), discarded); errors.Is(err, os.ErrNotExist) {
			err = nil
		}
	}
	if err == nil {
		if err = os.Remove(s.recordPath(id)); errors.Is(err, os.ErrNotExist) {
			err = nil
		}
	}
	if err == nil {
		delete(s.errors, id)
	}
	s.mu.Unlock()
	if err != nil {
		return err
	}
	return os.RemoveAll(discarded)
}

// roots are the directories a source's files are looked up in: the
// download first, then the package.
func (s *Store) roots(sourceID string) []string {
	roots := []string{filepath.Join(s.Root, sourceID)}
	if s.Package != "" {
		roots = append(roots, filepath.Join(s.Package, sourceID))
	}
	return roots
}

// locate finds a file of a source, downloaded or shipped.
func (s *Store) locate(sourceID, name string) (string, bool) {
	if !validName(sourceID) || !validName(name) {
		return "", false
	}
	for _, root := range s.roots(sourceID) {
		file := filepath.Join(root, filepath.FromSlash(name))
		if info, err := os.Stat(file); err == nil && info.Mode().IsRegular() {
			return file, true
		}
	}
	return "", false
}

func (s *Store) bundled(id string) bool {
	if s.Package == "" || !validName(id) {
		return false
	}
	info, err := os.Stat(filepath.Join(s.Package, id))
	return err == nil && info.IsDir()
}

// File returns a file's path relative to the plugin data directory, the form
// render.image path resources take, when the source has it downloaded or
// shipped; the host reads the same path from the package when the data
// directory lacks it. Root must be <data directory>/assets.
func (s *Store) File(sourceID, name string) (string, bool) {
	if _, ok := s.locate(sourceID, name); !ok {
		return "", false
	}
	return "assets/" + sourceID + "/" + name, true
}

// Ready tells whether a source has files, downloaded or shipped.
func (s *Store) Ready(id string) bool {
	info, err := os.Stat(filepath.Join(s.Root, id))
	return validName(id) && (err == nil && info.IsDir() || s.bundled(id))
}

// Open reads a file of a source, for sources that ship indexes next to images.
func (s *Store) Open(sourceID, name string) ([]byte, error) {
	file, ok := s.locate(sourceID, name)
	if !ok {
		return nil, os.ErrNotExist
	}
	return os.ReadFile(file)
}

func (s *Store) recordPath(id string) string { return filepath.Join(s.Root, id+".json") }

func (s *Store) download(ctx context.Context, source Source, current *job) error {
	staging := filepath.Join(s.Root, "."+source.ID+".partial")
	var lastErr error
	for _, archive := range source.Archives {
		if err := os.RemoveAll(staging); err != nil {
			return err
		}
		current.received.Store(0)
		saved, err := s.extract(ctx, source, archive, staging, current)
		if err == nil {
			return s.replace(source.ID, staging, saved)
		}
		lastErr = err
		if ctx.Err() != nil {
			break
		}
	}
	_ = os.RemoveAll(staging)
	if lastErr == nil {
		lastErr = errors.New("no archive configured")
	}
	return lastErr
}

func (s *Store) extract(ctx context.Context, source Source, archive, staging string, current *job) (record, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, archive, nil)
	if err != nil {
		return record{}, err
	}
	request.Header.Set("User-Agent", userAgent)
	client := s.Client
	if client == nil {
		client = &http.Client{Transport: &http.Transport{Proxy: http.ProxyFromEnvironment, TLSHandshakeTimeout: 15 * time.Second, ResponseHeaderTimeout: 30 * time.Second}}
	}
	response, err := client.Do(request)
	if err != nil {
		return record{}, err
	}
	defer func(body io.Closer) { _ = body.Close() }(response.Body)
	if response.StatusCode != http.StatusOK {
		return record{}, fmt.Errorf("%s returned HTTP %d", request.URL.Host, response.StatusCode)
	}
	if strings.HasPrefix(response.Header.Get("Content-Type"), "text/html") {
		return record{}, fmt.Errorf("%s returned a web page instead of an archive", request.URL.Host)
	}
	// A stalled connection keeps a download alive forever; give up after a
	// minute without data and let the next archive try.
	stall := time.AfterFunc(stallTimeout, cancel)
	defer stall.Stop()
	body := &progressReader{reader: response.Body, received: &current.received, stall: stall}
	unzipped, err := gzip.NewReader(body)
	if err != nil {
		return record{}, err
	}
	saved := record{Archive: archive}
	entries := tar.NewReader(unzipped)
	for {
		header, err := entries.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return record{}, err
		}
		if header.Typeflag == tar.TypeXGlobalHeader {
			// git archive records the commit in the global header.
			saved.Commit = header.PAXRecords["comment"]
			continue
		}
		if header.Typeflag != tar.TypeReg {
			continue
		}
		_, name, ok := strings.Cut(header.Name, "/")
		if !ok || !source.keeps(name) || !validName(name) {
			continue
		}
		if header.Size > maxFileBytes || saved.Bytes+header.Size > maxSourceBytes {
			return record{}, fmt.Errorf("%s is larger than allowed", name)
		}
		if err := writeFile(filepath.Join(staging, filepath.FromSlash(name)), entries, header.Size); err != nil {
			return record{}, err
		}
		saved.Files++
		saved.Bytes += header.Size
	}
	if saved.Files == 0 {
		return record{}, fmt.Errorf("%s has no matching files", request.URL.Host)
	}
	saved.DownloadedMS = time.Now().UnixMilli()
	return saved, nil
}

func (source Source) keeps(name string) bool {
	extension := strings.ToLower(path.Ext(name))
	matched := len(source.Extensions) == 0
	for _, allowed := range source.Extensions {
		matched = matched || extension == allowed
	}
	if !matched {
		return false
	}
	if len(source.Include) == 0 {
		return true
	}
	for _, prefix := range source.Include {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

// validName accepts canonical relative slash paths whose segments are usable
// file names on Windows, the shape render.image path resources require.
func validName(name string) bool {
	if name == "" || path.Clean(name) != name || strings.HasPrefix(name, "/") {
		return false
	}
	for _, segment := range strings.Split(name, "/") {
		if segment == "." || segment == ".." || strings.ContainsAny(segment, `<>:"|?*\`) || strings.HasSuffix(segment, ".") || strings.HasSuffix(segment, " ") {
			return false
		}
		for _, character := range segment {
			if character < 0x20 {
				return false
			}
		}
		base, _, _ := strings.Cut(strings.ToUpper(segment), ".")
		switch strings.TrimRight(base, " ") {
		case "CON", "PRN", "AUX", "NUL", "COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9", "LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9":
			return false
		}
	}
	return true
}

func writeFile(target string, source io.Reader, size int64) error {
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	file, err := os.Create(target)
	if err != nil {
		return err
	}
	_, copyErr := io.CopyN(file, source, size)
	if closeErr := file.Close(); copyErr == nil {
		copyErr = closeErr
	}
	return copyErr
}

// replace swaps the finished download in for the previous copy. Windows
// refuses to rename a directory while a file inside is open, which happens
// briefly while the host copies an image for a render, so renames retry.
func (s *Store) replace(id, staging string, saved record) error {
	final := filepath.Join(s.Root, id)
	previous := filepath.Join(s.Root, "."+id+".old")
	if err := os.RemoveAll(previous); err != nil {
		return err
	}
	hadPrevious := false
	if _, err := os.Stat(final); err == nil {
		if err := renameRetry(final, previous); err != nil {
			return err
		}
		hadPrevious = true
	}
	if err := renameRetry(staging, final); err != nil {
		if hadPrevious {
			_ = renameRetry(previous, final)
		}
		return err
	}
	raw, err := json.Marshal(saved)
	if err != nil {
		return err
	}
	if err := os.WriteFile(s.recordPath(id), raw, 0o644); err != nil {
		return err
	}
	return os.RemoveAll(previous)
}

func renameRetry(from, to string) error {
	var err error
	for attempt := range 10 {
		if err = os.Rename(from, to); err == nil || errors.Is(err, os.ErrNotExist) {
			return err
		}
		time.Sleep(time.Duration(attempt+1) * 100 * time.Millisecond)
	}
	return err
}

type progressReader struct {
	reader   io.Reader
	received *atomic.Int64
	stall    *time.Timer
}

func (r *progressReader) Read(buffer []byte) (int, error) {
	count, err := r.reader.Read(buffer)
	if count > 0 {
		r.received.Add(int64(count))
		r.stall.Reset(stallTimeout)
	}
	return count, err
}

// fetchPause is how long a file that could not be fetched is not asked for
// again. On-demand sources are the images official answers link to; a CDN
// failure is usually brief, and the pages draw another picture meanwhile.
const fetchPause = 5 * time.Minute

// Fetch returns a file of an on-demand source, downloading it from the first
// mirror that has it when it is not cached yet. The path is relative to the
// plugin data directory, as File returns it.
func (s *Store) Fetch(ctx context.Context, sourceID, name string) (string, bool) {
	if path, ok := s.File(sourceID, name); ok {
		return path, true
	}
	source, ok := s.source(sourceID)
	if !ok || len(source.Mirrors) == 0 || !validName(name) {
		return "", false
	}
	key := sourceID + "/" + name
	s.mu.Lock()
	if s.fetching == nil {
		s.fetching, s.failed = map[string]*sync.Mutex{}, map[string]time.Time{}
	}
	if until, paused := s.failed[key]; paused && time.Now().Before(until) {
		s.mu.Unlock()
		return "", false
	}
	lock := s.fetching[key]
	if lock == nil {
		lock = &sync.Mutex{}
		s.fetching[key] = lock
	}
	s.mu.Unlock()
	lock.Lock()
	defer lock.Unlock()
	if path, ok := s.File(sourceID, name); ok {
		return path, true
	}
	target := filepath.Join(s.Root, sourceID, filepath.FromSlash(name))
	for _, mirror := range source.Mirrors {
		if s.fetchFile(ctx, mirror+name, target) == nil {
			return s.File(sourceID, name)
		}
		if ctx.Err() != nil {
			return "", false
		}
	}
	s.mu.Lock()
	s.failed[key] = time.Now().Add(fetchPause)
	s.mu.Unlock()
	return "", false
}

func (s *Store) fetchFile(ctx context.Context, url, target string) error {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	request.Header.Set("User-Agent", userAgent)
	client := s.Client
	if client == nil {
		client = &http.Client{Transport: &http.Transport{Proxy: http.ProxyFromEnvironment}}
	}
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer func(body io.Closer) { _ = body.Close() }(response.Body)
	if response.StatusCode != http.StatusOK || strings.HasPrefix(response.Header.Get("Content-Type"), "text/html") {
		return fmt.Errorf("%s returned HTTP %d", request.URL.Host, response.StatusCode)
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	partial := target + ".partial"
	file, err := os.Create(partial)
	if err != nil {
		return err
	}
	size, copyErr := io.Copy(file, io.LimitReader(response.Body, maxFileBytes+1))
	if closeErr := file.Close(); copyErr == nil {
		copyErr = closeErr
	}
	if copyErr == nil && (size == 0 || size > maxFileBytes) {
		copyErr = errors.New("file size is out of range")
	}
	if copyErr != nil {
		_ = os.Remove(partial)
		return copyErr
	}
	return os.Rename(partial, target)
}
