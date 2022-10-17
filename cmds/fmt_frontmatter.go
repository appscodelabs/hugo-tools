/*
Copyright AppsCode Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cmds

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/gohugoio/hugo/parser"
	"github.com/spf13/cobra"
	"gomodules.xyz/sets"
	"gopkg.in/yaml.v2"
)

func NewCmdFormatFrontMatter() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "fmt-frontmatter",
		Short:             "Format front matter",
		DisableAutoGenTag: true,
		Run: func(cmd *cobra.Command, args []string) {
			fmtFrontMatter(args)
		},
	}
	return cmd
}

func fmtFrontMatter(args []string) {
	if len(args) < 1 {
		log.Fatalln("missing directory name")
	}
	for _, dir := range args {
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

			data, err := os.ReadFile(path)
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
				return nil
			}

			var obj yaml.MapSlice
			err = yaml.Unmarshal(fm, &obj)
			if err != nil {
				return nil // front matter does not have YAML format, so do nothing
			}
			for i, item := range obj {
				if item.Key.(string) == "tags" {
					valTags := item.Value.([]interface{})
					tags := sets.NewString()
					for _, tag := range valTags {
						tags.Insert(strings.ToLower(tag.(string)))
					}
					obj[i].Value = tags.List()
				}
			}
			ffm, err := yaml.Marshal(obj)
			if err != nil {
				return err
			}

			buf.Reset()
			buf.WriteString("---\n")
			buf.Write(ffm)
			buf.WriteString("---\n\n")
			buf.Write(page.Content())
			return os.WriteFile(path, buf.Bytes(), 0o644)
		})
		if err != nil {
			fmt.Printf("error walking the path %q: %v\n", dir, err)
		}
	}
}
