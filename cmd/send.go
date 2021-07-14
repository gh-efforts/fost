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
	"github.com/ipfs/go-cid"
	"golang.org/x/xerrors"
	"strings"
)

func (cmd *command) initSend() {
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
			} else {
				addrs, err := cmd.wallet.WalletList(ctx)
				if err != nil {
					return err
				}
				if len(addrs) == 0 {
					return fmt.Errorf("wallet is empty")
				} else if len(addrs) == 1 {
					params.From = addrs[0]
				} else {
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

			if c.Flags.String("params-json") != "" {
				if cmd.apiGetter != nil {
					decParams, err := cmd.DecodeTypedParamsFromJSON(ctx, params.To, params.Method, c.Flags.String("params-json"))
					if err != nil {
						return fmt.Errorf("failed to decode json params: %w", err)
					}
					params.Params = decParams
				} else {
					c.App.Println("Params: No chain node connection, can't decode params")
				}
			}
			if c.Flags.String("params-hex") != "" {
				decParams, err := hex.DecodeString(c.Flags.String("params-hex"))
				if err != nil {
					return fmt.Errorf("failed to decode hex params: %w", err)
				}
				params.Params = decParams
			}
			msg, err := cmd.buildMsg(ctx, params)
			if err != nil {
				return err
			}
			cmd.Info("message details:")

			v, err := json.MarshalIndent(msg, "", "    ")
			if err != nil {
				return err
			}
			cmd.Info(string(v))
			var confirm bool
			prompt := &survey.Confirm{
				Message: "Confirm to send ?:",
			}
			if err := survey.AskOne(prompt, &confirm, nil); err != nil {
				return err
			}

			if confirm {
				id, err := cmd.sendMsg(ctx, msg)
				if err != nil {
					return err
				}
				if id != cid.Undef {
					cmd.Info(id.String())
				}
			}
			return nil
		},
	}
	cmd.app.AddCommand(walletCommand)
}

type SendParams struct {
	To    address.Address
	From  address.Address
	Value abi.TokenAmount

	GasPremium *abi.TokenAmount
	GasFeeCap  *abi.TokenAmount
	GasLimit   *int64

	Nonce  *uint64
	Method abi.MethodNum
	Params []byte
}

func (cmd *command) buildMsg(ctx context.Context, params SendParams) (*lotusTypes.Message, error) {
	msg := &lotusTypes.Message{
		From:   params.From,
		To:     params.To,
		Value:  params.Value,
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

	if !cmd.IsOffline() {
		oApi, closer, err := cmd.apiGetter()
		if err != nil {
			return nil, err
		}
		defer closer()

		msg, err = oApi.GasEstimateMessageGas(ctx, msg, nil, lotusTypes.EmptyTSK)
		if err != nil {
			return nil, xerrors.Errorf("GasEstimateMessageGas error: %w", err)
		}
	} else {
		if msg.GasFeeCap.IsZero() || msg.GasPremium.IsZero() || msg.GasLimit == 0 {
			return nil, xerrors.Errorf("in offline mode, you must manually set gas-premium, gas-feecap, gas-limit, nonce !!!")
		}
	}

	return msg, nil
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

func (cmd *command) sendMsg(ctx context.Context, msg *lotusTypes.Message) (cid.Cid, error) {
	if cmd.IsOffline() {
		if msg.Nonce == 0 {
			return cid.Cid{}, fmt.Errorf("in offline mode, the nonce must be specified")
		}

		sm, err := cmd.signMsg(ctx, msg)
		if err != nil {
			return cid.Cid{}, err
		}

		smg, err := sm.MarshalJSON()
		if err != nil {
			return cid.Cid{}, fmt.Errorf("error serializing message: %w", err)
		}

		cmd.Warning("In offline mode, you must manually send to the network: ")
		curlData := `
=================================================================================================
curl -X POST \
-H "Content-Type: application/json" \
--data '{ "jsonrpc": "2.0", "method": "Filecoin.MpoolPush", "params": [$paramsData], "id": 1 }' \
'$rpcAddr'
=================================================================================================
`
		curlData = strings.Replace(curlData, "$paramsData", string(smg), 1)
		curlData = strings.Replace(curlData, "$rpcAddr", cmd.config.Rpc, 1)
		cmd.Info(curlData)
		return cid.Cid{}, nil

	} else {
		oApi, closer, err := cmd.apiGetter()
		if err != nil {
			return cid.Cid{}, err
		}
		defer closer()
		msg.Nonce, err = oApi.MpoolGetNonce(ctx, msg.From)
		if err != nil {
			return cid.Cid{}, err
		}

		sm, err := cmd.signMsg(ctx, msg)
		if err != nil {
			return cid.Cid{}, err
		}
		return oApi.MpoolPush(ctx, sm)
	}
}
