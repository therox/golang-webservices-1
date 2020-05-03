package main

import (
	"fmt"
	"io"
	"os"
	"sort"
)

func dirTree(out io.Writer, path string, printFiles bool) error {
	// Начинаем обработку
	var currentLevel = 0
	err := walk(out, path, printFiles, currentLevel, "")
	return err
}

func walk(out io.Writer, path string, printFiles bool, currentLevel int, indent string) error {
	d, err := os.Open(path)
	if err != nil {
		return err
	}
	defer d.Close()
	content, err := d.Readdir(0)
	if err != nil {
		return err
	}

	requiredContent := make([]os.FileInfo, 0)
	for i := range content {
		if !content[i].IsDir() && !printFiles {
			continue
		}
		requiredContent = append(requiredContent, content[i])
	}
	sort.Slice(requiredContent, func(i, j int) bool {
		return requiredContent[i].Name() < requiredContent[j].Name()
	})
	var indentPath string

	for i := range requiredContent {

		if i < len(requiredContent)-1 {
			indentPath = indent + "├───"
		} else {
			indentPath = indent + "└───"
		}
		fName := requiredContent[i].Name()
		if !requiredContent[i].IsDir() {
			if requiredContent[i].Size() == 0 {
				fName += " (empty)"
			} else {
				fName += fmt.Sprintf(" (%db)", requiredContent[i].Size())
			}
		}
		fmt.Fprintln(out, indentPath+fName)

		if requiredContent[i].IsDir() {
			var newIndent = indent
			if i < len(requiredContent)-1 {
				newIndent += "│\t"
			} else {
				newIndent += "\t"
			}
			err = walk(out, path+string(os.PathSeparator)+requiredContent[i].Name(), printFiles, currentLevel+1, newIndent)
		}
	}
	return nil
}

func main() {
	out := os.Stdout
	if !(len(os.Args) == 2 || len(os.Args) == 3) {
		panic("usage go run main.go . [-f]")
	}
	path := os.Args[1]
	printFiles := len(os.Args) == 3 && os.Args[2] == "-f"
	err := dirTree(out, path, printFiles)
	if err != nil {
		panic(err.Error())
	}
}
