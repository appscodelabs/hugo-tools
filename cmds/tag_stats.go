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
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

	"github.com/gohugoio/hugo/parser"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"gomodules.xyz/encoding/json/query"
)

func NewCmdTagStats() *cobra.Command {
	invalidOnly := false
	cmd := &cobra.Command{
		Use:               "tag-stats",
		Short:             "Print list of tags",
		DisableAutoGenTag: true,
		Run: func(cmd *cobra.Command, args []string) {
			if err := calculateTagStats(args, invalidOnly); err != nil {
				fmt.Fprintln(os.Stderr, "\nerror:", err)
				os.Exit(1)
			}
		},
	}
	cmd.Flags().BoolVar(&invalidOnly, "invalid-only", invalidOnly, "Only report invalid tags")
	return cmd
}

func calculateTagStats(args []string, invalidOnly bool) error {
	if len(args) < 1 {
		return errors.New("missing directory name")
	}

	stats := map[string]int{}
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

			fm, err := page.Metadata()
			if err != nil {
				return err
			}
			tags, _, err := query.NestedStringSlice(fm, "tags")
			if err != nil {
				return err
			}
			for _, tag := range tags {
				stats[tag]++
			}

			return nil
		})
		if err != nil {
			return errors.Wrapf(err, "error walking the path %q", dir)
		}
	}

	keys := make([]string, 0, len(stats))
	for tag := range stats {
		if invalidOnly {
			fields := strings.FieldsFunc(tag, func(r rune) bool {
				return unicode.IsSpace(r) || r == '-' || r == '_'
			})
			// too many words or not in lower case
			if len(fields) > 3 || strings.ToLower(tag) != tag {
				keys = append(keys, tag)
			}
		} else {
			keys = append(keys, tag)
		}
	}
	sort.Strings(keys)
	fmt.Println("TAG:___________________________________")
	for _, key := range keys {
		fmt.Println(key, stats[key])
	}
	if invalidOnly && len(keys) > 0 {
		return errors.Errorf("%d invalid tags found", len(keys))
	}
	return nil
}
