package cmd

import (
	"encoding/hex"
	"encoding/json"
	"github.com/AlecAivazis/survey/v2"
	"github.com/desertbit/grumble"
	"github.com/filecoin-project/go-address"
	lotusTypes "github.com/filecoin-project/lotus/chain/types"
	"io/ioutil"
	"strings"
)

func (cmd *command) initWallet () {
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
}

func (cmd *command) addWalletList(parent *grumble.Command)  {
	s := &grumble.Command{
		Name: "list",
		Help: "list wallet",
		Run: func(c *grumble.Context) error {
			ctx, cancel := c.App.Context()
			defer cancel()
			addrs, err := cmd.wallet.WalletList(ctx)
			if err != nil {
				return err
			}
			for _, addr := range addrs {
				c.App.Println(addr.String())
			}
			return nil
		},
	}
	parent.AddCommand(s)
}
func (cmd *command) addWalletNew(parent *grumble.Command)  {
	s := &grumble.Command{
		Name: "new",
		Help: "create new wallet",
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
			c.App.Println(addr.String())
			return nil
		},
	}
	parent.AddCommand(s)
}


func (cmd *command) addWalletExport(parent *grumble.Command)  {
	s := &grumble.Command{
		Name: "export",
		Help: "export keys",
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

			c.App.Println(hex.EncodeToString(b))
			return nil
		},
	}
	parent.AddCommand(s)
}


func (cmd *command) addWalletImport(parent *grumble.Command)  {
	s := &grumble.Command{
		Name: "import",
		Help: "import keys",
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
			}else {
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
			c.App.Println(addr.String())
			return nil
		},
	}
	parent.AddCommand(s)
}
