package base

import (
	"errors"
	"os"
	"path"
	"runtime"
)

func Cwd() string {
	var cwd string
	_, filename, _, ok := runtime.Caller(1)
	if ok {
		cwd = path.Dir(filename)
	}
	return cwd
}
func Mkdir(path string) error {
	fi, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			err = os.Mkdir(path, os.ModePerm)
		} else {
			return err
		}
	} else if !fi.IsDir() {
		return errors.New("not direcotory")
	}
	return err
}
func IsDir(path string) bool {
	if len(path) == 0 {
		return false
	}
	fi, err := os.Stat(path)
	if err == nil && fi.IsDir() {
		return true
	}
	return false
}

func IsFileExist(path string) bool {
	if len(path) == 0 {
		return false
	}
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}
	return false
}
