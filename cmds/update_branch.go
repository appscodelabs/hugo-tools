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
	"encoding/json"
	"fmt"
	"os"

	saapi "github.com/appscode/static-assets/api"
	"github.com/spf13/cobra"
	"sigs.k8s.io/yaml"
)

var (
	filename string
	branch   string
)

func NewCmdUpdateBranch() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "update-branch",
		Short:             "Update branch name for latest version",
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return updateBranch()
		},
	}
	cmd.Flags().StringVar(&filename, "filename", filename, "Path to product file")
	cmd.Flags().StringVar(&branch, "branch", branch, "Product branch or git commit sha or git tag")
	return cmd
}

func updateBranch() error {
	if !Exists(filename) {
		// Avoid missing product files like stash-catalog key
		return nil
	}

	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}
	var prod saapi.Product
	err = yaml.Unmarshal(data, &prod)
	if err != nil {
		return err
	}

	fmt.Println(prod.LatestVersion)
	for i, v := range prod.Versions {
		if v.Version == prod.LatestVersion {
			v.Branch = branch
			prod.Versions[i] = v
			break
		}
	}

	var buf bytes.Buffer
	e := json.NewEncoder(&buf)
	e.SetEscapeHTML(false)
	e.SetIndent("", "  ")
	err = e.Encode(prod)
	if err != nil {
		return err
	}
	return os.WriteFile(filename, buf.Bytes(), 0o644)
}

// Exists reports whether the named file or directory Exists.
func Exists(name string) bool {
	if _, err := os.Stat(name); os.IsNotExist(err) {
		return false
	}
	return true
}
