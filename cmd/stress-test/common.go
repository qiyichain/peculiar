package main

import (
	"context"
	"crypto/ecdsa"
	"encoding/binary"
	"encoding/hex"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"golang.org/x/sync/errgroup"
	"gopkg.in/urfave/cli.v1"
)

const (
	separator = ","
)

var g errgroup.Group

// newClient creates a client with specified remote URL.
func newClient(url string) *ethclient.Client {
	client, err := ethclient.Dial(url)
	if err != nil {
		utils.Fatalf("Failed to connect to Ethereum node: %v", err)
	}

	return client
}

func newClients(urls []string) []*ethclient.Client {
	clients := make([]*ethclient.Client, 0)

	for _, url := range urls {
		client, err := ethclient.Dial(url)
		if err != nil {
			continue
		}

		clients = append(clients, client)
	}

	return clients
}

func getRPCList(ctx *cli.Context) []string {
	urlStr := ctx.GlobalString(nodeURLFlag.Name)
	list := make([]string, 0)

	for _, url := range strings.Split(urlStr, separator) {
		if url = strings.Trim(url, " "); len(url) != 0 {
			list = append(list, url)
		}
	}
	if len(list) == 0 {
		utils.Fatalf("Failed to find any valid rpc url in: %v", urlStr)
	}

	return list
}

// newAccount creates a ethereum account with bind transactor by plaintext key string in hex format .
func newAccount(hexKey string, chainID *big.Int) *bind.TransactOpts {
	key, err := crypto.HexToECDSA(hexKey)
	if err != nil {
		utils.Fatalf("Failed to get privkey by hex key: %v", err)
	}

	transact, _ := bind.NewKeyedTransactorWithChainID(key, chainID)

	return transact
}

func newAccounts(keys []*ecdsa.PrivateKey, chainID *big.Int) []*bind.TransactOpts {
	accounts := make([]*bind.TransactOpts, 0)

	for _, key := range keys {
		transact, _ := bind.NewKeyedTransactorWithChainID(key, chainID)
		accounts = append(accounts, transact)
	}

	return accounts
}

// newRandomAccount generates a random ethereum account with bind transactor
func newRandomAccount(chainID *big.Int) *bind.TransactOpts {
	key, err := crypto.GenerateKey()
	if err != nil {
		utils.Fatalf("Failed to genreate random key: %v", err)
	}

	transact, _ := bind.NewKeyedTransactorWithChainID(key, chainID)

	return transact
}

// generateRandomAccounts generates servial random accounts
// concurrent do this if account amount is to big.
func generateRandomAccounts(amount int, chainID *big.Int) ([]*ecdsa.PrivateKey, []*bind.TransactOpts, error) {
	keys := make([]*ecdsa.PrivateKey, 0)
	result := make([]*bind.TransactOpts, 0)

	workFn := func(start, end int, data ...interface{}) ([]interface{}, error) {
		tmpAccounts := make([]interface{}, 0)
		for i := start; i < end; i++ {
			key, err := crypto.GenerateKey()
			if err != nil {
				log.Error("Generate keys failed: %v", err)
				return nil, err
			}

			tmpAccounts = append(tmpAccounts, key)
		}

		return tmpAccounts, nil
	}

	ret := concurrentWork(amount/jobsPerThread+1, amount, workFn, nil)
	if err := g.Wait(); err != nil {
		return nil, nil, err
	}

	for _, account := range ret {
		transact, _ := bind.NewKeyedTransactorWithChainID(account.(*ecdsa.PrivateKey), chainID)
		keys = append(keys, account.(*ecdsa.PrivateKey))
		result = append(result, transact)
	}

	return keys, result, nil
}

func sendEtherToRandomAccount(adminAccount *bind.TransactOpts, accounts []*bind.TransactOpts, amount *big.Int, client *ethclient.Client) error {
	nonce, err := client.PendingNonceAt(context.Background(), adminAccount.From)
	if err != nil {
		log.Error("Failed to get account nonce: %v", err)
		return err
	}

	var lastHash common.Hash
	for _, account := range accounts {
		signedTx, err := adminAccount.Signer(adminAccount.From, generateTransferTx(nonce, account.From, amount))
		if err != nil {
			log.Error("Failed to sign transaction", "from", account.From, "to", receiver,
				"nonce", nonce, "error", err)
			return err
		}
		if err := client.SendTransaction(context.Background(), signedTx); err != nil {
			log.Error("Failed to send ether to random account: %v", err)
			return err
		}

		lastHash = signedTx.Hash()
		nonce++
	}

	waitForTx(lastHash, client)
	return nil
}

// newSendEtherTransaction creates a transfer transfer transaction.
func newHBStansferTransaction(nonce uint64, to common.Address, amount *big.Int) *types.Transaction {
	gasPrice := big.NewInt(10)
	gasPrice.Mul(gasPrice, big.NewInt(params.GWei))

	return types.NewTransaction(nonce, to, amount, ethTransferLimit, gasPrice, []byte{})
}

func generateTransferTx(nonce uint64, to common.Address, amount *big.Int) *types.Transaction {
	return newHBStansferTransaction(nonce, to, amount)
}

func packTransferTokenData(to common.Address, amount *big.Int) []byte {
	data := make([]byte, 68)

	sig, _ := hex.DecodeString(tokenTransferSig)
	copy(data[:4], sig[:])

	toBytes := to.Bytes()
	copy(data[36-len(toBytes):36], toBytes[:])

	vBytes := amount.Bytes()
	copy(data[68-len(vBytes):], vBytes[:])

	return data
}

func packAddWhiteListData(addr common.Address) []byte {
	data := make([]byte, 36)

	sig, _ := hex.DecodeString(addDeveloperSig)
	copy(data[:4], sig[:])

	addrBytes := addr.Bytes()
	copy(data[36-len(addrBytes):], addrBytes)

	return data
}

func newAddDeveloperWhiteListTx(nonce uint64, to common.Address, account common.Address) *types.Transaction {
	gasPrice := big.NewInt(10)
	gasPrice.Mul(gasPrice, big.NewInt(params.GWei))

	return types.NewTransaction(nonce, to, new(big.Int), addDeveloperLimit, gasPrice, packAddWhiteListData(account))
}

func addDeveloperWhiteList(adminAccount *bind.TransactOpts, accounts []*bind.TransactOpts, to common.Address, client *ethclient.Client) error {
	nonce, err := client.PendingNonceAt(context.Background(), adminAccount.From)
	if err != nil {
		log.Error("Failed to get account nonce: %v", err)
		return err
	}

	var lastHash common.Hash
	for _, account := range accounts {
		signedTx, _ := adminAccount.Signer(adminAccount.From, newAddDeveloperWhiteListTx(nonce, to, account.From))
		if err := client.SendTransaction(context.Background(), signedTx); err != nil {
			log.Error("Failed to add devloper into whitelist: %v", err)
			return err
		}

		lastHash = signedTx.Hash()
		nonce++
	}

	waitForTx(lastHash, client)
	return nil
}

func createDeployERC721ContractsTxs(accounts []*bind.TransactOpts, client *ethclient.Client) ([]common.Address, error) {
	gasPrice := big.NewInt(10)
	gasPrice.Mul(gasPrice, big.NewInt(params.GWei))

	addrs := make([]common.Address, len(accounts))
	for i, account := range accounts {
		nonce, err := client.PendingNonceAt(context.Background(), account.From)
		if err != nil {
			log.Error("Failed to get account nonce: %v", err)
			return nil, err
		}

		account.GasPrice = gasPrice
		account.GasLimit = deployERC721Limit
		account.Value = new(big.Int)

		account.Nonce = big.NewInt(int64(nonce))
		addrs[i], err = deployERC721(account, client)

		if err != nil {
			log.Error("Failed to deploy ERC721 token contract: %v", err)
			return nil, err
		}
	}

	return addrs, nil
}

func deployERC721(adminAccount *bind.TransactOpts, client *ethclient.Client) (common.Address, error) {
	contractAddr, _, _, err := DeployERC721(adminAccount, client)
	if err != nil {
		log.Error("Failed to deploy ERC721 contract: %v", err)
		return common.Address{}, err
	}

	return contractAddr, nil
}

func createDeployERC1155ContractsTxs(accounts []*bind.TransactOpts, client *ethclient.Client) ([]common.Address, error) {
	gasPrice := big.NewInt(10)
	gasPrice.Mul(gasPrice, big.NewInt(params.GWei))

	// fixme: use make instead of
	addrs := make([]common.Address, len(accounts))
	for i, account := range accounts {
		nonce, err := client.PendingNonceAt(context.Background(), account.From)
		if err != nil {
			log.Error("Failed to fetch account nonce %v", err)
		}

		account.GasPrice = gasPrice
		account.GasLimit = deployERC1155Limit
		account.Value = new(big.Int)
		account.Nonce = big.NewInt(int64(nonce))

		addrs[i], err = deployERC1155(account, client)

		if err != nil {
			log.Error("Failed to deploy ERC721 token contract: %v", err)
			return nil, err
		}
	}

	return addrs, nil
}

func deployERC1155(adminAccount *bind.TransactOpts, client *ethclient.Client) (common.Address, error) {
	contractAddr, _, _, err := DeployERC1155(adminAccount, client)
	if err != nil {
		log.Error("Failed to deploy ERC1155 contract: %v", err)
		return common.Address{}, err
	}
	return contractAddr, nil
}

func getTokenCurrentID(token, owner common.Address, client *ethclient.Client) (uint64, error) {
	currentBlock, _ := client.BlockByNumber(context.Background(), nil)
	data, _ := hex.DecodeString(ERC721CurrentIDSig)

	msg := ethereum.CallMsg{
		From: owner,
		To:   &token,
		Data: data,
	}

	ret, err := client.CallContract(context.Background(), msg, currentBlock.Number())
	if err != nil {
		log.Error("Get ERCToken current ID Failed: %v", err)
		return 0, err
	}
	return binary.BigEndian.Uint64(ret[len(ret)-8:]), nil
}

func packMintTokenData(to common.Address) []byte {
	data := make([]byte, 132)

	sig, _ := hex.DecodeString(ERC721MintSig)
	copy(data[:4], sig[:])

	toBytes := to.Bytes()
	copy(data[36-len(toBytes):36], toBytes[:])

	data[36+leftSpace] = uriMetaData

	return data
}

func newMintERC721TokenTx(nonce uint64, to common.Address, token common.Address) *types.Transaction {
	gasPrice := big.NewInt(10)
	gasPrice.Mul(gasPrice, big.NewInt(params.GWei))

	return types.NewTransaction(nonce, token, new(big.Int), ERC721MintLimit, gasPrice, packMintTokenData(to))
}

func mintERC721Tokens(total int, tokens []common.Address, accounts []*bind.TransactOpts, client *ethclient.Client) ([]*types.Transaction, error) {
	txs := make([]*types.Transaction, 0)
	tokensLen, accountsLen := len(tokens), len(accounts)
	jobsPerThreadTmp := total / len(tokens)

	workFn := func(start, end int, data ...interface{}) ([]interface{}, error) {
		account := accounts[start%accountsLen]
		owner := accounts[start/jobsPerThreadTmp]
		token := tokens[start/jobsPerThreadTmp]
		result := make([]interface{}, 0)

		nonce, err := client.PendingNonceAt(context.Background(), owner.From)
		if err != nil {
			log.Error("Failed to get account nonce: %v", err)
			return nil, err
		}

		for i := start; i < end; i++ {
			signedTx, err := owner.Signer(owner.From, newMintERC721TokenTx(nonce, account.From, token))
			if err != nil {
				log.Error("Failed to sign transaction", "from", account.From, "to", token,
					"nonce", nonce, "error", err)
				return nil, err
			}
			result = append(result, signedTx)
			nonce++
		}

		return result, nil
	}

	result := concurrentWork(tokensLen, total, workFn, nil)
	if err := g.Wait(); err != nil {
		return nil, err
	}

	for _, tx := range result {
		tmp := tx.(*types.Transaction)
		txs = append(txs, tmp)
	}

	return txs, nil
}

func mintTokenRR(tokens []common.Address, accounts []*bind.TransactOpts, client *ethclient.Client) (map[common.Address]uint64, error) {
	var lastHash common.Hash
	tokenID := make(map[common.Address]uint64, len(tokens))
	for i, token := range tokens {
		owner := accounts[i].From
		nonce, err := client.PendingNonceAt(context.Background(), owner)
		if err != nil {
			log.Error("Failed to get account nonce: %v", err)
			return nil, err
		}

		startID, err := getTokenCurrentID(token, owner, client)
		if err != nil {
			return nil, err
		}
		tokenID[token] = startID

		for _, account := range accounts {
			signedTx, err := accounts[i].Signer(owner, newMintERC721TokenTx(nonce, account.From, token))
			if err != nil {
				log.Error("Failed to sign transaction", "from", account.From, "to", receiver,
					"nonce", nonce, "error", err)
				return nil, err
			}
			if err = client.SendTransaction(context.Background(), signedTx); err != nil {
				log.Error("Failed to send minted token to random account: %v", err)
				return nil, err
			}
			lastHash = signedTx.Hash()
			nonce++
		}
	}

	waitForTx(lastHash, client)
	return tokenID, nil
}

func packTokenTransferData(to common.Address, id uint64) []byte {
	data := make([]byte, 68)

	sig, _ := hex.DecodeString(ERC721TransferSig)
	copy(data[:4], sig[:])

	toBytes := to.Bytes()
	copy(data[36-len(toBytes):36], toBytes[:])

	var idBytes = make([]byte, 8)
	binary.BigEndian.PutUint64(idBytes, id)
	copy(data[68-len(idBytes):], idBytes)

	return data
}

func generateTransferTokenTx(nonce uint64, token, to common.Address, id uint64) *types.Transaction {
	gasPrice := big.NewInt(10)
	gasPrice.Mul(gasPrice, big.NewInt(params.GWei))

	return types.NewTransaction(nonce, token, new(big.Int), tokenTransferLimit, gasPrice, packTokenTransferData(to, id))
}

func generateSignedTokenTransactions(total int, tokens []common.Address, accounts []*bind.TransactOpts, startIDs map[common.Address]uint64, client *ethclient.Client) ([]*types.Transaction, error) {
	txs := make([]*types.Transaction, 0)
	tokensLen, jobsPerThreadTmp := len(tokens), total/len(accounts)
	workFn := func(start, end int, data ...interface{}) ([]interface{}, error) {
		id := uint64(start / jobsPerThreadTmp)
		account := accounts[id]
		currentNonce, err := client.PendingNonceAt(context.Background(), account.From)
		if err != nil {
			log.Error("Failed to get account nonce: %v", err)
			return nil, err
		}

		result := make([]interface{}, 0)
		for i := 0; i < end-start; i++ {
			tokenID := i % tokensLen
			token := tokens[tokenID]
			startID := startIDs[token]
			signedTx, err := account.Signer(account.From, generateTransferTokenTx(currentNonce, token, account.From, id+startID))
			if err != nil {
				log.Error("Failed to sign transaction", "from", account.From, "to", receiver,
					"nonce", currentNonce, "error", err)
				return nil, err
			}
			result = append(result, signedTx)

			currentNonce++
		}

		return result, nil
	}

	// accounts
	result := concurrentWork(len(accounts), total, workFn, nil)
	if err := g.Wait(); err != nil {
		return nil, err
	}

	for _, tx := range result {
		txs = append(txs, tx.(*types.Transaction))
	}

	return txs, nil
}

// generateSignedTransferTransactions generates transfer transactions.
func generateSignedTransferTransactions(total int, accounts []*bind.TransactOpts, amount *big.Int, client *ethclient.Client) ([]*types.Transaction, error) {
	txs := make([]*types.Transaction, 0)

	// total txs
	workFn := func(start, end int, data ...interface{}) ([]interface{}, error) {
		// like 15 threads, 15 account, 1000 txs
		account := accounts[start/(total/len(accounts))]
		currentNonce, err := client.PendingNonceAt(context.Background(), account.From)
		if err != nil {
			log.Error("Failed to get account nonce: %v", err)
			return nil, err
		}

		result := make([]interface{}, 0)
		for i := start; i < end; i++ {
			signedTx, err := account.Signer(account.From, generateTransferTx(currentNonce, receiver, amount))
			if err != nil {
				log.Error("Failed to sign transaction", "from", account.From, "to", receiver,
					"nonce", currentNonce, "error", err)
				return nil, err
			}
			result = append(result, signedTx)

			currentNonce++
		}

		return result, nil
	}

	// accounts
	result := concurrentWork(len(accounts), total, workFn, nil)
	if err := g.Wait(); err != nil {
		return nil, err
	}

	for _, tx := range result {
		txs = append(txs, tx.(*types.Transaction))
	}

	return txs, nil
}

func generateSignedTransferTransactionsRR(total int, accounts []*bind.TransactOpts, amount *big.Int, client *ethclient.Client) ([]*types.Transaction, error) {
	txs := make([]*types.Transaction, 0)
	result := make([]interface{}, 0)
	accountLen := len(accounts)
	accountsNonce := make(map[common.Address]uint64, accountLen)

	for _, account := range accounts {
		currentNonce, err := client.PendingNonceAt(context.Background(), account.From)
		if err != nil {
			log.Error("Failed to get account nonce: %v", err)
			return nil, err
		}
		accountsNonce[account.From] = currentNonce
	}

	for i := 0; i < total; i++ {
		account := accounts[i%accountLen]
		signedTx, err := account.Signer(account.From, generateTransferTx(accountsNonce[account.From], receiver, amount))
		if err != nil {
			log.Error("Failed to sign transaction", "from", account.From, "to", receiver,
				"nonce", accountsNonce[account.From], "error", err)
			return nil, err
		}
		accountsNonce[account.From]++
		result = append(result, signedTx)
	}

	for _, tx := range result {
		txs = append(txs, tx.(*types.Transaction))
	}

	return txs, nil
}

func waitForTx(hash common.Hash, client *ethclient.Client) {
	log.Info("wait for transaction packed", "tx", hash.Hex())
	for {
		receipt, _ := client.TransactionReceipt(context.Background(), hash)
		if receipt != nil {
			log.Info("transaction packed!")
			return
		}

		time.Sleep(time.Second)
	}
}

func stressSendTransactions(txs []*types.Transaction, threads int, clients []*ethclient.Client) error {
	jobsPerThreadTmp := len(txs) / threads

	workFn := func(start, end int, data ...interface{}) ([]interface{}, error) {
		c := clients[(start/jobsPerThreadTmp)%len(clients)]

		for i := start; i < end; i++ {
			if err := c.SendTransaction(context.Background(), txs[i]); err != nil {
				if err.Error() != core.ErrAlreadyKnown.Error() {
					log.Error("send tx failed", "err", err)
					return nil, err
				}
			}
		}

		return []interface{}{}, nil
	}

	concurrentWork(threads, len(txs), workFn, nil)
	if err := g.Wait(); err != nil {
		return err
	}
	return nil
}

func stressSendTransactionsRR(txs []*types.Transaction, threads int, clients []*ethclient.Client) error {
	// jobsPerThreadTmp := len(txs) / threads
	clientsLen := len(clients)
	workFn := func(start, end int, data ...interface{}) ([]interface{}, error) {
		for i := start; i < end; i++ {
			c := clients[i%clientsLen]
			if err := c.SendTransaction(context.Background(), txs[i]); err != nil {
				if err.Error() != core.ErrAlreadyKnown.Error() {
					log.Error("send tx failed", "err", err)
					return nil, err
				}
			}
		}

		return []interface{}{}, nil
	}

	concurrentWork(threads, len(txs), workFn, nil)
	if err := g.Wait(); err != nil {
		return err
	}
	return nil
}

func divisor(decimal int) *big.Int {
	if decimal <= 0 {
		return big.NewInt(1)
	}

	d := big.NewInt(10)
	for i := 0; i < decimal; i++ {
		d.Mul(d, big.NewInt(10))
	}

	return d
}

// type workFunc func(start, end int, data ...interface{}) []interface{}
type workFunc func(start, end int, data ...interface{}) ([]interface{}, error)

func concurrentWork(threads, totalWorks int, job workFunc, data ...interface{}) []interface{} {
	dataChan := make(chan []interface{})

	for i := 0; i < threads; i++ {
		i := i
		// go doJobFunc(i)
		g.Go(func() error {
			start := i * totalWorks / threads
			// cal end of the work
			end := (i + 1) * totalWorks / threads
			// fmt.Println("start: ", start, "   ", "end: ", end)
			if end > totalWorks {
				end = totalWorks
			}
			result, err := job(start, end, data)
			dataChan <- result
			return err
		})
	}

	// wait for all job done
	doneJob := 0
	result := make([]interface{}, 0)
	for {
		if doneJob == threads {
			break
		}

		select {
		case data := <-dataChan:
			result = append(result, data...)
			doneJob++
		}
	}

	return result
}
