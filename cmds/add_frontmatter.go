package cmds

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
	"github.com/spf13/cobra"
)

var (
	dirTPL = template.Must(template.New("dir").Parse(`---
---
title: {{ .title }}
menu:
  {{- if .shared }}
  product_{{ .product }}_{{ .version }}:
  {{- else }}
  docs_{{ .version }}:
  {{- end }}
    identifier: {{ .id }}
    name: {{ .title }}
    parent: {{ .pid }}
    weight: 1
{{- if .shared }}
menu_name: product_{{ .product }}_{{ .version }}
{{- else }}
menu_name: docs_{{ .version }}
{{- end }}
---

`))

	mdTPL = template.Must(template.New("md").Parse(`---
title: {{ .title }}
menu:
  {{- if .shared }}
  product_{{ .product }}_{{ .version }}:
  {{- else }}
  docs_{{ .version }}:
  {{- end }}
    identifier: {{ .id }}
    name: {{ .title }}
    parent: {{ .pid }}
    weight: 1
{{- if .shared }}
product_name: {{ .product }}
menu_name: product_{{ .product }}_{{ .version }}
{{- else }}
menu_name: docs_{{ .version }}
{{- end }}
section_menu_id: {{ .section }}
---

`))
)

var product string
var version string
var shared bool
var section = "reference"
var skipDir bool

func NewCmdAddFrontMatter() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "add-frontmatter",
		Short:             "Add front matter",
		DisableAutoGenTag: true,
		Run: func(cmd *cobra.Command, args []string) {
			addFrontMatter(args)
		},
	}
	cmd.Flags().StringVar(&product, "product", product, "Name of product")
	cmd.Flags().StringVar(&version, "version", version, "Product version")
	cmd.Flags().BoolVar(&shared, "shared", shared, "Shared or product specific project")
	cmd.Flags().StringVar(&section, "section", section, "Website section")
	cmd.Flags().BoolVar(&skipDir, "skipDir", skipDir, "If true, skips generating dir _index.md files")
	return cmd
}

func addFrontMatter(args []string) {
	if len(args) < 1 {
		log.Fatalln("missing directory name")
	}
	for _, dir := range args {
		err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				fmt.Printf("prevent panic by handling failure accessing a path %q: %v\n", dir, err)
				return err
			}

			self := clean(strings.TrimSuffix(filepath.Base(path), ".md"))
			parent := clean(filepath.Base(filepath.Dir(path)))
			granny := clean(filepath.Base(filepath.Dir(filepath.Dir(path))))
			data := map[string]interface{}{
				"id":      id(self + " " + parent),
				"pid":     id(parent + " " + granny),
				"title":   strings.Title(parent + " " + self),
				"product": product,
				"version": version,
				"shared":  shared,
				"section": section,
			}

			if info.IsDir() {
				if !skipDir {
					var out bytes.Buffer
					err = dirTPL.Execute(&out, data)
					if err != nil {
						return err
					}
					ioutil.WriteFile(filepath.Join(path, "_index.md"), out.Bytes(), 0755)
				}
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
