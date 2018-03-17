package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"text/template"

	"github.com/gohugoio/hugo/parser"
)

var (
	dirTPL = template.Must(template.New("dir").Parse(`---
---
title: {{ .title }}
menu:
  product_voyager_6.0.0-rc.0:
    identifier: {{ .id }}
    name: {{ .title }}
    parent: {{ .pid }}
    weight: 1
menu_name: product_voyager_6.0.0-rc.0
---

`))

	mdTPL = template.Must(template.New("md").Parse(`---
title: {{ .title }}
menu:
  product_voyager_6.0.0-rc.0:
    identifier: {{ .id }}
    name: {{ .title }}
    parent: {{ .pid }}
    weight: 1
product_name: voyager
menu_name: product_voyager_6.0.0-rc.0
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
				"id":    id(self + " " + parent),
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
