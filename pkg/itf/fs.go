package itf

import (
	"bytes"
	"compress/zlib"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func GetFileSHA256(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func IsEmptyDir(name string) (bool, error) {
	f, err := os.Open(name)
	if err != nil {
		return false, err
	}
	defer f.Close()

	_, err = f.Readdirnames(1)
	if err == io.EOF {
		return true, nil
	}
	return false, err
}

type PathResolver struct {
	wd string
}

func NewPathResolver() (*PathResolver, error) {
	wd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("could not get current working directory: %w", err)
	}
	return &PathResolver{wd: wd}, nil
}

func (r *PathResolver) Resolve(relativePath string) string {
	if filepath.IsAbs(relativePath) {
		return filepath.Clean(relativePath)
	}
	return filepath.Join(r.wd, relativePath)
}

func (r *PathResolver) ResolveExisting(relativePath string) string {
	path := r.Resolve(relativePath)
	if _, err := os.Stat(path); err == nil {
		return path
	}
	return ""
}

func GetFileActionsAndDirs(targetPaths []string, renameDestinations map[string]struct{}) (map[string]string, map[string]struct{}) {
	fileActions := make(map[string]string)
	dirsToCreate := make(map[string]struct{})

	for _, path := range targetPaths {
		_, isRenameDest := renameDestinations[path]
		_, err := os.Stat(path)
		exists := !os.IsNotExist(err)

		if !isRenameDest {
			if exists {
				fileActions[path] = "modify"
			} else {
				fileActions[path] = "create"
			}
		}

		dir := filepath.Dir(path)
		if dir != "." && dir != "/" {
			if _, err := os.Stat(dir); os.IsNotExist(err) {
				dirsToCreate[dir] = struct{}{}
			}
		}
	}
	return fileActions, dirsToCreate
}

func CreateDirs(dirs map[string]struct{}) error {
	for dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("error creating directory '%s': %w", dir, err)
		}
	}
	return nil
}

func TrashFile(path string, trashPath string, wd string) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return err
	}

	relPath, err := filepath.Rel(wd, absPath)
	if err != nil {
		relPath = filepath.Base(absPath)
	}

	destPath := filepath.Join(trashPath, relPath)
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return err
	}

	content, err := os.ReadFile(absPath)
	if err != nil {
		return err
	}

	var b bytes.Buffer
	w := zlib.NewWriter(&b)
	if _, err := w.Write(content); err != nil {
		return err
	}
	w.Close()

	if err := os.WriteFile(destPath, b.Bytes(), 0644); err != nil {
		return err
	}

	return os.Remove(absPath)
}

func RestoreFileFromTrash(originalPath string, trashPath string, wd string) error {
	absPath, err := filepath.Abs(originalPath)
	if err != nil {
		return err
	}

	relPath, err := filepath.Rel(wd, absPath)
	if err != nil {
		relPath = filepath.Base(originalPath)
	}

	srcPath := filepath.Join(trashPath, relPath)
	data, err := os.ReadFile(srcPath)
	if err != nil {
		return err
	}

	if !isZlibCompressed(data) {
		return os.Rename(srcPath, absPath)
	}

	r, err := zlib.NewReader(bytes.NewReader(data))
	if err != nil {
		return err
	}
	defer r.Close()

	content, err := io.ReadAll(r)
	if err != nil {
		return err
	}

	if err := os.WriteFile(absPath, content, 0644); err != nil {
		return err
	}

	return os.Remove(srcPath)
}

func WriteBlob(dir string, hash string, content []byte) error {
	blobDir := filepath.Join(dir, "blobs")
	if err := os.MkdirAll(blobDir, 0755); err != nil {
		return err
	}

	var b bytes.Buffer
	w := zlib.NewWriter(&b)
	if _, err := w.Write(content); err != nil {
		return err
	}
	w.Close()

	return os.WriteFile(filepath.Join(blobDir, hash), b.Bytes(), 0644)
}

func ReadBlob(dir string, hash string) ([]byte, error) {
	if hash == "" {
		return []byte{}, nil
	}

	data, err := os.ReadFile(filepath.Join(dir, "blobs", hash))
	if err != nil {
		return nil, err
	}

	if !isZlibCompressed(data) {
		return data, nil
	}

	r, err := zlib.NewReader(bytes.NewReader(data))
	if err != nil {
		return data, nil
	}
	defer r.Close()

	return io.ReadAll(r)
}

func isZlibCompressed(data []byte) bool {
	return len(data) > 2 && data[0] == 0x78
}
