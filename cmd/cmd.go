package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"fost/util"
	"github.com/common-nighthawk/go-figure"
	"github.com/desertbit/grumble"
	"github.com/fatih/color"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-jsonrpc"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/api"
	lotusApi "github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/chain/consensus/filcns"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/lotus/chain/wallet"
	"github.com/filecoin-project/lotus/node/repo"
	logging "github.com/ipfs/go-log/v2"
	cbg "github.com/whyrusleeping/cbor-gen"
	"reflect"
)

var (
	ErrWalletEmpty = fmt.Errorf("wallet is empty")
	log            = logging.Logger("cmd")
)

type command struct {
	app       *grumble.App
	wallet    lotusApi.Wallet
	config    *Config
	apiGetter func() (api.FullNode, jsonrpc.ClientCloser, error)
}

func newCommand() (c *command, err error) {
	c = &command{
		app: grumble.New(&grumble.Config{
			Name:                  "fost",
			Description:           "Filecoin simple command line wallet!",
			Prompt:                "fost » ",
			PromptColor:           color.New(color.FgGreen, color.Bold),
			HelpHeadlineColor:     color.New(color.FgGreen),
			HelpHeadlineUnderline: true,
			Flags:                 ConfigFlags(),
		}),
		config: &Config{},
	}
	c.app.OnInit(func(a *grumble.App, flags grumble.FlagMap) error {

		c.config.Offline = flags.Bool("offline")
		c.config.Rpc = flags.String("rpc")
		c.config.Token = flags.String("token")
		c.config.Repo = flags.String("repo")

		if c.config.Repo == "" {
			c.wallet, err = wallet.NewWallet(wallet.NewMemKeyStore())
			if err != nil {
				return fmt.Errorf("new wallet: %s", err)
			}
		} else {
			r, err := repo.NewFS(c.config.Repo)
			if err != nil {
				return err
			}

			ok, err := r.Exists()
			if err != nil {
				return err
			}
			if !ok {
				if err := r.Init(repo.Worker); err != nil {
					return err
				}
			}

			lr, err := r.Lock(repo.Wallet)
			if err != nil {
				return err
			}

			ks, err := lr.KeyStore()
			if err != nil {
				return err
			}

			c.wallet, err = wallet.NewWallet(ks)
		}

		ctx, _ := a.Context()
		c.SetOffline(ctx, c.config.Offline)
		return nil
	})

	c.initLogo()
	c.initWallet()
	c.initSend()
	c.initSign()
	c.initVerify()
	c.initConfig()
	c.initSendMulti()

	return c, nil
}

func (cmd *command) initLogo() {
	cmd.app.SetPrintASCIILogo(func(a *grumble.App) {
		f1 := figure.NewFigure("FOST", "", true)
		f1.Print()
	})
}

func (cmd *command) Warning(msg string) {
	c := color.New(color.FgHiRed)
	c.Println("")
	c.Println(msg)
}

func (cmd *command) Info(msg string) {
	c := color.New(color.FgHiWhite)
	c.Println("")
	c.Println(msg)
}

func (cmd *command) Println(msg string) {
	fmt.Println(msg)
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

func (cmd *command) DecodeTypedParamsFromJSON(ctx context.Context, to address.Address, method abi.MethodNum, paramstr string) ([]byte, error) {
	api, closer, err := cmd.apiGetter()
	if err != nil {
		return nil, err
	}
	defer closer()
	act, err := api.StateGetActor(ctx, to, types.EmptyTSK)
	if err != nil {
		return nil, err
	}

	methodMeta, found := filcns.NewActorRegistry().Methods[act.Code][method]
	if !found {
		return nil, fmt.Errorf("method %d not found on actor %s", method, act.Code)
	}

	p := reflect.New(methodMeta.Params.Elem()).Interface().(cbg.CBORMarshaler)

	if err := json.Unmarshal([]byte(paramstr), p); err != nil {
		return nil, fmt.Errorf("unmarshaling input into params type: %w", err)
	}

	buf := new(bytes.Buffer)
	if err := p.MarshalCBOR(buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (cmd *command) SetOffline(ctx context.Context, o bool) {
	cmd.config.Offline = o
	if !o {
		cmd.apiGetter = func() (api.FullNode, jsonrpc.ClientCloser, error) {
			return util.GetFullNodeAPIUsingCredentials(ctx, cmd.config.Rpc, cmd.config.Token)
		}
	} else {
		cmd.apiGetter = nil
	}
}

func (cmd *command) IsOffline() bool {
	if cmd.apiGetter == nil {
		return true
	} else if cmd.config.Offline {
		return true
	} else {
		return false
	}
}

func ConfigFlags() func(f *grumble.Flags) {
	v := func(f *grumble.Flags) {
		f.Bool("o", "offline", false, "don't query chain state in interactive mode!")
		f.String("r", "rpc", "https://api.node.glif.io/rpc/v0", "lotus rpc url")
		f.String("t", "token", "", "rpc token")
		f.String("d", "repo", "", "wallet repo path (If not specified, use memory storage)")
	}
	return v
}
