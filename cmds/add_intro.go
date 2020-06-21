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
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/gohugoio/hugo/parser"
	"github.com/spf13/cobra"
)

func NewCmdAddIntro() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "add-intro",
		Short:             "Add intro",
		DisableAutoGenTag: true,
		Run: func(cmd *cobra.Command, args []string) {
			addIntro(args)
		},
	}
	return cmd
}

func addIntro(args []string) {
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

			data, err := ioutil.ReadFile(path)
			if err != nil {
				return err
			}
			buf := bytes.NewBuffer(data)

			page, err := parser.ReadFrom(buf)
			if err != nil {
				return err
			}

			var b2 bytes.Buffer
			b2.Write(page.FrontMatter())
			c := page.Content()
			bytes.TrimSpace(c)
			b2.WriteString("> New to Voyager? Please start [here](/docs/concepts/overview.md).")
			b2.WriteRune('\n')
			b2.WriteRune('\n')
			b2.Write(c)
			return ioutil.WriteFile(path, b2.Bytes(), 0755)
		})
		if err != nil {
			fmt.Printf("error walking the path %q: %v\n", dir, err)
		}
	}
}
