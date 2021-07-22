module github.com/appscodelabs/hugo-tools

go 1.16

require (
	github.com/appscode/static-assets v0.6.7
	github.com/codeskyblue/go-sh v0.0.0-20200712050446-30169cf553fe
	github.com/gohugoio/hugo v0.79.1
	github.com/imdario/mergo v0.3.5
	github.com/spf13/cast v1.3.1
	github.com/spf13/cobra v1.1.1
	github.com/spf13/pflag v1.0.5
	gomodules.xyz/runtime v0.0.0-20201104200926-d838b09dda8b
	gomodules.xyz/x v0.0.0-20201105065653-91c568df6331
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/apimachinery v0.18.3
	sigs.k8s.io/yaml v1.2.0
)

replace github.com/imdario/mergo => github.com/imdario/mergo v0.3.5
