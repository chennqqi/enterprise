package plugin

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/micro/cli"
	"github.com/micro/go-log"
	"github.com/micro/micro/plugin"
)

func buildSo(soPath string, parts []string) error {
	// check if .so file exists
	if _, err := os.Stat(soPath); os.IsExist(err) {
		return nil
	}

	// name and things
	name := parts[len(parts)-1]
	// type of plugin
	typ := parts[0]
	// new func signature
	newfn := fmt.Sprintf("New%s", strings.Title(typ))

	// micro has NewPlugin type def
	if typ == "micro" {
		newfn = "NewPlugin"
	}

	// now build the plugin
	if err := Build(soPath, &Plugin{
		Name:    name,
		Type:    typ,
		Path:    filepath.Join(append([]string{"github.com/micro/go-plugins"}, parts...)...),
		NewFunc: newfn,
	}); err != nil {
		return fmt.Errorf("Failed to build plugin %s: %v", name, err)
	}

	return nil
}

func load(p string) error {
	p = strings.TrimSpace(p)

	if len(p) == 0 {
		return nil
	}

	parts := strings.Split(p, "/")

	// 1 part means local plugin
	// plugin/foobar
	if len(parts) == 1 {
		return fmt.Errorf("Unknown plugin %s", p)
	}

	// set soPath to specified path
	soPath := p

	// build on the fly if not .so
	if !strings.HasSuffix(p, ".so") {
		// set new so path
		soPath = filepath.Join("plugin", p+".so")

		// build new .so
		if err := buildSo(soPath, parts); err != nil {
			return err
		}
	}

	// load the plugin
	pl, err := Load(soPath)
	if err != nil {
		return fmt.Errorf("Failed to load plugin %s: %v", soPath, err)
	}

	// Initialise the plugin
	return Init(pl)
}

// returns a micro plugin which loads plugins
func NewPlugin() plugin.Plugin {
	return plugin.NewPlugin(
		plugin.WithName("plugins"),
		plugin.WithFlag(
			cli.StringSliceFlag{
				Name:   "plugins",
				EnvVar: "MICRO_PLUGINS",
				Usage:  "Comma separated list of plugins e.g broker/rabbitmq, registry/etcd, micro/basic_auth, /path/to/plugin.so",
			},
		),
		plugin.WithInit(func(ctx *cli.Context) error {
			plugins := ctx.StringSlice("plugins")
			if len(plugins) == 0 {
				return nil
			}

			for _, p := range plugins {
				if err := load(p); err != nil {
					return err
				}
				log.Logf("Loaded plugin %s\n", p)
			}

			return nil
		}),
	)
}
