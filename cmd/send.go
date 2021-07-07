package cmd

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/AlecAivazis/survey/v2"
	"github.com/desertbit/grumble"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/api"
	lotusApi "github.com/filecoin-project/lotus/api"
	lotusTypes "github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/specs-actors/actors/builtin"
	"golang.org/x/xerrors"
)

func (cmd *command) initSend () {
	walletCommand := &grumble.Command{
		Name:     "send",
		Help:     "Send funds between accounts",
		LongHelp: "[targetAddress] [amount]",
		Args: func(a *grumble.Args) {
			a.String("targetAddress", "specify targetAddress to send")
			a.String("amount", "specify amount to send in FIL")
		},
		Flags: func(f *grumble.Flags) {
			f.String("", "from", "", "optionally specify the account to send funds from")
			f.String("", "gas-premium", "0", "specify gas price to use in AttoFIL")
			f.String("", "gas-feecap", "0", "specify gas fee cap to use in AttoFIL")
			f.Int64("", "gas-limit", 0, "specify gas limit")
			f.Uint64("", "nonce", 0, "specify the nonce to use")
			f.Uint64("", "method", uint64(builtin.MethodSend), "specify method to invoke")
			f.String("", "params-json", "", "specify invocation parameters in json")
			f.String("", "params-hex", "", "specify invocation parameters in hex")
		},
		Run: func(c *grumble.Context) (err error) {
			ctx, cancel := c.App.Context()
			defer cancel()

			var params SendParams
			params.To, err = address.NewFromString(c.Args.String("targetAddress"))
			if err != nil {
				return fmt.Errorf("failed to parse target address: %w", err)
			}

			val, err := lotusTypes.ParseFIL(c.Args.String("amount"))
			if err != nil {
				return fmt.Errorf("failed to parse amount: %w", err)
			}
			params.Value = abi.TokenAmount(val)

			if from := c.Flags.String("from"); from != "" {
				addr, err := address.NewFromString(from)
				if err != nil {
					return err
				}
				params.From = addr
			}else {
				addrs, err := cmd.wallet.WalletList(ctx)
				if err != nil {
					return err
				}
				if len(addrs) == 0 {
					return fmt.Errorf("wallet is empty")
				}else if len(addrs) == 1 {
					params.From = addrs[0]
				}else {
					key := ""
					prompt := &survey.Select{
						Message: "Send from?:",
					}
					for _, addr := range addrs {
						prompt.Options = append(prompt.Options, addr.String())
					}

					if err := survey.AskOne(prompt, &key, nil); err != nil {
						return err
					}
					params.From, _ = address.NewFromString(key)
				}
			}

			if c.Flags.String("gas-premium") != "" {
				gp, err := lotusTypes.BigFromString(c.Flags.String("gas-premium"))
				if err != nil {
					return err
				}
				params.GasPremium = &gp
			}

			if c.Flags.String("gas-feecap") != "" {
				gfc, err := lotusTypes.BigFromString(c.Flags.String("gas-feecap"))
				if err != nil {
					return err
				}
				params.GasFeeCap = &gfc
			}

			if c.Flags.Int64("gas-limit") != 0 {
				limit := c.Flags.Int64("gas-limit")
				params.GasLimit = &limit
			}

			params.Method = abi.MethodNum(c.Flags.Uint64("method"))

			if c.Flags.Uint64("nonce") != 0 {
				n := c.Flags.Uint64("nonce")
				params.Nonce = &n
			}

			msg := buildMsg(params)

			c.App.Println("sign message:")

			v, err := json.MarshalIndent(msg, "", "    ")
			if err != nil {
				return err
			}
			c.App.Println(string(v))
			var confirm bool
			prompt := &survey.Confirm{
				Message: "confirm ?:",
			}
			if err := survey.AskOne(prompt, &confirm, nil); err != nil {
				return err
			}

			if confirm {
				sm, err := cmd.signMsg(ctx, msg)
				if err != nil {
					return err
				}
				smg, err := sm.Serialize()
				if err != nil {
					return fmt.Errorf("error serializing message: %w", err)
				}
				c.App.Println(hex.EncodeToString(smg))
			}
			return nil
		},
	}
	cmd.app.AddCommand(walletCommand)
}

type SendParams struct {
	To   address.Address
	From address.Address
	Value  abi.TokenAmount

	GasPremium *abi.TokenAmount
	GasFeeCap  *abi.TokenAmount
	GasLimit   *int64

	Nonce  *uint64
	Method abi.MethodNum
	Params []byte
}

func buildMsg(params SendParams) *lotusTypes.Message {
	msg := lotusTypes.Message{
		From:  params.From,
		To:    params.To,
		Value: params.Value,
		Method: params.Method,
		Params: params.Params,
	}
	if params.GasPremium != nil {
		msg.GasPremium = *params.GasPremium
	} else {
		msg.GasPremium = lotusTypes.NewInt(0)
	}
	if params.GasFeeCap != nil {
		msg.GasFeeCap = *params.GasFeeCap
	} else {
		msg.GasFeeCap = lotusTypes.NewInt(0)
	}
	if params.GasLimit != nil {
		msg.GasLimit = *params.GasLimit
	} else {
		msg.GasLimit = 0
	}

	if params.Nonce != nil {
		msg.Nonce = *params.Nonce
	}
	return &msg
}

func (cmd *command) signMsg(ctx context.Context, msg *lotusTypes.Message) (*lotusTypes.SignedMessage, error) {
	mb, err := msg.ToStorageBlock()
	if err != nil {
		return nil, xerrors.Errorf("serializing message: %w", err)
	}

	sig, err := cmd.wallet.WalletSign(ctx, msg.From, mb.Cid().Bytes(), lotusApi.MsgMeta{
		Type:  api.MTChainMsg,
		Extra: mb.RawData(),
	})
	if err != nil {
		return nil, xerrors.Errorf("failed to sign message: %w", err)
	}

	return &lotusTypes.SignedMessage{
		Message:   *msg,
		Signature: *sig,
	}, nil
}
