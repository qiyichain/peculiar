package main

import (
	"bufio"
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"os"

	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

func writeAccounts(path string, accounts []*ecdsa.PrivateKey) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	lines := make([]string, 0)
	for _, account := range accounts {
		lines = append(lines, hex.EncodeToString(crypto.FromECDSA(account)))
	}

	w := bufio.NewWriter(file)
	for _, line := range lines {
		fmt.Fprintln(w, line)
	}
	return w.Flush()
}

func appendAccounts(path string, accounts []*ecdsa.PrivateKey) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	for _, account := range accounts {
		if _, err := f.WriteString(hex.EncodeToString(crypto.FromECDSA(account)) + "\n"); err != nil {
			return err
		}
	}

	return nil
}

func loadAccounts(path string) ([]*ecdsa.PrivateKey, error) {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			file, err = os.Create(path)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}
	defer file.Close()

	accounts := make([]*ecdsa.PrivateKey, 0)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		key, err := crypto.HexToECDSA(scanner.Text())
		if err != nil {
			continue
		}
		accounts = append(accounts, key)
	}

	return accounts, scanner.Err()
}

func getStorePath() string {
	return storePath
}

func writeContractAddrs(path string, addrs []common.Address) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	lines := make([]string, 0)
	for _, addr := range addrs {
		lines = append(lines, addr.String())
	}

	w := bufio.NewWriter(file)
	for _, line := range lines {
		fmt.Fprintln(w, line)
	}
	return w.Flush()
}

func loadContractAddrs(path string) ([]common.Address, error) {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			utils.Fatalf("Please use deploy721 or deploy1155 commands to deploy contracts on chain first")
		}
		return nil, err
	}
	defer file.Close()

	addrs := make([]common.Address, 0)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		addr := common.HexToAddress(scanner.Text())
		addrs = append(addrs, addr)
	}

	return addrs, scanner.Err()
}
