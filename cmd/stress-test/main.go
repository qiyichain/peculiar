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
	ERC1155MintLimit          = uint64(100000) // gasLimit of each tx
	deployERC1155Limit        = uint64(6000000)
	ERC1155TransferTokenLimit = uint64(100000) // gasLimit of each tx, the first storage slot take much more gas

	// sig
	tokenTransferSig   = "a9059cbb"
	addDeveloperSig    = "22fbf1e8"
	ERC721CurrentIDSig = "01ec915a"
	ERC721MintSig      = "4c2f6dd3"
	ERC721TransferSig  = "9dd3045b"
	ERC1155MintSig     = "1f7fdffa" // function _mintBatch(address to, uint256[] memory ids, uint256[] memory amounts, bytes memory data)
	ERC1155TransferSig = "2eb2c2d6" // function safeBatchTransferFrom( address from, address to, uint256[] memory ids, uint256[] memory amounts, bytes memory data)

	leftSpace   = 31
	uriMetaData = byte(64)

	erc1155MintString        = "0000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000012000000000000000000000000000000000000000000000000000000000000001c0000000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000027740000000000000000000000000000000000000000000000000000000000002775000000000000000000000000000000000000000000000000000000000000277600000000000000000000000000000000000000000000000000000000000027770000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000003b9aca00000000000000000000000000000000000000000000000000000000e8d4a51000000000000000000000000000000000000000000000000000000000003b9aca0000000000000000000000000000000000000000000000000000038d7ea4c6800000000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000000"
	erc1155BatchTransferFrom = "00000000000000000000000000000000000000000000000000000000000000a0000000000000000000000000000000000000000000000000000000000000014000000000000000000000000000000000000000000000000000000000000001e000000000000000000000000000000000000000000000000000000000000000040000000000000000000000000000000000000000000000000000000000002774000000000000000000000000000000000000000000000000000000000000277500000000000000000000000000000000000000000000000000000000000027760000000000000000000000000000000000000000000000000000000000002777000000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000186a00000000000000000000000000000000000000000000000000000000005f5e10000000000000000000000000000000000000000000000000000000000000186a0000000000000000000000000000000000000000000000000000000174876e80000000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000000"
	
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
		commandStressTestERC1155TokenMint,
		commandStressTestERC1155TransferFrom,
		// TODO: commandstressTestERC1155Transfer,
		// TODO: commandStressTestERC1155TokenMint,
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
