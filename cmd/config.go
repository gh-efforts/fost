package cmd

import (
	"encoding/json"
	"fmt"
	"github.com/desertbit/grumble"
	"strconv"
)

type Config struct {
	Offline bool   `json:"offline"`
	Rpc     string `json:"rpc"`
	Token   string `json:"token"`
}

func (cmd *command) initConfig() {
	configCommand := &grumble.Command{
		Name: "config",
		Help: "manager config",
	}
	cmd.app.AddCommand(configCommand)

	cmd.addConfigList(configCommand)
	cmd.addConfigSet(configCommand)
}

func (cmd *command) addConfigList(parent *grumble.Command) {
	s := &grumble.Command{
		Name: "show",
		Help: "show config",
		Run: func(c *grumble.Context) error {
			v, err := json.MarshalIndent(cmd.config, "", "    ")
			if err != nil {
				return err
			}
			cmd.Info(string(v))
			return nil
		},
	}
	parent.AddCommand(s)
}

func (cmd *command) addConfigSet(parent *grumble.Command) {
	s := &grumble.Command{
		Name: "set",
		Help: "set config value",
		Args: func(a *grumble.Args) {
			a.String("field", "config field")
			a.String("value", "config value")
		},
		Run: func(c *grumble.Context) error {
			field := c.Args.String("field")
			value := c.Args.String("value")
			switch field {
			case "offline":
				v, err := strconv.ParseBool(value)
				if err != nil {
					return err
				}
				ctx, cancel := c.App.Context()
				defer cancel()
				cmd.SetOffline(ctx, v)
			case "rpc":
				cmd.config.Rpc = value
			case "token":
				cmd.config.Token = value
			default:
				cmd.Warning(fmt.Sprintf("unrecognized field: %s", field))
			}
			return nil
		},
	}
	parent.AddCommand(s)
}
