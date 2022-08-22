package types

import "github.com/ethereum/go-ethereum/common"

// EvmExtraValidator contains some extra validations to a transaction,
// and the validator is used inside the evm.
type EvmExtraValidator interface {
	// IsAddressDenied returns whether an address is denied.
	IsAddressDenied(address common.Address, cType common.AddressCheckType) bool
	// IsLogDenied returns whether a log (contract event) is denied.
	IsLogDenied(log *Log) bool

	// TODO(yqq): 2022-08-21, 
	// Add validation in evm excutor?
	// If a node syncs from block 0 ? 
	// In some cases, a tx avoids txpool validateTx 
	// and be excuted by evm directly.
	// whether it could cause a node's chain state be inconsistency with others?
	//
	// IsBEndAuthorized(common.Address) bool
}
