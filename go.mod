module code.vegaprotocol.io/check_validator_setup

go 1.19

require (
	code.vegaprotocol.io/vega v0.71.3
	github.com/fatih/color v1.15.0
	github.com/jedib0t/go-pretty/v6 v6.4.6
	github.com/schollz/progressbar/v3 v3.13.1
	google.golang.org/grpc v1.52.0
)

require (
	github.com/ethereum/go-ethereum v1.11.2 // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.9.0 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.17 // indirect
	github.com/mattn/go-runewidth v0.0.14 // indirect
	github.com/mitchellh/colorstring v0.0.0-20190213212951-d06e56a500db // indirect
	github.com/rivo/uniseg v0.2.0 // indirect
	golang.org/x/crypto v0.5.0 // indirect
	golang.org/x/net v0.7.0 // indirect
	golang.org/x/sys v0.6.0 // indirect
	golang.org/x/term v0.6.0 // indirect
	golang.org/x/text v0.7.0 // indirect
	google.golang.org/genproto v0.0.0-20221118155620-16455021b5e6 // indirect
	google.golang.org/protobuf v1.28.2-0.20220831092852-f930b1dc76e8 // indirect
)

replace (
	github.com/btcsuite/btcd => github.com/btcsuite/btcd v0.23.3
	github.com/fergusstrange/embedded-postgres => github.com/vegaprotocol/embedded-postgres v1.13.1-0.20221123183204-2e7a2feee5bb
	github.com/shopspring/decimal => github.com/vegaprotocol/decimal v1.3.1-uint256
	github.com/tendermint/tendermint => github.com/vegaprotocol/cometbft v0.34.28-0.20230322133204-3d8588de736e
	github.com/tendermint/tm-db => github.com/cometbft/cometbft-db v0.6.7
)
