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
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
	"time"

	"github.com/appscode/static-assets/api"
	"github.com/appscode/static-assets/hugo"
	shell "github.com/codeskyblue/go-sh"
	"github.com/gohugoio/hugo/helpers"
	"github.com/gohugoio/hugo/parser"
	"github.com/imdario/mergo"
	"github.com/spf13/cast"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/sets"
)

type PageInfo struct {
	Version           string `json:"version"`
	SubProjectVersion string `json:"subproject_version,omitempty"`
	// Git GitInfo `json:"git"`
}

func (p PageInfo) Map(extra map[string]string) (map[string]interface{}, error) {
	data, err := json.Marshal(p)
	if err != nil {
		return nil, err
	}
	var m map[string]interface{}
	err = json.Unmarshal(data, &m)
	if err != nil {
		return nil, err
	}

	for k, v := range extra {
		if _, ok := m[k]; !ok {
			m[k] = v
		}
	}
	return m, nil
}

var sharedSite = false
var onlyAssets = false
var fmReplacements = map[string]string{}

func NewCmdDocsAggregator() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "docs-aggregator",
		Short:             "Aggregate Docs",
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			rootDir, err := os.Getwd()
			if err != nil {
				return err
			}
			return process(rootDir)
		},
	}
	cmd.Flags().StringVar(&product, "product", product, "Name of product")
	cmd.Flags().BoolVar(&sharedSite, "shared", sharedSite, "If true, considered a shared site like appscode.com instead of a product specific site like kubedb.com")
	cmd.Flags().BoolVar(&onlyAssets, "only-assets", onlyAssets, "If true, only aggregates config")
	cmd.Flags().StringToStringVar(&fmReplacements, "fm-replacements", fmReplacements, "Frontmatter replacements")
	return cmd
}

func process(rootDir string) error {
	err := processHugoConfig(rootDir)
	if err != nil {
		return err
	}

	cfg, err := processDataConfig(rootDir)
	if err != nil {
		return err
	}

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

	sh := shell.NewSession()
	sh.ShowCMD = true

	if !sharedSite {
		sharedSite = len(cfg.Products) > 1
	}

	err = processAssets(cfg.Assets, rootDir, sh, tmpDir)
	if err != nil {
		return err
	}

	if onlyAssets {
		return nil // exit
	}

	for _, name := range cfg.Products {
		if product != "" && product != name {
			continue
		}

		pfile := filepath.Join(rootDir, "data", "products", name+".json")
		fmt.Println("using product_listing_file=", pfile)

		var p api.Product
		data, err := ioutil.ReadFile(pfile)
		if err != nil {
			return err
		}
		err = json.Unmarshal(data, &p)
		if err != nil {
			return err
		}

		if p.Key == "" {
			return fmt.Errorf("missing product key in file=%s", pfile)
		}

		err = processProduct(p, rootDir, sh, tmpDir)
		if err != nil {
			return err
		}
		fmt.Println()
		fmt.Println("... ... ...")
		time.Sleep(5 * time.Second)
	}
	return nil
}

func processHugoConfig(rootDir string) error {
	if err := processHugoConfigEnv(rootDir, "dev"); err != nil {
		log.Println("failed to process params.dev.json")
		log.Println(err)
	}
	return processHugoConfigEnv(rootDir, "")
}

func processHugoConfigEnv(rootDir, env string) error {
	pf := "params.json"
	if env != "" {
		pf = "params." + env + ".json"
	}
	baseData, err := hugo.Asset(pf)
	if err != nil {
		return err
	}

	var baseParams map[string]string
	err = json.Unmarshal(baseData, &baseParams)
	if err != nil {
		return err
	}

	cf := "config.yaml"
	if env != "" {
		cf = "config." + env + ".yaml"
	}
	filename := filepath.Join(rootDir, cf)
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}

	var m2 yaml.MapSlice
	err = yaml.Unmarshal(data, &m2)
	if err != nil {
		return err
	}
	for i := range m2 {
		if sk, ok := m2[i].Key.(string); ok && sk == "params" {
			p2, _ := m2[i].Value.(yaml.MapSlice)
			for j := range p2 {
				key := p2[j].Key.(string)
				if v, found := baseParams[key]; found {
					p2[j].Value = v
					delete(baseParams, key)
				}
			}
			for k, v := range baseParams {
				p2 = append(p2, yaml.MapItem{
					Key:   k,
					Value: v,
				})
			}
			m2[i].Value = p2
		}
	}

	data, err = yaml.Marshal(m2)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(filename, data, 0644)
}

func processDataConfig(rootDir string) (*api.Listing, error) {
	filename := filepath.Join(rootDir, "data", "config.json")
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("product_listing file not found, err:%v", err)
	} else if err != nil {
		return nil, err
	}
	if info.IsDir() {
		return nil, errors.New("product_listing file is actually a dir")
	}

	baseData, err := hugo.Asset("config.json")
	if err != nil {
		return nil, err
	}
	var baseCfg map[string]interface{}
	err = json.Unmarshal(baseData, &baseCfg)
	if err != nil {
		return nil, err
	}

	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	var cfg map[string]interface{}
	err = json.Unmarshal(data, &cfg)
	if err != nil {
		return nil, err
	}
	for k, v := range baseCfg {
		if k != "assets" || !hasKey(cfg, k) {
			// inject assets if not found, all other keys are always injected
			cfg[k] = v
		}
	}

	data3, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return nil, err
	}
	err = ioutil.WriteFile(filename, data3, 0644)
	if err != nil {
		return nil, err
	}

	var out api.Listing
	err = json.Unmarshal(data3, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func hasKey(m map[string]interface{}, key string) bool {
	_, ok := m[key]
	return ok
}

func processAssets(a api.AssetListing, rootDir string, sh *shell.Session, tmpDir string) error {
	tmpDir = filepath.Join(tmpDir, "assets")
	repoDir := filepath.Join(tmpDir, "repo")
	err := os.MkdirAll(repoDir, 0755)
	if err != nil {
		return err
	}

	err = sh.Command("git", "clone", a.RepoURL, repoDir).Run()
	if err != nil {
		return err
	}

	fmt.Println()
	sh.SetDir(repoDir)
	err = sh.Command("git", "checkout", a.Version).Run()
	if err != nil {
		return err
	}

	for src, dst := range a.Dirs {
		err = sh.Command("cp", "-r", src, filepath.Dir(filepath.Join(rootDir, dst))).Run()
		if err != nil {
			return err
		}
		if src == "data" {
			err = sh.Command("find", filepath.Join(rootDir, dst), "-name", "bindata.go").Command("xargs", "rm", "-rf", "{}").Run()
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func processProduct(p api.Product, rootDir string, sh *shell.Session, tmpDir string) error {
	tmpDir = filepath.Join(tmpDir, p.Key)
	repoDir := filepath.Join(tmpDir, "repo")
	err := os.MkdirAll(repoDir, 0755)
	if err != nil {
		return err
	}

	err = sh.Command("git", "clone", p.RepoURL, repoDir).Run()
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
		ref := v.Branch
		if ref == "" {
			ref = v.Version
		}
		err = sh.Command("git", "checkout", ref).Run()
		if err != nil {
			return err
		}

		var vDir string
		if sharedSite {
			vDir = filepath.Join(rootDir, "content", "products", p.Key, v.Version)
		} else {
			vDir = filepath.Join(rootDir, "content", "docs", v.Version)
		}
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

		// process sub project
		err = processSubProject(p, v, rootDir, vDir, sh, tmpDir)
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

			if strings.Contains(string(content), "/docs") {
				prefix := `/products/` + p.Key + `/` + v.Version
				if !sharedSite {
					prefix = `/docs/` + v.Version
				}

				re1 := regexp.MustCompile(`(\(/docs)`)
				content = re1.ReplaceAll(content, []byte(`(`+prefix))

				var re2 *regexp.Regexp
				if sharedSite {
					re2 = regexp.MustCompile(`(\(/products/.*)(.md)(#.*)?\)`)
				} else {
					re2 = regexp.MustCompile(`(\(/docs/.*)(.md)(#.*)?\)`)
				}
				for idx := 0; idx < 5; idx++ {
					content = re2.ReplaceAll(content, []byte(`${1}${3})`))
				}

				//if strings.Index(string(content), ".md") > -1 {
				//	fmt.Println(string(content))
				//	content = re2.ReplaceAll(content, []byte(`${1}${3})`))
				//}
				content = bytes.ReplaceAll(content, []byte(`"/docs/images`), []byte(`"`+prefix+`/images`))
			}

			pageInfo, err := PageInfo{Version: v.Version}.Map(v.Info)
			if err != nil {
				return err
			}

			t, err := template.New("x2").Parse(string(page.FrontMatter()))
			if err != nil {
				return fmt.Errorf("failed to process frontmatter template for file %q. reason: %v", path, err)
			}
			var buf2 bytes.Buffer
			err = t.Execute(&buf2, pageInfo)
			if err != nil {
				return fmt.Errorf("failed to process frontmatter template for file %q. reason: %v", path, err)
			}

			if buf2.Len() > 0 {
				if rune(buf2.Bytes()[0]) == '-' {
					var m2 yaml.MapSlice
					err = yaml.Unmarshal(buf2.Bytes(), &m2)
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

					// inject Page params info.***
					var infoFound bool
					for i := range m2 {
						if sk, ok := m2[i].Key.(string); ok && sk == "info" {
							d3, err := yaml.Marshal(m2[i].Value)
							if err != nil {
								return err
							}
							m3 := make(map[string]interface{})
							err = yaml.Unmarshal(d3, &m3)
							if err != nil {
								return err
							}

							// merge needs a map as dst
							err = mergo.Merge(&m3, pageInfo)
							if err != nil {
								return err
							}
							m2[i].Value = m3
							infoFound = true
						}
					}
					if !infoFound {
						m2 = append(m2, yaml.MapItem{Key: "info", Value: pageInfo})
					}

					d2, err := yaml.Marshal(m2)
					if err != nil {
						return err
					}
					d2 = applyFrontmatterReplacements(d2)
					buf2.Reset()
					_, err = buf2.WriteString("---\n")
					if err != nil {
						return err
					}
					_, err = buf2.Write(d2)
					if err != nil {
						return err
					}
					_, err = buf2.WriteString("---\n\n")
					if err != nil {
						return err
					}
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

					// inject Page params info.***
					existingInfo, ok, err := unstructured.NestedFieldNoCopy(metadata, "info")
					if err != nil {
						return err
					}
					if ok {
						err = mergo.Merge(&existingInfo, pageInfo)
						if err != nil {
							return err
						}
						err = unstructured.SetNestedField(metadata, existingInfo, "info")
						if err != nil {
							return err
						}
					} else {
						err = unstructured.SetNestedField(metadata, pageInfo, "info")
						if err != nil {
							return err
						}
					}

					metaYAML, err := yaml.Marshal(metadata)
					if err != nil {
						return err
					}
					metaYAML = applyFrontmatterReplacements(metaYAML)
					buf2.Reset()
					_, err = buf2.WriteString("---\n")
					if err != nil {
						return err
					}
					_, err = buf2.Write(metaYAML)
					if err != nil {
						return err
					}
					_, err = buf2.WriteString("---\n\n")
					if err != nil {
						return err
					}
				}
			}

			_, err = buf2.Write(content)
			if err != nil {
				return err
			}
			return ioutil.WriteFile(path, buf2.Bytes(), 0644)
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func applyFrontmatterReplacements(data []byte) []byte {
	s := string(data)
	for k, v := range fmReplacements {
		s = strings.ReplaceAll(s, k, v)
	}
	return []byte(s)
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

func processSubProject(p api.Product, v api.ProductVersion, rootDir, vDir string, sh *shell.Session, rootTempDir string) error {
	for spKey, info := range p.SubProjects {
		// create project version specific subfolder for the subprojects
		tmpDir := filepath.Join(rootTempDir, p.Key+"-"+v.Version, spKey)
		repoDir := filepath.Join(tmpDir, "repo")

		pfile := filepath.Join(rootDir, "data", "products", spKey+".json")
		fmt.Println("using product_listing_file=", pfile)

		var sp api.Product
		data, err := ioutil.ReadFile(pfile)
		if err != nil {
			return err
		}
		err = json.Unmarshal(data, &sp)
		if err != nil {
			return err
		}

		err = os.MkdirAll(tmpDir, 0755)
		if err != nil {
			return err
		}
		if !exists(repoDir) {
			err = sh.Command("git", "clone", sp.RepoURL, repoDir).Run()
			if err != nil {
				return err
			}
		}

		for _, mapping := range info.Mappings {
			if sets.NewString(mapping.Versions...).Has(v.Version) {

				for _, spVersion := range mapping.SubProjectVersions {
					spv, err := findVersion(sp.Versions, spVersion)
					if err != nil {
						return err
					}

					if !spv.HostDocs {
						continue
					}
					if spv.DocsDir == "" {
						spv.DocsDir = "docs"
					}

					fmt.Println()
					sh.SetDir(repoDir)
					ref := spv.Branch
					if ref == "" {
						ref = spv.Version
					}
					err = sh.Command("git", "checkout", ref).Run()
					if err != nil {
						return err
					}

					spvDir := filepath.Join(vDir, info.Dir, spv.Version)
					err = os.RemoveAll(spvDir)
					if err != nil {
						return err
					}
					err = os.MkdirAll(filepath.Dir(spvDir), 0755) // create parent dir
					if err != nil {
						return err
					}

					sh.SetDir(tmpDir)
					err = sh.Command("cp", "-r", filepath.Join("repo", spv.DocsDir), spvDir).Run()
					if err != nil {
						return err
					}
					fmt.Println(">>> ", spvDir)

					err = filepath.Walk(spvDir, func(path string, info os.FileInfo, err error) error {
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

						pageInfo, err := PageInfo{
							Version:           v.Version,
							SubProjectVersion: spv.Version,
						}.Map(v.Info)
						if err != nil {
							return err
						}

						t := template.Must(template.New("x2").Parse(string(page.FrontMatter())))
						var buf2 bytes.Buffer
						err = t.Execute(&buf2, pageInfo)
						if err != nil {
							return err
						}

						// inject Page params info.***
						// https://gohugo.io/variables/page/#page-level-params
						if rune(buf2.Bytes()[0]) == '-' {
							var m2 yaml.MapSlice
							err = yaml.Unmarshal(buf2.Bytes(), &m2)
							if err != nil {
								return err
							}
							var infoFound bool
							for i := range m2 {
								if sk, ok := m2[i].Key.(string); ok && sk == "info" {
									m2[i].Value = pageInfo
									infoFound = true
								} else if vv, changed := stringifyMapKeys(m2[i].Value); changed {
									m2[i].Value = vv
								}
							}
							if !infoFound {
								m2 = append(m2, yaml.MapItem{Key: "info", Value: pageInfo})
							}

							d2, err := yaml.Marshal(m2)
							if err != nil {
								return err
							}
							buf2.Reset()
							_, err = buf2.WriteString("---\n")
							if err != nil {
								return err
							}
							_, err = buf2.Write(d2)
							if err != nil {
								return err
							}
							_, err = buf2.WriteString("---\n\n")
							if err != nil {
								return err
							}
						} else {
							metadata, err := page.Metadata()
							if err != nil {
								return err
							}

							err = unstructured.SetNestedField(metadata, pageInfo, "info")
							if err != nil {
								return err
							}

							metaYAML, err := yaml.Marshal(metadata)
							if err != nil {
								return err
							}
							buf2.Reset()
							_, err = buf2.WriteString("---\n")
							if err != nil {
								return err
							}
							_, err = buf2.Write(metaYAML)
							if err != nil {
								return err
							}
							_, err = buf2.WriteString("---\n\n")
							if err != nil {
								return err
							}
						}

						_, err = buf2.Write(page.Content())
						if err != nil {
							return err
						}
						return ioutil.WriteFile(path, buf2.Bytes(), 0644)
					})
					if err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}

func findVersion(versions []api.ProductVersion, x string) (api.ProductVersion, error) {
	for _, v := range versions {
		if v.Version == x {
			return v, nil
		}
	}
	return api.ProductVersion{}, fmt.Errorf("version %s not found", x)
}

// exists reports whether the named file or directory Exists.
func exists(name string) bool {
	if _, err := os.Stat(name); os.IsNotExist(err) {
		return false
	}
	return true
}
