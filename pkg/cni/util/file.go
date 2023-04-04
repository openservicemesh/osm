package util

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"
)

// AtomicCopy copies file by reading the file then writing atomically into the target directory
func AtomicCopy(srcFilepath, targetDir, targetFilename string) error {
	//#nosec G304
	in, err := os.Open(srcFilepath)
	if err != nil {
		return err
	}
	defer func(in *os.File) {
		_ = in.Close()
	}(in)

	perm, err := in.Stat()
	if err != nil {
		return err
	}

	input, err := io.ReadAll(in)
	if err != nil {
		return err
	}

	return AtomicWrite(filepath.Join(targetDir, targetFilename), input, perm.Mode())
}

// AtomicWrite atomically by writing to a temporary file in the same directory then renaming
func AtomicWrite(path string, data []byte, mode os.FileMode) (err error) {
	tmpFile, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".tmp.")
	if err != nil {
		return
	}
	defer func() {
		if Exists(tmpFile.Name()) {
			if rmErr := os.Remove(tmpFile.Name()); rmErr != nil {
				if err != nil {
					err = fmt.Errorf("%s: %w", rmErr.Error(), err)
				} else {
					err = rmErr
				}
			}
		}
	}()

	if err = os.Chmod(tmpFile.Name(), mode); err != nil {
		return
	}

	_, err = tmpFile.Write(data)
	if err != nil {
		if closeErr := tmpFile.Close(); closeErr != nil {
			err = fmt.Errorf("%s: %w", closeErr.Error(), err)
		}
		return
	}
	if err = tmpFile.Close(); err != nil {
		return
	}

	err = os.Rename(tmpFile.Name(), path)
	return
}

// Exists checks whether the file exists
func Exists(name string) bool {
	_, err := os.Stat(name)
	return err == nil
}

const (
	// PrivateFileMode grants owner to read/write a file.
	PrivateFileMode = 0o600
)

// IsDirWriteable checks if dir is writable by writing and removing a file
// to dir. It returns nil if dir is writable.
// Inspired by etcd fileutil.
func IsDirWriteable(dir string) error {
	f := filepath.Join(dir, ".touch")
	if err := os.WriteFile(f, []byte(""), PrivateFileMode); err != nil {
		return err
	}
	return os.Remove(f)
}

// Inode returns the inode of file
func Inode(path string) (uint64, error) {
	f, err := os.Stat(path)
	if err != nil {
		return 0, fmt.Errorf("failed to get the inode of %s", path)
	}
	stat, ok := f.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, fmt.Errorf("not syscall.Stat_t")
	}
	return stat.Ino, nil
}

// DirEquals check if two directories are referring to the same directory
func DirEquals(a, b string) (bool, error) {
	aa, err := filepath.Abs(a)
	if err != nil {
		return false, err
	}
	bb, err := filepath.Abs(b)
	if err != nil {
		return false, err
	}
	return aa == bb, nil
}
