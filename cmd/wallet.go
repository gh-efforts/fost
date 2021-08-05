package cmd

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/AlecAivazis/survey/v2"
	"github.com/desertbit/grumble"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-jsonrpc"
	"github.com/filecoin-project/lotus/api/v0api"
	lotusTypes "github.com/filecoin-project/lotus/chain/types"
	"github.com/olekukonko/tablewriter"
	"io/ioutil"
	"strings"
)

func (cmd *command) initWallet() {
	walletCommand := &grumble.Command{
		Name:     "wallet",
		Help:     "wallet tools",
		LongHelp: "wallet administration tools",
	}
	cmd.app.AddCommand(walletCommand)
	cmd.addWalletList(walletCommand)
	cmd.addWalletNew(walletCommand)
	cmd.addWalletExport(walletCommand)
	cmd.addWalletImport(walletCommand)
	cmd.addWalletRemove(walletCommand)
}

func (cmd *command) addWalletList(parent *grumble.Command) {
	s := &grumble.Command{
		Name: "list",
		Help: "list keys",
		Run: func(c *grumble.Context) error {
			ctx, cancel := c.App.Context()
			defer cancel()
			addrs, err := cmd.wallet.WalletList(ctx)
			if err != nil {
				return err
			}

			var node v0api.FullNode
			var closer jsonrpc.ClientCloser
			if !cmd.IsOffline() {
				node, closer, err = cmd.apiGetter()
				if err != nil {
					return err
				}
				defer closer()
			}

			table := tablewriter.NewWriter(c.App.Stdout())
			table.SetHeader([]string{"Address", "Balance"})
			for _, addr := range addrs {
				var balance string
				if node != nil {
					act, err := node.StateGetActor(ctx, addr, lotusTypes.EmptyTSK)
					if err == nil {
						balance = lotusTypes.FIL(act.Balance).String()
					}
				}
				table.Append([]string{addr.String(), balance})
			}
			table.Render()
			return nil
		},
	}
	parent.AddCommand(s)
}

func (cmd *command) addWalletNew(parent *grumble.Command) {
	s := &grumble.Command{
		Name: "new",
		Help: "create new key",
		Run: func(c *grumble.Context) error {
			ctx, cancel := c.App.Context()
			defer cancel()

			keyType := ""
			prompt := &survey.Select{
				Message: "Choose a type:",
				Options: []string{string(lotusTypes.KTBLS), string(lotusTypes.KTSecp256k1)},
				Default: string(lotusTypes.KTBLS),
			}

			if err := survey.AskOne(prompt, &keyType, nil); err != nil {
				return err
			}

			addr, err := cmd.wallet.WalletNew(ctx, lotusTypes.KeyType(keyType))
			if err != nil {
				return err
			}
			cmd.Info(addr.String())
			cmd.Warning(fmt.Sprintf("Before exiting %s, back up (export) your wallet, otherwise ALL DATA will be lost!!!", c.App.Config().Name))
			return nil
		},
	}
	parent.AddCommand(s)
}

func (cmd *command) addWalletExport(parent *grumble.Command) {
	s := &grumble.Command{
		Name: "export",
		Help: "export keys",
		Flags: func(f *grumble.Flags) {
			f.String("p", "path", "", "export private key to file path")
		},
		Run: func(c *grumble.Context) error {
			ctx, cancel := c.App.Context()
			defer cancel()

			key := ""

			addrs, err := cmd.wallet.WalletList(ctx)
			if err != nil {
				return err
			}

			prompt := &survey.Select{
				Message: "Choose a key:",
			}

			for _, addr := range addrs {
				prompt.Options = append(prompt.Options, addr.String())
			}

			if err := survey.AskOne(prompt, &key, nil); err != nil {
				return err
			}

			addr, _ := address.NewFromString(key)

			keyInfo, err := cmd.wallet.WalletExport(ctx, addr)
			if err != nil {
				return err
			}

			b, err := json.Marshal(keyInfo)
			if err != nil {
				return err
			}

			if c.Flags.String("path") != "" {
				err := ioutil.WriteFile(c.Flags.String("path"), []byte(hex.EncodeToString(b)), 0600)
				if err != nil {
					return fmt.Errorf("write private key failed: %s", err)
				}
				c.App.Printf("write private key to: %s\n", c.Flags.String("path"))
			} else {
				cmd.Info(hex.EncodeToString(b))
			}
			cmd.Warning("Keep your private key safe, the private key is everything!!!")
			return nil
		},
	}
	parent.AddCommand(s)
}

func (cmd *command) addWalletImport(parent *grumble.Command) {
	s := &grumble.Command{
		Name:     "import",
		Help:     "import keys",
		LongHelp: "[<path> (optional, will read from stdin if omitted)]",
		Flags: func(f *grumble.Flags) {
			f.String("p", "path", "", "private key file path")
		},
		Run: func(c *grumble.Context) error {
			ctx, cancel := c.App.Context()
			defer cancel()

			input := ""

			if c.Flags.String("path") == "" {

				prompt := &survey.Password{
					Message: "Please type private key",
				}
				if err := survey.AskOne(prompt, &input); err != nil {
					return err
				}
			} else {
				fd, err := ioutil.ReadFile(c.Flags.String("path"))
				if err != nil {
					return err
				}
				input = string(fd)
			}

			var ki lotusTypes.KeyInfo
			data, err := hex.DecodeString(strings.TrimSpace(input))
			if err != nil {
				return err
			}

			if err := json.Unmarshal(data, &ki); err != nil {
				return err
			}

			addr, err := cmd.wallet.WalletImport(ctx, &ki)
			if err != nil {
				return err
			}
			cmd.Info(fmt.Sprintf("imported key %s successfully!", addr.String()))
			return nil
		},
	}
	parent.AddCommand(s)
}

func (cmd *command) addWalletRemove(parent *grumble.Command) {
	s := &grumble.Command{
		Name: "remove",
		Help: "remove keys",
		Run: func(c *grumble.Context) error {
			ctx, cancel := c.App.Context()
			defer cancel()
			addrs, err := cmd.wallet.WalletList(ctx)
			if err != nil {
				return err
			}

			key := ""
			prompt := &survey.Select{
				Message: "Which key do you want to remove? :",
			}
			for _, addr := range addrs {
				prompt.Options = append(prompt.Options, addr.String())
			}

			if err := survey.AskOne(prompt, &key, nil); err != nil {
				return err
			}
			addr, _ := address.NewFromString(key)

			var confirm bool
			promptConfirm := &survey.Confirm{
				Message: "Confirm to remove ?:",
			}
			if err := survey.AskOne(promptConfirm, &confirm, nil); err != nil {
				return err
			}

			if confirm {
				return cmd.wallet.WalletDelete(ctx, addr)
			}
			return nil
		},
	}
	parent.AddCommand(s)
}
