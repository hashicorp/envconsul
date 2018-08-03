package command

import (
	"fmt"
	"strings"

	"github.com/hashicorp/vault/api"
	"github.com/hashicorp/vault/meta"
)

// AuthEnableCommand is a Command that enables a new endpoint.
type AuthEnableCommand struct {
	meta.Meta
}

func (c *AuthEnableCommand) Run(args []string) int {
	var description, path, pluginName string
	var local bool
	flags := c.Meta.FlagSet("auth-enable", meta.FlagSetDefault)
	flags.StringVar(&description, "description", "", "")
	flags.StringVar(&path, "path", "", "")
	flags.StringVar(&pluginName, "plugin-name", "", "")
	flags.BoolVar(&local, "local", false, "")
	flags.Usage = func() { c.Ui.Error(c.Help()) }
	if err := flags.Parse(args); err != nil {
		return 1
	}

	args = flags.Args()
	if len(args) != 1 {
		flags.Usage()
		c.Ui.Error(fmt.Sprintf(
			"\nauth-enable expects one argument: the type to enable."))
		return 1
	}

	authType := args[0]

	// If no path is specified, we default the path to the backend type
	// or use the plugin name if it's a plugin backend
	if path == "" {
		if authType == "plugin" {
			path = pluginName
		} else {
			path = authType
		}
	}

	client, err := c.Client()
	if err != nil {
		c.Ui.Error(fmt.Sprintf(
			"Error initializing client: %s", err))
		return 2
	}

	if err := client.Sys().EnableAuthWithOptions(path, &api.EnableAuthOptions{
		Type:        authType,
		Description: description,
		PluginName:  pluginName,
		Local:       local,
	}); err != nil {
		c.Ui.Error(fmt.Sprintf(
			"Error: %s", err))
		return 2
	}

	c.Ui.Output(fmt.Sprintf(
		"Successfully enabled '%s' at '%s'!",
		authType, path))

	return 0
}

func (c *AuthEnableCommand) Synopsis() string {
	return "Enable a new auth provider"
}

func (c *AuthEnableCommand) Help() string {
	helpText := `
Usage: vault auth-enable [options] type

  Enable a new auth provider.

  This command enables a new auth provider. An auth provider is responsible
  for authenticating a user and assigning them policies with which they can
  access Vault.

General Options:
` + meta.GeneralOptionsUsage() + `
Auth Enable Options:

  -description=<desc>     Human-friendly description of the purpose of the
                          auth provider. This shows up in the auth -methods command.

  -path=<path>            Mount point for the auth provider. This defaults
                          to the type of the mount. This will make the auth
                          provider available at "/auth/<path>"

  -plugin-name            Name of the auth plugin to use based from the name 
                          in the plugin catalog.

  -local                  Mark the mount as a local mount. Local mounts
                          are not replicated nor (if a secondary)
                          removed by replication.
`
	return strings.TrimSpace(helpText)
}
