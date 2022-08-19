package main

import (
	"context"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConcurrentWorks(t *testing.T) {
	// generate new account
	workFn := func(start, end int, data ...interface{}) ([]interface{}, error) {
		tmpAccounts := make([]interface{}, 0)
		for i := start; i < end; i++ {
			tmpAccounts = append(tmpAccounts, newRandomAccount(big.NewInt(int64(2285))))
		}

		return tmpAccounts, nil
	}
	accounts := concurrentWork(10, 101, workFn, context.Background(), nil)
	g.Wait()
	assert.Equal(t, 1000, len(accounts))

	accounts = concurrentWork(10, 5, workFn, context.Background(), nil)
	g.Wait()
	assert.Equal(t, 5, len(accounts))
}
