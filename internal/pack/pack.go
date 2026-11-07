package pack

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

const FileName = "packman.json"

func Exists(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, FileName))
	return err == nil
}

func Load(dir string) (*Pack, error) {
	data, err := os.ReadFile(filepath.Join(dir, FileName))
	if errors.Is(err, os.ErrNotExist) {
		return nil, errors.New("no packman.json found here : run `packman init` first")
	}
	if err != nil {
		return nil, err
	}
	var p Pack
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

func Save(dir string, p Pack) error {
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, FileName), append(data, '\n'), 0o644)
}
