package types

import (
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
	"math/big"
)

var (
	MetaPrefix           = "234d6574615472616e73616374696f6e23"
	ErrInvalidFeePercent = errors.New("invalid FeePrecent")
	ErrInvalidMetaData   = errors.New("invalid metadata")
	BIG100               = new(big.Int).SetUint64(100)
	MetaPrefixBytesLen   = 17
)

type MetaData struct {
	FeePercent uint64 `json:"feeprecent" gencodec:"required"`
	// Signature values
	V       *big.Int `json:"v" gencodec:"required"`
	R       *big.Int `json:"r" gencodec:"required"`
	S       *big.Int `json:"s" gencodec:"required"`
	Payload []byte   `json:"input"    gencodec:"required"`
}

func IsMetaTransaction(data []byte) bool {
	if len(data) >= MetaPrefixBytesLen {
		prefix := hex.EncodeToString(data[:MetaPrefixBytesLen])
		return prefix == MetaPrefix
	}
	return false
}

func DecodeMetaData(encodedData []byte) (*MetaData, error) {
	metaData := new(MetaData)
	if len(encodedData) <= MetaPrefixBytesLen {
		return metaData, ErrInvalidMetaData
	}
	encodedData = encodedData[MetaPrefixBytesLen:]
	if err := rlp.DecodeBytes(encodedData, metaData); err != nil {
		fmt.Println(err)
		return metaData, err
	}
	if metaData.FeePercent > 100 {
		return metaData, ErrInvalidFeePercent
	}
	return metaData, nil
}

func (metadata *MetaData) ParseMetaData(nonce uint64, gasPrice *big.Int, gas uint64, to *common.Address, value *big.Int, payload []byte, from common.Address, chainID *big.Int) (common.Address, error) {
	var data interface{} = []interface{}{
		nonce,
		gasPrice,
		gas,
		to,
		value,
		payload,
		from,
		metadata.FeePercent,
	}
	raw, _ := rlp.EncodeToBytes(data)
	log.Debug("meta rlpencode" + hexutil.Encode(raw[:]))
	hash := rlpHash(data)
	log.Debug("meta rlpHash", hexutil.Encode(hash[:]))

	var big8 = big.NewInt(8)
	chainMul := new(big.Int).Mul(chainID, big.NewInt(2))
	V := new(big.Int).Sub(metadata.V, chainMul)
	V.Sub(V, big8)
	addr, err := RecoverPlain(hash, metadata.R, metadata.S, V, true)
	if err != nil {
		return common.HexToAddress(""), err
	}
	return addr, nil
}
