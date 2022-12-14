package main

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/congress/systemcontract"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"gopkg.in/urfave/cli.v1"
)

var commandStressTestTransfer = cli.Command{
	Name:  "testTransfer",
	Usage: "Send eth transfer transactions for stress test",
	Flags: []cli.Flag{
		nodeURLFlag,
		privKeyFlag,
		accountNumberFlag,
		totalTxsFlag,
		threadsFlag,
		addDeveloperFlag,
		rrModeFlag,
		loopFlag,
		txGenPeriodFlag,
	},
	Action: utils.MigrateFlags(stressTestTransfer),
}

var commandstressTestERC721Transfer = cli.Command{
	Name:  "testERC721Transfer",
	Usage: "Send ERC721 token transfer transactions for stress test",
	Flags: []cli.Flag{
		nodeURLFlag,
		privKeyFlag,
		accountNumberFlag,
		totalTxsFlag,
		threadsFlag,
		addDeveloperFlag,
		decimalFlag,
		pathFlag,
		loopFlag,
		txGenPeriodFlag,
		// mintFlag,
	},
	Action: utils.MigrateFlags(stressTestERC721Transfer),
}

var commandStressTestERC721TokenMint = cli.Command{
	Name:  "testERC721Mint",
	Usage: "Mint ERC721 token for stress test",
	Flags: []cli.Flag{
		nodeURLFlag,
		privKeyFlag,
		accountNumberFlag,
		totalTxsFlag,
		threadsFlag,
		pathFlag,
		loopFlag,
		txGenPeriodFlag,
	},
	Action: utils.MigrateFlags(stressTestERC721Mint),
}

var commandDeployERC721 = cli.Command{
	Name:  "deploy721",
	Usage: "Deploy ERC721 contract for test",
	Flags: []cli.Flag{
		nodeURLFlag,
		privKeyFlag,
		addDeveloperFlag,
		pathFlag,
		deployFlag,
	},
	Action: utils.MigrateFlags(deployERC721Contracts),
}

var commandDeployERC1155 = cli.Command{
	Name:  "deploy1155",
	Usage: "Deploy ERC1155 contract for test",
	Flags: []cli.Flag{
		nodeURLFlag,
		privKeyFlag,
		addDeveloperFlag,
		pathFlag,
		deployFlag,
	},
	Action: utils.MigrateFlags(deployERC1155Contracts),
}

var commandStressTestERC1155TokenMint = cli.Command{
	Name:  "testERC1155Mint",
	Usage: "Mint ERC1155 contract for stress test",
	Flags: []cli.Flag{
		nodeURLFlag,
		privKeyFlag,
		accountNumberFlag,
		totalTxsFlag,
		threadsFlag,
		pathFlag,
		loopFlag,
		txGenPeriodFlag,
	},
	Action: utils.MigrateFlags(stressTestERC1155BatchMint),
}

var commandStressTestERC1155TransferFrom = cli.Command{
	Name:  "testERC1155Transfer",
	Usage: "Transfer ERC1155 tokenFrom",
	Flags: []cli.Flag{
		nodeURLFlag,
		privKeyFlag,
		accountNumberFlag,
		totalTxsFlag,
		threadsFlag,
		addDeveloperFlag,
		rrModeFlag,
		pathFlag,
		loopFlag,
		txGenPeriodFlag,
	},
	Action: utils.MigrateFlags(stressTestERC1155BatchTransferFrom),
}

func initEthClients(ctx *cli.Context) ([]*ethclient.Client, error) {
	clients := newClients(getRPCList(ctx))
	if len(clients) == 0 {
		return nil, errors.New("no rpc url set")
	}
	return clients, nil
}

func initAccounts(accountNumber int, chainID *big.Int) ([]*bind.TransactOpts, error) {
	if accountNumber <= 0 {
		return nil, fmt.Errorf("invalid account number: %v", accountNumber)
	}

	var (
		accounts []*bind.TransactOpts
		first    bool
	)

	keys, err := loadAccounts(getStorePath())
	if err != nil {
		first = true
		log.Warn("load accounts failed", "err", err)
		return nil, err
	}

	toGen := accountNumber - len(keys)
	if len(keys) > 0 {
		accounts = append(accounts, newAccounts(keys, chainID)...)
	}

	if toGen > 0 {
		genKeys, genAccounts, err := generateRandomAccounts(toGen, chainID)
		if err != nil {
			log.Error("generate accounts failed", "err", err)
			return nil, err
		}
		log.Info("generate accounts over", "generated", len(genAccounts))

		accounts = append(accounts, genAccounts...)
		if first {
			if err := writeAccounts(getStorePath(), genKeys); err != nil {
				return nil, err
			}
		} else {
			if err := appendAccounts(getStorePath(), genKeys); err != nil {
				return nil, err
			}
		}
	}

	return accounts[:accountNumber], nil
}

func initTransfer(sender *bind.TransactOpts, receivers []*bind.TransactOpts, client *ethclient.Client) error {
	amount := big.NewInt(params.Ether)
	amount.Mul(amount, big.NewInt(1e+9))

	log.Info("start initTransfer: sending eth to test account")
	// send eth for normal eth transfer test or pay gas fees
	err := sendEtherToRandomAccount(sender, receivers, amount, client)
	log.Info("end initTransfer")
	return err
}

func initERC721Tokens(tokens []common.Address, accounts []*bind.TransactOpts, client *ethclient.Client) (map[common.Address]uint64, error) {
	log.Info("start initERC721Tokens: sending token to test account")
	tokenID, err := mintTokenRR(tokens, accounts, client)
	log.Info("end initERC721Tokens")

	return tokenID, err
}

func stressTestTransfer(ctx *cli.Context) error {
	clients, err := initEthClients(ctx)
	if err != nil {
		return err
	}

	var (
		client        = clients[0]
		chainID, _    = client.ChainID(context.Background())
		adminAccount  = newAccount(ctx.GlobalString(privKeyFlag.Name), chainID)
		accountNumber = ctx.Int(accountNumberFlag.Name)
		total         = ctx.Int(totalTxsFlag.Name)
		threads       = ctx.Int(threadsFlag.Name)
		addDeveloper  = ctx.Bool(addDeveloperFlag.Name)
		rr            = ctx.Bool(rrModeFlag.Name)
		loop          = ctx.Bool(loopFlag.Name)
		intv          = ctx.Int(txGenPeriodFlag.Name)
		done          chan struct{}
		fail          = make(chan error, 1)
		again         = true
	)

	if threads > total {
		return errors.New("threads amount should lower than total tx amount")
	}

	if total < accountNumber || total%accountNumber != 0 {
		return errors.New("total tx amount should be a multiple of account amount")
	}

	accounts, err := initAccounts(accountNumber, chainID)
	if err != nil {
		return err
	}

	err = initTransfer(adminAccount, accounts, client)
	if err != nil {
		return err
	}

	if addDeveloper {
		err = addDeveloperWhiteList(adminAccount, accounts, systemcontract.AddressListContractAddr, client)
		if err != nil {
			return err
		}
	}

	// generate signed transactions
	amount := big.NewInt(params.Ether)
	amount.Div(amount, big.NewInt(1e+6))
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	timer := time.NewTicker(time.Second * time.Duration(intv))
	defer timer.Stop()

	if !loop {
		go func() {
			sigs <- syscall.SIGTERM
		}()
	}

LOOP:
	for {
		if done == nil && again {
			done = make(chan struct{})

			var (
				txs []*types.Transaction
				err error
			)

			if rr {
				txs, err = generateSignedTransferTransactionsRR(total, accounts, amount, client)
			} else {
				txs, err = generateSignedTransferTransactions(total, accounts, amount, client)
			}

			if err != nil {
				log.Error("generate signed transfer txs falied", "err", err)
				fail <- err
				continue
			}
			log.Info("generate transfer txs over", "total", len(txs))

			currentBlock, _ := client.BlockByNumber(context.Background(), nil)
			log.Info("current block", "number", currentBlock.Number())

			// send txs
			start := time.Now()
			if rr {
				err = stressSendTransactionsRR(txs, threads, clients)
			} else {
				err = stressSendTransactions(txs, threads, clients)
			}

			if err != nil {
				log.Error("send transfer txs falied", "err", err)
				fail <- err
				continue
			}
			log.Info("send transfer txs over", "cost(milliseconds)", time.Since(start))

			close(done)
		}

		select {
		case <-sigs:
			log.Info("capture interupt, shutting down...")
			break LOOP

		case <-timer.C:
			if !again {
				again = true
			}

		case <-done:
			done = nil
			again = false

		case err := <-fail:
			log.Error("capture error while doing task,  shutting down...", "err", err)
			break LOOP
		}

	}

	return nil
}

func stressTestERC721Transfer(ctx *cli.Context) error {
	path := ctx.String(pathFlag.Name)
	tokens, err := loadContractAddrs(path)
	if err != nil {
		return err
	}

	if len(tokens) == 0 {
		return errors.New("no ERC721 token address")
	}

	decimal := ctx.Int(decimalFlag.Name)
	if decimal > 18 || decimal <= 0 {
		return fmt.Errorf("unsupported decimal %d", decimal)
	}

	clients, err := initEthClients(ctx)
	if err != nil {
		return err
	}

	var (
		client        = clients[0]
		chainID, _    = client.ChainID(context.Background())
		adminAccount  = newAccount(ctx.GlobalString(privKeyFlag.Name), chainID)
		accountNumber = ctx.Int(accountNumberFlag.Name)
		total         = ctx.Int(totalTxsFlag.Name)
		threads       = ctx.Int(threadsFlag.Name)
		addDeveloper  = ctx.Bool(addDeveloperFlag.Name)
		loop          = ctx.Bool(loopFlag.Name)
		intv          = ctx.Int(txGenPeriodFlag.Name)
		done          chan struct{}
		fail          = make(chan error, 1)
		again         = true
	)

	if threads > total {
		return errors.New("threads amount should lower than total tx amount")
	}

	if total < accountNumber || total%accountNumber != 0 {
		return errors.New("total tx amount should be a multiple of account amount")
	}

	accounts, err := initAccounts(accountNumber, chainID)
	if err != nil {
		return err
	}

	err = initTransfer(adminAccount, accounts, client)
	if err != nil {
		return err
	}

	if addDeveloper {
		err = addDeveloperWhiteList(adminAccount, accounts, systemcontract.AddressListContractAddr, client)
		if err != nil {
			return err
		}
	}

	startIDs, err := initERC721Tokens(tokens, accounts, client)
	if err != nil {
		return err
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	timer := time.NewTicker(time.Second * time.Duration(intv))
	defer timer.Stop()

	if !loop {
		go func() {
			sigs <- syscall.SIGTERM
		}()
	}

LOOP:
	for {
		if done == nil && again {
			done = make(chan struct{})

			txs, err := generateSignedERC721TransferTransactions(total, tokens, accounts, startIDs, client)

			if err != nil {
				log.Error("generate signed ERC721 token transfer txs falied", "err", err)
				fail <- err
				continue
			}
			log.Info("generate ERC721 token transfer txs over", "total", len(txs))

			currentBlock, _ := client.BlockByNumber(context.Background(), nil)
			log.Info("current block", "number", currentBlock.Number())

			// send txs
			start := time.Now()
			err = stressSendTransactions(txs, threads, clients)

			if err != nil {
				log.Error("send ERC721 token transfer txs falied", "err", err)
				fail <- err
				continue
			}
			log.Info("send ERC721 token transfer txs over", "cost(milliseconds)", time.Since(start))

			close(done)
		}

		select {
		case <-sigs:
			log.Info("capture interupt, shutting down...")
			break LOOP

		case <-timer.C:
			if !again {
				again = true
			}

		case <-done:
			done = nil
			again = false

		case <-fail:
			log.Error("capture error while doing task, shutting down...", "err", err)
			break LOOP
		}
	}

	return nil
}

func stressTestERC721Mint(ctx *cli.Context) error {
	path := ctx.String(pathFlag.Name)
	tokens, err := loadContractAddrs(path)
	if err != nil {
		return err
	}

	if len(tokens) == 0 {
		return errors.New("no ERC721 token address exists")
	}

	clients, err := initEthClients(ctx)
	if err != nil {
		return err
	}

	var (
		client        = clients[0]
		chainID, _    = client.ChainID(context.Background())
		adminAccount  = newAccount(ctx.GlobalString(privKeyFlag.Name), chainID)
		accountNumber = ctx.Int(accountNumberFlag.Name)
		total         = ctx.Int(totalTxsFlag.Name)
		threads       = ctx.Int(threadsFlag.Name)
		loop          = ctx.Bool(loopFlag.Name)
		intv          = ctx.Int(txGenPeriodFlag.Name)
		done          chan struct{}
		fail          = make(chan error, 1)
		again         = true
	)

	if threads > total || threads > len(tokens) {
		return errors.New("threads amount should lower than total tx amount and total token class")
	}

	if total < accountNumber {
		return errors.New("total tx amount should bigger than account amount")
	}

	if total%len(tokens) != 0 {
		return errors.New("total tx amount should be a multiple of total token class")
	}

	accounts, err := initAccounts(accountNumber, chainID)
	if err != nil {
		return err
	}

	err = initTransfer(adminAccount, accounts, client)
	if err != nil {
		return err
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	timer := time.NewTicker(time.Second * time.Duration(intv))
	defer timer.Stop()

	if !loop {
		go func() {
			sigs <- syscall.SIGTERM
		}()
	}

LOOP:
	for {
		if done == nil && again {
			done = make(chan struct{})
			txs, err := mintERC721Tokens(total, tokens, accounts, client)

			if err != nil {
				log.Error("generate signed ERC721 token mint txs falied", "err", err)
				fail <- err
				continue
			}
			log.Info("generate ERC721 token mint txs over", "total", len(txs))

			currentBlock, _ := client.BlockByNumber(context.Background(), nil)
			log.Info("current block", "number", currentBlock.Number())

			// send txs
			start := time.Now()
			err = stressSendTransactions(txs, threads, clients)

			if err != nil {
				log.Error("send ERC721 token mint txs falied", "err", err)
				fail <- err
				continue
			}
			log.Info("send ERC721 token mint txs over", "cost(milliseconds)", time.Since(start))

			close(done)
		}

		select {
		case <-sigs:
			log.Info("capture interupt, shutting down...")
			break LOOP

		case <-timer.C:
			if !again {
				again = true
			}

		case <-done:
			done = nil
			again = false

		case <-fail:
			log.Error("capture error while doing task, shutting down...", "err", err)
			break LOOP
		}
	}

	return nil
}

func deployERC721Contracts(ctx *cli.Context) error {
	clients, err := initEthClients(ctx)
	if err != nil {
		return err
	}

	var (
		client       = clients[0]
		chainID, _   = client.ChainID(context.Background())
		adminAccount = newAccount(ctx.GlobalString(privKeyFlag.Name), chainID)
		addDeveloper = ctx.Bool(addDeveloperFlag.Name)
		deploy       = ctx.Int(deployFlag.Name)
		path         = ctx.String(pathFlag.Name)
	)

	accounts, err := initAccounts(deploy, chainID)
	if err != nil {
		return err
	}

	err = initTransfer(adminAccount, accounts, client)
	if err != nil {
		return err
	}

	if addDeveloper {
		err = addDeveloperWhiteList(adminAccount, accounts, systemcontract.AddressListContractAddr, client)
		if err != nil {
			return err
		}
	}

	addrs, err := createDeployERC721ContractsTxs(accounts, client)
	if err != nil {
		log.Error("Deploy ERC721 token addresses failed", "err", err)
		return err
	}

	for i := 0; i < deploy; i++ {
		log.Info("Created ERC721 token successfully", "addr", addrs[i])
	}

	err = writeContractAddrs(path, addrs)
	if err != nil {
		log.Error("Write token addresses file failed", "err", err)
		return err
	}
	return nil
}

func deployERC1155Contracts(ctx *cli.Context) error {
	clients, err := initEthClients(ctx)
	if err != nil {
		return err
	}

	var (
		client       = clients[0]
		chainID, _   = client.ChainID(context.Background())
		adminAccount = newAccount(ctx.GlobalString(privKeyFlag.Name), chainID)
		addDeveloper = ctx.Bool(addDeveloperFlag.Name)
		deploy       = ctx.Int(deployFlag.Name)
		path         = ctx.String(pathFlag.Name)
	)

	accounts, err := initAccounts(deploy, chainID)
	if err != nil {
		return err
	}

	err = initTransfer(adminAccount, accounts, client)
	if err != nil {
		return err
	}

	if addDeveloper {
		err = addDeveloperWhiteList(adminAccount, accounts, systemcontract.AddressListContractAddr, client)
		if err != nil {
			return err
		}
	}

	addrs, err := createDeployERC1155ContractsTxs(accounts, client)
	if err != nil {
		log.Error("Deploy ERC1155 token addresses failed", "err", err)
		return err
	}

	for i := 0; i < len(addrs); i++ {
		log.Info("Created ERC1155 token successfully", "addr", addrs[i])
	}

	err = writeContractAddrs(path, addrs)
	if err != nil {
		log.Error("Write ERC1155 token addresses failed", "err", err)
		return err
	}
	return nil
}

func stressTestERC1155BatchMint(ctx *cli.Context) error {
	path := ctx.String(pathFlag.Name)
	tokens, err := loadContractAddrs(path)
	if err != nil {
		return err
	}

	if len(tokens) == 0 {
		return errors.New("no erc1155 token addresses exist")
	}

	clients, err := initEthClients(ctx)
	if err != nil {
		return err
	}

	var (
		client        = clients[0]
		chainId, _    = client.ChainID(context.Background())
		adminAccount  = newAccount(ctx.GlobalString(privKeyFlag.Name), chainId)
		accountNumber = ctx.Int(accountNumberFlag.Name)
		total         = ctx.Int(totalTxsFlag.Name)
		threads       = ctx.Int(threadsFlag.Name)
		loop          = ctx.Bool(loopFlag.Name)
		intv          = ctx.Int(txGenPeriodFlag.Name)
		done          chan struct{}
		fail          = make(chan error, 1)
		again         = true
	)

	if threads > total || threads > len(tokens) {
		return errors.New("threads amount should be lower than total txs and token kinds")
	}

	if total < accountNumber {
		return errors.New("total tx amount should bigger than account amount")
	}

	accounts, err := initAccounts(accountNumber, chainId)
	if err != nil {
		return err
	}

	err = initTransfer(adminAccount, accounts, client)
	if err != nil {
		return err
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	timer := time.NewTicker(time.Duration(intv) * time.Second)
	defer timer.Stop()

	if !loop {
		go func() {
			sigs <- syscall.SIGTERM
		}()
	}

LOOP:
	for {
		if done == nil && again {
			done = make(chan struct{})

			// generate erc1155 mint tx
			txs, err := mintERC1155Tokens(total, tokens, accounts, client)
			if err != nil {
				log.Error("generate signed ERC1155 token mint txs failed", "err", err)
				fail <- err
				continue
			}
			log.Info("generated signed ERC1155 token mint txs success", "total", len(txs))

			currentBlock, _ := client.BlockByNumber(context.Background(), nil)
			log.Info("current block", "number", currentBlock.Number())

			start := time.Now()
			err = stressSendTransactions(txs, threads, clients)
			if err != nil {
				log.Error("send ERC1155 token mint txs failed", "err", err)
				fail <- err
				continue
			}
			log.Info("send ERC1155 token mint txs success", "cost(milliseconds)", time.Since(start), "total", len(txs))

			close(done)
		}

		select {
		case <-done:
			done = nil
			again = false

		case <-timer.C:
			again = true

		case <-fail:
			log.Error("capture error while doing task, shutting down...")
			break LOOP

		case <-sigs:
			log.Info("capture interrupt, shutting down...")
			break LOOP
		}
	}

	return nil
}

func initMintErc1155(tokens []common.Address, accounts []*bind.TransactOpts, client *ethclient.Client) error {
	log.Info(">>> start init ERC1155 Token", "sending to", "test accounts")
	err := mint1155TokenRR(tokens, accounts, client)
	log.Info(">>> end init ERC1155 Token", "error", err != nil)
	return err
}

func stressTestERC1155BatchTransferFrom(ctx *cli.Context) error {
	// load contract address from path
	path := ctx.String(pathFlag.Name)
	tokens, err := loadContractAddrs(path)
	if err != nil {
		return err
	}

	// tokens len should be positive number
	if len(tokens) == 0 {
		return errors.New("no erc1155 token addresses")
	}

	// init clients
	clients, err := initEthClients(ctx)
	if err != nil {
		return err
	}

	// var
	var (
		client        = clients[0]
		chainID, _    = client.ChainID(context.Background())
		adminAccount  = newAccount(ctx.GlobalString(privKeyFlag.Name), chainID)
		accountNumber = ctx.Int(accountNumberFlag.Name)
		total         = ctx.Int(totalTxsFlag.Name)
		threads       = ctx.Int(threadsFlag.Name)
		addDeveloper  = ctx.Bool(addDeveloperFlag.Name)
		loop          = ctx.Bool(loopFlag.Name)
		intv          = ctx.Int(txGenPeriodFlag.Name)
		done          chan struct{}
		fail          = make(chan error, 1)
		again         = true
	)

	if threads > total {
		return errors.New("threads exceeded, should be lower than total")
	}
	if total < accountNumber || total%accountNumber != 0 {
		return errors.New("total should be a multiple of accountNumber")
	}

	accounts, err := initAccounts(accountNumber, chainID)
	if err != nil {
		return err
	}

	err = initTransfer(adminAccount, accounts, client)
	if err != nil {
		return err
	}

	if addDeveloper {
		err = addDeveloperWhiteList(adminAccount, accounts, systemcontract.AddressListContractAddr, client)
		if err != nil {
			return err
		}
	}

	// init mint
	err = initMintErc1155(tokens, accounts, client)
	if err != nil {
		return err
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	timer := time.NewTicker(time.Duration(intv) * time.Second)
	defer timer.Stop()

	if !loop {
		go func() {
			sigs <- syscall.SIGTERM
		}()
	}

LOOP:
	for {
		if done == nil && again {
			done = make(chan struct{})
			// generate batch transfer t8n
			txs, err := generateSignedERC1155TransferTransactions(total, tokens, accounts, client)
			if err != nil {
				log.Error("Failed to generate batch transfer transaction", "err", err)
				fail <- err
				continue
			}
			log.Info("Generate batch transfer transactions successfully")

			start := time.Now()
			err = stressSendTransactions(txs, threads, clients)

			if err != nil {
				log.Error("Failed to send batch transfer transactions", "err", err)
				fail <- err
				continue
			}

			log.Info("Send batch transfer transactions successfully", "Cost(ms)", time.Since(start))

			close(done)
		}

		select {
		case <-done:
			done = nil
			again = false

		case <-sigs:
			log.Info("Sending batch transfer transactions successfully")
			break LOOP

		case <-timer.C:
			if !again {
				again = true
			}

		case <-fail:
			log.Error("capture error while doing task, shutting down...", "err", err)
			break LOOP
		}
	}

	return nil

}
