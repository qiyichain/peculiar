package main

import (
	"fmt"
	"os"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/fdlimit"
	"github.com/ethereum/go-ethereum/internal/flags"
	"github.com/ethereum/go-ethereum/log"
	"gopkg.in/urfave/cli.v1"
)

var (
	// Git SHA1 commit hash of the release (set via linker flags)
	gitCommit = ""
	gitDate   = ""
)

// const test params
var (
	receiver                = common.HexToAddress("0xf513e4e5Ded9B510780D016c482fC158209DE9AA")
	AddressListContractAddr = common.HexToAddress("0x000000000000000000000000000000000000F003")

	// gasLimit
	ethTransferLimit          = uint64(21000)
	tokenTransferLimit        = uint64(100000)
	addDeveloperLimit         = uint64(100000)
	deployERC721Limit         = uint64(4000000)
	ERC721MintLimit           = uint64(150000)
	ERC721TransferTokenLimit  = uint64(100000)
	deployERC1155Limit        = uint64(8000000) // todo: need to change
	ERC1155MintLimit          = uint64(150000)  // todo: need to change
	ERC1155TransferTokenLimit = uint64(100000)  // todo: need to change

	// sig
	tokenTransferSig   = "a9059cbb"
	addDeveloperSig    = "22fbf1e8"
	ERC721CurrentIDSig = "01ec915a"
	ERC721MintSig      = "4c2f6dd3"
	ERC721TransferSig  = "9dd3045b"

	leftSpace   = 31
	uriMetaData = byte(64)

	defaultDecimal = 18

	jobsPerThread = 16

	storePath = "/data/stress-test/keys"
)

var app *cli.App

func init() {
	app = flags.NewApp(gitCommit, gitDate, "ethereum checkpoint helper tool")
	app.Commands = []cli.Command{
		commandStressTestTransfer,
		commandstressTestERC721Transfer,
		commandStressTestERC721TokenMint,
		commandDeployERC721,
		commandDeployERC1155,
		// TODO: commandstressTestERC1155Transfer,
		// TODO: commandStressTestERC1155TokenMint,
		// TODO: commandDeployERC1155,
	}
	app.Flags = []cli.Flag{
		nodeURLFlag,
		privKeyFlag,
	}
	cli.CommandHelpTemplate = flags.OriginCommandHelpTemplate
}

// Commonly used command line flags.
var (
	nodeURLFlag = cli.StringFlag{
		Name:  "rpc",
		Value: "http://172.16.100.101:8545, http://172.16.100.102:8545, http://172.16.100.103:8545, http://172.16.100.104:8545",
		Usage: "The rpc endpoint list of servial local or remote geth nodes(separator ',')",
	}
	privKeyFlag = cli.StringFlag{
		Name: "privkey",
		// 0xf513e4e5Ded9B510780D016c482fC158209DE9AA
		Value: "5ea30eea9ba9500f3601f7659f0ccace819c562456e2f745fb2555918ab32277",
		Usage: "The main account used for test",
	}
	accountNumberFlag = cli.IntFlag{
		Name:  "accountNumber",
		Value: 100,
		Usage: "The number of accounts used for test",
	}
	totalTxsFlag = cli.IntFlag{
		Name:  "totalTxs",
		Value: 10000,
		Usage: "The total number of transactions sent for test",
	}
	threadsFlag = cli.IntFlag{
		Name:  "threads",
		Value: 100,
		Usage: "The go routine number for test (note: threads should be lower than totalTxs)",
	}
	decimalFlag = cli.IntFlag{
		Name:  "decimal",
		Value: defaultDecimal,
		Usage: "The decimal of token",
	}
	addDeveloperFlag = cli.BoolFlag{
		Name:  "addDeveloper",
		Usage: "Determine whether add accounts to white list or not",
	}
	deployFlag = cli.IntFlag{
		Name:  "deploy",
		Value: 20,
		Usage: "The number of deploying contract",
	}
	pathFlag = cli.StringFlag{
		Name:  "path",
		Value: "/data/stress-test/721Tokens",
		Usage: "The absolute path of token addresses file",
	}
	rrModeFlag = cli.BoolFlag{
		Name:  "rr",
		Usage: "Determine whether run on Round-Robin mode or not",
	}
	loopFlag = cli.BoolFlag{
		Name:  "loop",
		Usage: "Determine whether send transactions in loop",
	}
	txGenPeriodFlag = cli.IntFlag{
		Name:  "period",
		Value: 1,
		Usage: "The period of generating transactions",
	}
)

func main() {
	log.Root().SetHandler(log.LvlFilterHandler(log.LvlInfo, log.StreamHandler(os.Stderr, log.TerminalFormat(true))))
	fdlimit.Raise(10000)

	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
