package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/gohugoio/hugo/parser"
)

func main() {
	args := os.Args
	if len(args) < 2 {
		log.Fatalln("missing directory name")
	}
	for _, dir := range os.Args[1:] {
		err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				fmt.Printf("prevent panic by handling failure accessing a path %q: %v\n", dir, err)
				return err
			}
			if info.IsDir() {
				return nil
			}

			filename := filepath.Join(path, info.Name())
			f, err := os.Open(filename)
			if err != nil {
				return err
			}
			defer f.Close()

			page, err := parser.ReadFrom(f)
			if err != nil {
				return err
			}
			fm := page.FrontMatter()
			if len(fm) == 0 {
				fmt.Println(filename)
			}
			return nil
		})
		if err != nil {
			fmt.Printf("error walking the path %q: %v\n", dir, err)
		}
	}

	parser.ReadFrom(nil)
}
