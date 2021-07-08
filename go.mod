module fost

go 1.16

require (
	github.com/AlecAivazis/survey/v2 v2.2.14
	github.com/common-nighthawk/go-figure v0.0.0-20210622060536-734e95fb86be
	github.com/desertbit/grumble v1.1.1
	github.com/fatih/color v1.12.0
	github.com/filecoin-project/go-address v0.0.5
	github.com/filecoin-project/go-jsonrpc v0.1.4-0.20210217175800-45ea43ac2bec
	github.com/filecoin-project/go-state-types v0.1.1-0.20210506134452-99b279731c48
	github.com/filecoin-project/lotus v1.10.1
	github.com/filecoin-project/specs-actors v0.9.13
	github.com/ipfs/go-log/v2 v2.2.0 // indirect
	github.com/multiformats/go-multihash v0.0.15 // indirect
	github.com/whyrusleeping/cbor-gen v0.0.0-20210422071115-ad5b82622e0f
	golang.org/x/crypto v0.0.0-20210616213533-5ff15b29337e // indirect
	golang.org/x/sys v0.0.0-20210630005230-0f9fa26af87c // indirect
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1
)

replace github.com/filecoin-project/filecoin-ffi => ./extern/filecoin-ffi
