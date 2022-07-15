package util

import (
	"errors"
	"fmt"
	"os"
	"path"
)

func CreateParentIfNotExist(file string) error {
	dirPath := path.Dir(file)

	if stat, err := os.Stat(dirPath); err != nil && os.IsNotExist(err) {
		if err = os.MkdirAll(dirPath, 0755); err != nil {
			return err
		}
	} else if !stat.IsDir() {
		return errors.New(fmt.Sprintf("%s is not a dir\n", dirPath))
	}
	return nil
}

func CreateFileIfNotExist(file string) error {
	if _, err := os.Stat(file); err != nil && os.IsNotExist(err) {
		if err = CreateParentIfNotExist(file); err != nil {
			return err
		}
		if _, err := os.Create(file); err != nil {
			return err
		}
	}
	return nil
}
