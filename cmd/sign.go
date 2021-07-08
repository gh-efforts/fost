package cmd

import (
	"encoding/hex"
	"fmt"
	"github.com/AlecAivazis/survey/v2"
	"github.com/desertbit/grumble"
	"github.com/filecoin-project/go-address"
	lotusApi "github.com/filecoin-project/lotus/api"
)

func (cmd *command) initSign() {
	signCommand := &grumble.Command{
		Name:     "sign",
		Help:     "sign a message",
		LongHelp: "[hexMessage]",
		Args: func(a *grumble.Args) {
			a.String("hexMessage", "specify hexMessage to sign")
		},
		Flags: func(f *grumble.Flags) {
			f.String("", "signer", "", "optionally specify the account to sign")
		},
		Run: func(c *grumble.Context) (err error) {
			ctx, cancel := c.App.Context()
			defer cancel()
			hexData := c.Args.String("hexMessage")

			signData, err := hex.DecodeString(hexData)
			if err != nil {
				return err
			}

			signer := c.Flags.String("signer")

			if signer == "" {
				addrs, err := cmd.wallet.WalletList(ctx)
				if err != nil {
					return err
				}
				if len(addrs) == 0 {
					return ErrWalletEmpty
				} else if len(addrs) == 1 {
					signer = addrs[0].String()
				} else {
					prompt := &survey.Select{
						Message: "Sign with ?:",
					}
					for _, addr := range addrs {
						prompt.Options = append(prompt.Options, addr.String())
					}

					if err := survey.AskOne(prompt, &signer, nil); err != nil {
						return err
					}
				}
				addr, err := address.NewFromString(signer)
				if err != nil {
					return err
				}
				cmd.Info("sign: ")
				cmd.Info(hexData)
				c.App.Printf("with %s ?\n", addr.String())
				var confirm bool
				prompt := &survey.Confirm{
					Message: "confirm ?:",
				}
				if err := survey.AskOne(prompt, &confirm, nil); err != nil {
					return err
				}
				if confirm {
					sig, err := cmd.wallet.WalletSign(ctx, addr, signData, lotusApi.MsgMeta{
						Type: lotusApi.MTUnknown,
					})
					if err != nil {
						return fmt.Errorf("sign err: %s", err)
					}
					sigBytes := append([]byte{byte(sig.Type)}, sig.Data...)
					cmd.Info(hex.EncodeToString(sigBytes))
				}
			}
			return nil
		},
	}
	cmd.app.AddCommand(signCommand)
}
