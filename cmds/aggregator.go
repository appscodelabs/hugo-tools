package cmds

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	shell "github.com/codeskyblue/go-sh"
	"github.com/gohugoio/hugo/helpers"
	"github.com/gohugoio/hugo/parser"
	"github.com/spf13/cast"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type ProductVersion struct {
	Branch    string `json:"branch"`
	HostDocs  bool   `json:"hostDocs"`
	VDropdown bool   `json:"v-dropdown,omitempty"`
	DocsDir   string `json:"docsDir,omitempty"` // default: "docs"
}

type Product struct {
	Name          string           `json:"-"`
	URL           string           `json:"url"`
	Versions      []ProductVersion `json:"versions"`
	LatestVersion string           `json:"latestVersion"`
	GithubURL     string           `json:"githubUrl"`
}

type SharedWebsite struct {
	Products map[string]Product `json:"products"`
}

func NewCmdDocsAggregator() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "docs-aggregator",
		Short:             "Aggregate Docs",
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return errors.New("missing product_listing_file argument")
			}

			filename := args[0]
			var err error
			if !filepath.IsAbs(filename) {
				filename, err = filepath.Abs(filename)
				if err != nil {
					return err
				}
			}
			fmt.Println("using product_listing_file=", filename)

			info, err := os.Stat(filename)
			if os.IsNotExist(err) {
				return fmt.Errorf("product_listing file not found, err:%v", err)
			}
			if info.IsDir() {
				return errors.New("product_listing file is actually a dir")
			}

			return process(filename)
		},
	}
	return cmd
}

func process(filename string) error {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}
	rootDir := filepath.Dir(filename)

	tmpDir, err := ioutil.TempDir("", "docs-aggregator")
	if err != nil {
		return err
	}
	defer func() {
		fmt.Println("removing tmp dir=", tmpDir)
		e2 := os.RemoveAll(tmpDir)
		if e2 != nil {
			fmt.Fprintf(os.Stderr, "failed to remove tmp dir, err : %v", err)
		}
	}()

	var cfg SharedWebsite
	err = json.Unmarshal(data, &cfg)
	if err != nil {
		return err
	}

	sh := shell.NewSession()
	sh.SetDir(rootDir)
	sh.ShowCMD = true

	for name, p := range cfg.Products {
		p.Name = name
		err = processProduct(p, rootDir, sh, filepath.Join(tmpDir, p.Name))
		if err != nil {
			return err
		}
		fmt.Println()
		fmt.Println("... ... ...")
		time.Sleep(5 * time.Second)
	}
	return nil
}

func processProduct(p Product, rootDir string, sh *shell.Session, tmpDir string) error {
	repoDir := filepath.Join(tmpDir, "repo")
	err := os.MkdirAll(repoDir, 0755)
	if err != nil {
		return err
	}

	err = sh.Command("git", "clone", p.GithubURL, repoDir).Run()
	if err != nil {
		return err
	}

	for _, v := range p.Versions {
		if !v.HostDocs {
			continue
		}
		if v.DocsDir == "" {
			v.DocsDir = "docs"
		}

		fmt.Println()
		sh.SetDir(repoDir)
		err = sh.Command("git", "checkout", v.Branch).Run()
		if err != nil {
			return err
		}

		vDir := filepath.Join(rootDir, "content", "products", p.Name, v.Branch)
		err = os.RemoveAll(vDir)
		if err != nil {
			return err
		}
		err = os.MkdirAll(filepath.Dir(vDir), 0755) // create parent dir
		if err != nil {
			return err
		}

		sh.SetDir(tmpDir)
		err = sh.Command("cp", "-r", filepath.Join("repo", v.DocsDir), vDir).Run()
		if err != nil {
			return err
		}

		fmt.Println(">>> ", vDir)

		err := filepath.Walk(vDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				fmt.Printf("prevent panic by handling failure accessing a path %q: %v\n", vDir, err)
				return err
			}
			if info.IsDir() || !strings.HasSuffix(path, ".md") {
				return nil // skip
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

			content := page.Content()

			if strings.Index(string(content), "/docs") > -1 {
				re1 := regexp.MustCompile(`(\(/docs)`)
				content = re1.ReplaceAll(content, []byte(`(/products/`+p.Name+`/`+v.Branch))

				re2 := regexp.MustCompile(`(\(/products/.*)(.md)(#.*)?\)`)
				for idx := 0; idx < 5; idx++ {
					content = re2.ReplaceAll(content, []byte(`${1}${3})`))
				}

				//if strings.Index(string(content), ".md") > -1 {
				//	fmt.Println(string(content))
				//	content = re2.ReplaceAll(content, []byte(`${1}${3})`))
				//}
				content = bytes.ReplaceAll(content, []byte(`"/docs/images`), []byte(`"/products/`+p.Name+`/`+v.Branch+`/images`))
			}

			var out string
			frontmatter := page.FrontMatter()

			if len(frontmatter) != 0 {
				out = "---\n"

				if rune(frontmatter[0]) == '-' {
					var m2 yaml.MapSlice
					err = yaml.Unmarshal(frontmatter, &m2)
					if err != nil {
						return err
					}
					for i := range m2 {
						if sk, ok := m2[i].Key.(string); ok && sk == "aliases" {

							v2, ok := m2[i].Value.([]interface{})
							if !ok {
								continue
							}
							strSlice := make([]string, 0, len(v2))
							for _, v := range v2 {
								if str, ok := v.(string); ok {
									// make aliases abs path
									if !strings.HasPrefix(str, "/") {
										str = "/" + str
									}

									strSlice = append(strSlice, str)
								} else {
									continue
								}
							}
							m2[i].Value = strSlice
						} else if vv, changed := stringifyMapKeys(m2[i].Value); changed {
							m2[i].Value = vv
						}
					}

					d2, err := yaml.Marshal(m2)
					if err != nil {
						return err
					}
					out += string(d2)
				} else {
					metadata, err := page.Metadata()
					if err != nil {
						return err
					}

					aliases, ok, err := unstructured.NestedStringSlice(metadata, "aliases")
					if err != nil {
						return err
					}
					if ok {
						for i := range aliases {
							if !strings.HasPrefix(aliases[i], "/") {
								aliases[i] = "/" + aliases[i]
							}
						}
						err = unstructured.SetNestedStringSlice(metadata, aliases, "aliases")
						if err != nil {
							return err
						}
					}

					metaYAML, err := yaml.Marshal(metadata)
					if err != nil {
						return err
					}
					out += string(metaYAML)
				}

				out = out + "---\n\n"
			}

			out = out + string(content)
			return ioutil.WriteFile(path, []byte(out), 0644)
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// stringifyMapKeys recurses into in and changes all instances of
// map[interface{}]interface{} to map[string]interface{}. This is useful to
// work around the impedence mismatch between JSON and YAML unmarshaling that's
// described here: https://github.com/go-yaml/yaml/issues/139
//
// Inspired by https://github.com/stripe/stripe-mock, MIT licensed
func stringifyMapKeys(in interface{}) (interface{}, bool) {
	switch in := in.(type) {
	case []interface{}:
		for i, v := range in {
			if vv, replaced := stringifyMapKeys(v); replaced {
				in[i] = vv
			}
		}
	case map[interface{}]interface{}:
		res := make(map[string]interface{})
		var (
			ok  bool
			err error
		)
		for k, v := range in {
			var ks string

			if ks, ok = k.(string); !ok {
				ks, err = cast.ToStringE(k)
				if err != nil {
					ks = fmt.Sprintf("%v", k)
				}
				// TODO(bep) added in Hugo 0.37, remove some time in the future.
				helpers.DistinctFeedbackLog.Printf("WARNING: YAML data/frontmatter with keys of type %T is since Hugo 0.37 converted to strings", k)
			}
			if vv, replaced := stringifyMapKeys(v); replaced {
				res[ks] = vv
			} else {
				res[ks] = v
			}
		}
		return res, true
	}

	return nil, false
}
