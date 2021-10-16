module github.com/appscodelabs/hugo-tools

go 1.16

require (
	github.com/appscode/static-assets v0.7.1
	github.com/gohugoio/hugo v0.49.1
	github.com/imdario/mergo v0.3.5
	github.com/spf13/cast v1.3.1
	github.com/spf13/cobra v1.1.3
	github.com/spf13/pflag v1.0.5
	gomodules.xyz/go-sh v0.1.0
	gomodules.xyz/logs v0.0.5
	gomodules.xyz/runtime v0.2.0
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/apimachinery v0.18.3
	k8s.io/klog/v2 v2.8.0
	sigs.k8s.io/yaml v1.2.0
)

replace github.com/imdario/mergo => github.com/imdario/mergo v0.3.5
