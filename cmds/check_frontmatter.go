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
)

func NewCmdCheckFrontMatter() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "check-frontmatter",
		Short:             "Check front matter",
		DisableAutoGenTag: true,
		Run: func(cmd *cobra.Command, args []string) {
			checkFrontMatter(args)
		},
	}
	return cmd
}

func checkFrontMatter(args []string) {
	if len(args) < 1 {
		log.Fatalln("missing directory name")
	}
	var names []string
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
				names = append(names, path)
			}
			return nil
		})
		if err != nil {
			fmt.Printf("error walking the path %q: %v\n", dir, err)
		}
		for _, file := range names {
			fmt.Println(file)
		}
		if len(names) != 0 {
			os.Exit(1)
		}
	}
}
