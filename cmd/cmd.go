package cmd

import (
	"fmt"
	"github.com/common-nighthawk/go-figure"
	"github.com/desertbit/grumble"
	"github.com/fatih/color"
	lotusApi "github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/chain/wallet"
)

type command struct {
	app *grumble.App
	wallet lotusApi.Wallet
}

type option func(*command)


func newCommand(opts ...option) (c *command, err error) {
	wa, err := wallet.NewWallet(wallet.NewMemKeyStore())
	if err != nil {
		return nil, fmt.Errorf("new wallet: %s", err)
	}

	c = &command{
		app: grumble.New(&grumble.Config{
			Name:                  "fost",
			Description:           "Filecoin offline signature tool.",
			Prompt:                "fost » ",
			PromptColor:           color.New(color.FgGreen, color.Bold),
			HelpHeadlineColor:     color.New(color.FgGreen),
			HelpHeadlineUnderline: true,
			HelpSubCommands:       true,

			Flags: func(f *grumble.Flags) {
				f.String("d", "directory", "DEFAULT", "set an alternative root directory path")
				f.Bool("v", "verbose", false, "enable verbose mode")
			},
		}),
		wallet: wa,
	}

	for _, o := range opts {
		o(c)
	}

	c.initLogo()
	c.initWallet()
	c.initSend()

	return c, nil
}

func (cmd *command) initLogo()  {
		cmd.app.SetPrintASCIILogo(func(a *grumble.App) {
			myFigure := figure.NewFigure("FOST", "", true)
			myFigure.Print()
		})
}


func (cmd *command) Execute() error {
	grumble.Main(cmd.app)
	return nil
}


func Execute() (err error) {
	c, err := newCommand()
	if err != nil {
		return err
	}
	return c.Execute()
}
