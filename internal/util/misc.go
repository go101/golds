package util

import (
	"os"
)

func WorkingDirectory() string {
	wd, err := os.Getwd()
	if err != nil {
		return ":/\\" // an invalid dir path
	}
	return wd
}
