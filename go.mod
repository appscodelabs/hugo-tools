module github.com/appscodelabs/hugo-tools

go 1.14

require (
	github.com/appscode/go v0.0.0-20200323182826-54e98e09185a
	github.com/appscode/static-assets v0.5.1
	github.com/codeskyblue/go-sh v0.0.0-20190412065543-76bd3d59ff27
	github.com/gohugoio/hugo v0.49.1
	github.com/imdario/mergo v0.3.5
	github.com/spf13/cast v1.3.1
	github.com/spf13/cobra v1.0.0
	github.com/spf13/pflag v1.0.5
	gopkg.in/yaml.v2 v2.3.0
	k8s.io/apimachinery v0.18.3
)

replace (
	github.com/codeskyblue/go-sh => github.com/gomodules/go-sh v0.0.0-20200616225555-bfeba62378c9
	github.com/imdario/mergo => github.com/imdario/mergo v0.3.5
)
