package cmd

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"github.com/AlecAivazis/survey/v2"
	"github.com/desertbit/grumble"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	lotusTypes "github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/specs-actors/actors/builtin"
	"github.com/olekukonko/tablewriter"
	"os"
	"strings"
)

func (cmd *command) initSendMulti() {
	sendMultiCommand := &grumble.Command{
		Name: "send-multi",
		Help: "Send funds between multiple accounts (Only available in online mode)",
		LongHelp: `
[targetData] example:
	address1,value1
	address2,value2
	address3,value3
`,
		Args: func(a *grumble.Args) {
			a.String("path", "specify targetData file path to send")
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

			if cmd.IsOffline() {
				return fmt.Errorf("%s only available in online mode", c.Command.Name)
			}

			ctx, cancel := c.App.Context()
			defer cancel()

			am, err := readSendMultiFile(c.Args.String("path"))
			if err != nil {
				return err
			}
			if len(am) == 0 {
				return fmt.Errorf("targetData file is empty")
			}

			var params SendParams

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

			var buildMsg []*lotusTypes.Message
			table := tablewriter.NewWriter(c.App.Stdout())
			table.SetHeader([]string{"To", "Value", "Max Fees", "Max Total Cost"})

			for _, sv := range am {
				params := params
				params.To = sv.to
				params.Value = sv.value
				msg, err := cmd.buildMsg(ctx, params)
				if err != nil {
					return err
				}
				buildMsg = append(buildMsg, msg)
				table.Append([]string{
					msg.To.String(),
					lotusTypes.FIL(msg.Value).String(),
					lotusTypes.FIL(msg.RequiredFunds()).String(),
					lotusTypes.FIL(big.Add(msg.RequiredFunds(), msg.Value)).String()})
			}
			table.Render()

			var confirm bool
			prompt := &survey.Confirm{
				Message: "Confirm to send ?:",
				Help:    "They will be sent to the network!",
			}
			if err := survey.AskOne(prompt, &confirm, nil); err != nil {
				return err
			}

			if confirm {
				table = tablewriter.NewWriter(c.App.Stdout())
				table.SetHeader([]string{"To", "Value", "Max Fees", "Max Total Cost", "TX ID"})
				nonce, err := cmd.getNonce(ctx, params.From)
				if err != nil {
					return err
				}
				for idx, bm := range buildMsg {
					bm.Nonce = nonce + uint64(idx)
					id, err := cmd.sendMsg(ctx, bm)
					if err != nil {
						log.Errorf("send to %s: %s", bm.To.String(), err)
						table.Append([]string{
							bm.To.String(),
							lotusTypes.FIL(bm.Value).String(),
							lotusTypes.FIL(bm.RequiredFunds()).String(),
							lotusTypes.FIL(big.Add(bm.RequiredFunds(), bm.Value)).String()})
						continue
					}
					table.Append([]string{
						bm.To.String(),
						lotusTypes.FIL(bm.Value).String(),
						lotusTypes.FIL(bm.RequiredFunds()).String(),
						lotusTypes.FIL(big.Add(bm.RequiredFunds(), bm.Value)).String(),
						id.String(),
					})
				}
				table.Render()
			}
			return nil
		},
	}
	cmd.app.AddCommand(sendMultiCommand)
}

type sendValue struct {
	to    address.Address
	value abi.TokenAmount
}

func readSendMultiFile(path string) ([]sendValue, error) {
	file, err := os.Open(path)

	if err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)

	var lines []string

	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := file.Close(); err != nil {
		return nil, err
	}

	var resp []sendValue

	for _, line := range lines {
		sp := strings.Split(line, ",")
		if len(sp) != 2 {
			return nil, fmt.Errorf("error format: %s", line)
		}

		targetAddress, err := address.NewFromString(sp[0])
		if err != nil {
			return nil, fmt.Errorf("failed to parse target address %s: %w", sp[0], err)
		}

		val, err := lotusTypes.ParseFIL(sp[1])
		if err != nil {
			return nil, fmt.Errorf("failed to parse amount: %w", err)
		}

		resp = append(resp, sendValue{
			to:    targetAddress,
			value: abi.TokenAmount(val),
		})
	}
	return resp, nil
}
