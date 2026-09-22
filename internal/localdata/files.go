// Package localdata writes plugin-owned noncredential data. Callers serialize
// access to each file; it does not provide a second source of host state.
package localdata

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
)

func Read(path string, out any) error {
	file, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	defer file.Close()
	d := json.NewDecoder(io.LimitReader(file, 8*1024*1024))
	if err = d.Decode(out); err != nil {
		return err
	}
	if d.Decode(new(any)) != io.EOF {
		return errors.New("invalid local data")
	}
	return nil
}
func Write(path string, value any) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	if len(raw) > 8*1024*1024 {
		return errors.New("local data limit exceeded")
	}
	if err = os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	file, err := os.CreateTemp(filepath.Dir(path), ".write-")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	if err = file.Chmod(0600); err == nil {
		_, err = file.Write(raw)
	}
	if err == nil {
		err = file.Sync()
	}
	closed := file.Close()
	if err != nil {
		return err
	}
	if closed != nil {
		return closed
	}
	return os.Rename(file.Name(), path)
}
