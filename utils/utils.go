package utils

import (
	"path/filepath"
	"strings"

	"github.com/mitchellh/go-homedir"
)

func ExpandPath(path string) string {
	usr, _ := homedir.Dir()
	var dir string = usr
	if last := len(dir) - 1; last >= 0 && dir[last] != '/' {
		dir = dir + "/"
	}
	// Check in case of paths like "/something/~/something/"
	if path[:2] == "~/" {
		path = strings.Replace(path, "~/", dir, 1)
	}
	//	log.Printf("path = %s", path)

	// for file path separator difference between windows and unix
	path = filepath.FromSlash(path)
	return path
}
