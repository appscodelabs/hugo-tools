package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/gohugoio/hugo/parser"
	"text/template"
)

func main() {
	addFrontMatter()
}

var (
	dirTPL = template.Must(template.New("dir").Parse(`---
title: {{ .title }}
menu:
  docs_0.8.0-beta.2:
    identifier: {{ .id }}
    name: {{ .title }}
    parent: {{ .pid }}
    weight: 10
menu_name: docs_0.8.0-beta.2
---
`))

	mdTPL = template.Must(template.New("md").Parse(`---
title: {{ .title }}
menu:
  docs_0.8.0-beta.2:
    identifier: {{ .id }}
    name: {{ .title }}
    parent: {{ .pid }}
    weight: 10
menu_name: docs_0.8.0-beta.2
section_menu_id: guides
---
`))
)

func addFrontMatter() {
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

			self := clean(strings.TrimSuffix(filepath.Base(path), ".md"))
			parent := clean(filepath.Base(filepath.Dir(path)))
			granny := clean(filepath.Base(filepath.Dir(filepath.Dir(path))))
			data := map[string]string{
				"id":    id(self+" "+parent),
				"pid":   id(parent + " " + granny),
				"title": strings.Title(parent + " " + self),
			}


			if info.IsDir() {
				var out bytes.Buffer
				err = dirTPL.Execute(&out, data)
				if err != nil {
					return err
				}
				ioutil.WriteFile(filepath.Join(path, "_index.md"), out.Bytes(), 0755)
				return nil
			}
			if !strings.HasSuffix(path, ".md") {
				return nil
			}

			content, err := ioutil.ReadFile(path)
			if err != nil {
				return err
			}
			buf := bytes.NewBuffer(content)

			page, err := parser.ReadFrom(buf)
			if err != nil {
				return err
			}
			fm := page.FrontMatter()
			if len(fm) == 0 {
				var out bytes.Buffer
				err := mdTPL.Execute(&out, data)
				if err != nil {
					log.Fatalln(path, "err: ", err)
				}

				err = ioutil.WriteFile(path, []byte(out.String()+string(content)), 0755)
				if err != nil {
					log.Fatalln(path, "err: ", err)
				}
			}
			return nil
		})
		if err != nil {
			fmt.Printf("error walking the path %q: %v\n", dir, err)
		}
	}
}

func clean(s string) string {
	s = strings.Replace(s, "_", " ", -1)
	s = strings.Replace(s, "-", " ", -1)
	return s
}

func id(s string) string {
	return strings.ToLower(strings.Replace(s, " ", "-", -1))
}

func checkFrontMatter() {
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
			if !strings.HasSuffix(path, ".md") {
				return nil
			}

			data, err := ioutil.ReadFile(path)
			if err != nil {
				return err
			}
			buf := bytes.NewBuffer(data)

			page, err := parser.ReadFrom(buf)
			if err != nil {
				return err
			}
			fm := page.FrontMatter()
			if len(fm) == 0 {
				fmt.Println(path)
			}
			return nil
		})
		if err != nil {
			fmt.Printf("error walking the path %q: %v\n", dir, err)
		}
	}
}
