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
				names = append(names, path)
			}
			return nil
		})

		for _, file := range names {
			fmt.Println(file)
		}
		if len(names) != 0 {
			os.Exit(1)
		}

		if err != nil {
			fmt.Printf("error walking the path %q: %v\n", dir, err)
		}
	}
}
