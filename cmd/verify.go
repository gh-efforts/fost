package cmd

import (
	"encoding/hex"
	"github.com/desertbit/grumble"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/crypto"
	"github.com/filecoin-project/lotus/lib/sigs"
)

func (cmd *command) initVerify() {
	signCommand := &grumble.Command{
		Name:     "verify",
		Help:     "verify the signature of a message",
		LongHelp: "<signing address> <hexMessage> <signature>",
		Args: func(a *grumble.Args) {
			a.String("singer", "specify signing address")
			a.String("hexMessage", "specify hexMessage to verify")
			a.String("signature", "specify signature to verify")
		},
		Run: func(c *grumble.Context) (err error) {
			addr, err := address.NewFromString(c.Args.String("singer"))
			if err != nil {
				return err
			}

			msg, err := hex.DecodeString(c.Args.String("hexMessage"))
			if err != nil {
				return err
			}

			sigBytes, err := hex.DecodeString(c.Args.String("signature"))

			if err != nil {
				return err
			}

			var sig crypto.Signature
			if err := sig.UnmarshalBinary(sigBytes); err != nil {
				return err
			}

			err = sigs.Verify(&sig, addr, msg)
			if err != nil {
				return err
			}
			cmd.Info("valid")
			return nil
		},
	}
	cmd.app.AddCommand(signCommand)
}
