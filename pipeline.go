package main

import (
	"fmt"
	"github.com/liserjrqlxue/simple-util"
	"os"
	"path/filepath"
	"strings"
)

func createShell(fileName, script string, args ...string) {
	file, err := os.Create(fileName)
	simple_util.CheckErr(err)
	defer simple_util.DeferClose(file)

	_, err = fmt.Fprintf(file, "#!/bin/bash\nsh %s %s\n", script, strings.Join(args, " "))
	simple_util.CheckErr(err)
}

func createDir(workdir string, sampleDirList, sampleList []string) {
	for _, sampleID := range sampleList {
		for _, subdir := range sampleDirList {
			simple_util.CheckErr(
				os.MkdirAll(
					filepath.Join(
						workdir,
						sampleID,
						subdir,
					),
					0755,
				),
			)
		}
	}
}