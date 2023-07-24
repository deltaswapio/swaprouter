package tron

import (
	"bytes"
	"errors"
	"fmt"
	"strings"

	"github.com/deltaswapio/swaprouter/v3/common"
	"github.com/deltaswapio/swaprouter/v3/log"
	"github.com/deltaswapio/swaprouter/v3/params"
	"github.com/deltaswapio/swaprouter/v3/router"
	"github.com/deltaswapio/swaprouter/v3/tokens"
	"github.com/deltaswapio/swaprouter/v3/tokens/eth/abicoder"
	"github.com/deltaswapio/swaprouter/v3/types"

	tronaddress "github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"

	//nolint:staticcheck // ignore SA1019
	"github.com/golang/protobuf/ptypes"
)

// router contract's log topics
var (
	// LogAnySwapOut(address token, address from, address to, uint amount, uint fromChainID, uint toChainID);
	LogAnySwapOutTopic = common.FromHex("0x97116cf6cd4f6412bb47914d6db18da9e16ab2142f543b86e207c24fbd16b23a")
	// LogAnySwapOut(address token, address from, string to, uint amount, uint fromChainID, uint toChainID);
	LogAnySwapOut2Topic = common.FromHex("0x409e0ad946b19f77602d6cf11d59e1796ddaa4828159a0b4fb7fa2ff6b161b79")
	// LogAnySwapOutAndCall(address token, address from, string to, uint amount, uint fromChainID, uint toChainID, string anycallProxy, bytes data);
	LogAnySwapOutAndCallTopic = common.FromHex("0x8e7e5695fff09074d4c7d6c71615fd382427677f75f460c522357233f3bd3ec3")
)

func (b *Bridge) verifyERC20SwapTx(txHash string, logIndex int, allowUnstable bool) (*tokens.SwapTxInfo, error) {
	swapInfo := &tokens.SwapTxInfo{SwapInfo: tokens.SwapInfo{ERC20SwapInfo: &tokens.ERC20SwapInfo{}}}
	swapInfo.SwapType = tokens.ERC20SwapType                          // SwapType
	swapInfo.Hash = strings.TrimPrefix(strings.ToLower(txHash), "0x") // Hash
	swapInfo.LogIndex = logIndex                                      // LogIndex

	err := b.checkTxSuccess(swapInfo, allowUnstable)
	if err != nil {
		return swapInfo, err
	}

	var logs []*types.RPCLog
	logs, err = b.GetTransactionLog(swapInfo.Hash)
	if err != nil {
		return swapInfo, err
	}
	if logIndex >= len(logs) {
		return swapInfo, tokens.ErrLogIndexOutOfRange
	}

	err = b.verifyERC20SwapTxLog(swapInfo, logs[logIndex])
	if err != nil {
		return swapInfo, err
	}

	err = b.checkERC20SwapInfo(swapInfo)
	if err != nil {
		return swapInfo, err
	}

	if !allowUnstable {
		ctx := []interface{}{
			"identifier", params.GetIdentifier(),
			"from", swapInfo.From, "to", swapInfo.To,
			"bind", swapInfo.Bind, "value", swapInfo.Value,
			"txid", txHash, "logIndex", logIndex,
			"height", swapInfo.Height, "timestamp", swapInfo.Timestamp,
			"fromChainID", swapInfo.FromChainID, "toChainID", swapInfo.ToChainID,
			"token", swapInfo.ERC20SwapInfo.Token, "tokenID", swapInfo.ERC20SwapInfo.TokenID,
		}
		if swapInfo.ERC20SwapInfo.CallProxy != "" {
			ctx = append(ctx,
				"callProxy", swapInfo.ERC20SwapInfo.CallProxy,
			)
		}
		log.Info("verify router swap tx stable pass", ctx...)
	}

	return swapInfo, nil
}

func (b *Bridge) checkERC20SwapInfo(swapInfo *tokens.SwapTxInfo) error {
	err := b.checkCallByContract(swapInfo)
	if err != nil {
		return err
	}

	if swapInfo.FromChainID.String() != b.ChainConfig.ChainID {
		log.Error("router swap tx with mismatched fromChainID in receipt", "txid", swapInfo.Hash, "logIndex", swapInfo.LogIndex, "fromChainID", swapInfo.FromChainID, "toChainID", swapInfo.ToChainID, "chainID", b.ChainConfig.ChainID)
		return tokens.ErrFromChainIDMismatch
	}
	if swapInfo.FromChainID.Cmp(swapInfo.ToChainID) == 0 {
		return tokens.ErrSameFromAndToChainID
	}
	erc20SwapInfo := swapInfo.ERC20SwapInfo
	fromTokenCfg := b.GetTokenConfig(erc20SwapInfo.Token)
	if fromTokenCfg == nil || erc20SwapInfo.TokenID == "" {
		return tokens.ErrMissTokenConfig
	}
	multichainToken := router.GetCachedMultichainToken(erc20SwapInfo.TokenID, swapInfo.ToChainID.String())
	if multichainToken == "" {
		log.Warn("get multichain token failed", "tokenID", erc20SwapInfo.TokenID, "chainID", swapInfo.ToChainID, "txid", swapInfo.Hash)
		return tokens.ErrMissTokenConfig
	}
	toBridge := router.GetBridgeByChainID(swapInfo.ToChainID.String())
	if toBridge == nil {
		return tokens.ErrNoBridgeForChainID
	}
	toTokenCfg := toBridge.GetTokenConfig(multichainToken)
	if toTokenCfg == nil {
		log.Warn("get token config failed", "chainID", swapInfo.ToChainID, "token", multichainToken)
		return tokens.ErrMissTokenConfig
	}
	if !tokens.CheckTokenSwapValue(swapInfo, fromTokenCfg.Decimals, toTokenCfg.Decimals) {
		return tokens.ErrTxWithWrongValue
	}
	dstBridge := router.GetBridgeByChainID(swapInfo.ToChainID.String())
	if dstBridge == nil {
		return tokens.ErrNoBridgeForChainID
	}
	if !dstBridge.IsValidAddress(swapInfo.Bind) {
		log.Warn("wrong bind address in erc20 swap", "txid", swapInfo.Hash, "logIndex", swapInfo.LogIndex, "bind", swapInfo.Bind)
		return tokens.ErrWrongBindAddress
	}
	return nil
}

func (b *Bridge) checkTxSuccess(swapInfo *tokens.SwapTxInfo, allowUnstable bool) (err error) {
	if err != nil {
		return err
	}
	txStatus, err := b.GetTransactionStatus(swapInfo.Hash)
	if err != nil {
		return err
	}
	if txStatus == nil {
		log.Error("get tx receipt failed", "hash", swapInfo.Hash, "err", err)
		return err
	}
	if txStatus == nil || txStatus.BlockHeight == 0 {
		return tokens.ErrTxNotFound
	}
	if txStatus.BlockHeight < b.ChainConfig.InitialHeight {
		return tokens.ErrTxBeforeInitialHeight
	}

	swapInfo.Height = txStatus.BlockHeight  // Height
	swapInfo.Timestamp = txStatus.BlockTime // Timestamp

	if !allowUnstable && txStatus.Confirmations < b.ChainConfig.Confirmations {
		return tokens.ErrTxNotStable
	}

	tx, err := b.GetTronTransaction(swapInfo.Hash)
	if err != nil {
		return err
	}

	ret := tx.GetRet()
	if len(ret) != 1 {
		return errors.New("tron tx return not found")
	}
	if txret := ret[0].GetRet(); txret != core.Transaction_Result_SUCESS {
		return fmt.Errorf("tron tx not success: %+v", txret)
	}
	if cret := ret[0].GetContractRet(); cret != core.Transaction_Result_SUCCESS {
		return fmt.Errorf("tron tx contract not success: %+v", cret)
	}
	contract := tx.RawData.Contract[0]
	switch contract.Type {
	case core.Transaction_Contract_TriggerSmartContract:
		var c core.TriggerSmartContract
		//nolint:staticcheck // ignore SA1019
		err := ptypes.UnmarshalAny(contract.GetParameter(), &c)
		if err != nil {
			return errors.New("tx inconsistent")
		}
		from := fmt.Sprintf("%v", tronaddress.Address(c.OwnerAddress))
		contractAddress := tronaddress.Address(c.ContractAddress).String()
		if contractAddress == "" && !params.AllowCallByConstructor() {
			return tokens.ErrTxWithWrongContract
		} else {
			swapInfo.TxTo = contractAddress
		}
		swapInfo.From = from
	default:
		return errors.New("tron tx unknown error")
	}

	return nil
}

func (b *Bridge) checkCallByContract(swapInfo *tokens.SwapTxInfo) error {
	txTo := swapInfo.TxTo
	routerContract := b.GetRouterContract(swapInfo.ERC20SwapInfo.Token)
	if routerContract == "" {
		return tokens.ErrMissRouterInfo
	}

	if !params.AllowCallByContract() &&
		!common.IsEqualIgnoreCase(txTo, routerContract) &&
		!params.IsInCallByContractWhitelist(b.ChainConfig.ChainID, txTo) {
		if params.CheckEIP1167Master() {
			master := b.GetEIP1167Master(txTo)
			if master != (common.Address{}) &&
				params.IsInCallByContractWhitelist(b.ChainConfig.ChainID, master.LowerHex()) {
				return nil
			}
		}
		if params.HasCallByContractCodeHashWhitelist(b.ChainConfig.ChainID) {
			codehash := b.GetContractCodeHash(txTo)
			if codehash != (common.Hash{}) &&
				params.IsInCallByContractCodeHashWhitelist(b.ChainConfig.ChainID, codehash.String()) {
				return nil
			}
		}
		log.Warn("tx to with wrong contract", "txTo", txTo, "want", routerContract)
		return tokens.ErrTxWithWrongContract
	}

	return nil
}

func (b *Bridge) verifyERC20SwapTxLog(swapInfo *tokens.SwapTxInfo, rlog *types.RPCLog) (err error) {
	logTronAddr := convertToTronAddress(rlog.Address.Bytes()) // To

	swapInfo.To = logTronAddr

	logTopic := rlog.Topics[0].Bytes()
	switch {
	case bytes.Equal(logTopic, LogAnySwapOutTopic):
		err = b.parseERC20SwapoutTxLog(swapInfo, rlog)
	case bytes.Equal(logTopic, LogAnySwapOut2Topic):
		err = b.parseERC20Swapout2TxLog(swapInfo, rlog)
	case bytes.Equal(logTopic, LogAnySwapOutAndCallTopic):
		err = b.parseERC20SwapoutAndCallTxLog(swapInfo, rlog)
	default:
		return tokens.ErrSwapoutLogNotFound
	}
	if err != nil {
		log.Info(b.ChainConfig.BlockChain+" verifyERC20SwapTxLog fail", "tx", swapInfo.Hash, "logIndex", swapInfo.LogIndex, "err", err)
		return err
	}

	if rlog.Removed != nil && *rlog.Removed {
		return tokens.ErrTxWithRemovedLog
	}

	routerContract := b.GetRouterContract(swapInfo.ERC20SwapInfo.Token)
	if routerContract == "" {
		return tokens.ErrMissRouterInfo
	}
	if !common.IsEqualIgnoreCase(logTronAddr, routerContract) {
		log.Warn("router contract mismatch", "have", logTronAddr, "want", routerContract)
		return tokens.ErrTxWithWrongContract
	}
	return nil
}

func (b *Bridge) parseERC20SwapoutTxLog(swapInfo *tokens.SwapTxInfo, rlog *types.RPCLog) error {
	logTopics := rlog.Topics
	if len(logTopics) != 4 {
		return tokens.ErrTxWithWrongTopics
	}
	logData := *rlog.Data
	if len(logData) != 96 {
		return abicoder.ErrParseDataError
	}
	erc20SwapInfo := swapInfo.ERC20SwapInfo
	erc20SwapInfo.Token = convertToTronAddress(logTopics[1].Bytes())
	swapInfo.From = convertToTronAddress(logTopics[2].Bytes())
	swapInfo.Bind = common.BytesToAddress(logTopics[3].Bytes()).LowerHex()
	swapInfo.Value = common.GetBigInt(logData, 0, 32)
	swapInfo.FromChainID = b.ChainConfig.GetChainID()
	swapInfo.ToChainID = common.GetBigInt(logData, 64, 32)

	tokenCfg := b.GetTokenConfig(erc20SwapInfo.Token)
	if tokenCfg == nil {
		return tokens.ErrMissTokenConfig
	}
	erc20SwapInfo.TokenID = tokenCfg.TokenID

	return nil
}

func (b *Bridge) parseERC20Swapout2TxLog(swapInfo *tokens.SwapTxInfo, rlog *types.RPCLog) (err error) {
	logTopics := rlog.Topics
	if len(logTopics) != 3 {
		return tokens.ErrTxWithWrongTopics
	}
	logData := *rlog.Data
	if len(logData) < 160 {
		return abicoder.ErrParseDataError
	}
	erc20SwapInfo := swapInfo.ERC20SwapInfo
	erc20SwapInfo.Token = convertToTronAddress(logTopics[1].Bytes())
	swapInfo.From = convertToTronAddress(logTopics[2].Bytes())
	swapInfo.Bind, err = abicoder.ParseStringInData(logData, 0)
	if err != nil {
		return err
	}
	swapInfo.Value = common.GetBigInt(logData, 32, 32)
	if params.IsUseFromChainIDInReceiptDisabled(b.ChainConfig.ChainID) {
		swapInfo.FromChainID = b.ChainConfig.GetChainID()
	} else {
		swapInfo.FromChainID = common.GetBigInt(logData, 64, 32)
	}
	swapInfo.ToChainID = common.GetBigInt(logData, 96, 32)

	tokenCfg := b.GetTokenConfig(erc20SwapInfo.Token)
	if tokenCfg == nil {
		return tokens.ErrMissTokenConfig
	}
	erc20SwapInfo.TokenID = tokenCfg.TokenID

	return nil
}

func (b *Bridge) parseERC20SwapoutAndCallTxLog(swapInfo *tokens.SwapTxInfo, rlog *types.RPCLog) (err error) {
	logTopics := rlog.Topics
	if len(logTopics) != 3 {
		return tokens.ErrTxWithWrongTopics
	}
	logData := *rlog.Data
	if len(logData) < 288 {
		return abicoder.ErrParseDataError
	}
	erc20SwapInfo := swapInfo.ERC20SwapInfo
	erc20SwapInfo.Token = convertToTronAddress(logTopics[1].Bytes())
	swapInfo.From = convertToTronAddress(logTopics[2].Bytes())
	swapInfo.Bind, err = abicoder.ParseStringInData(logData, 0)
	if err != nil {
		return err
	}
	swapInfo.Value = common.GetBigInt(logData, 32, 32)
	if params.IsUseFromChainIDInReceiptDisabled(b.ChainConfig.ChainID) {
		swapInfo.FromChainID = b.ChainConfig.GetChainID()
	} else {
		swapInfo.FromChainID = common.GetBigInt(logData, 64, 32)
	}
	swapInfo.ToChainID = common.GetBigInt(logData, 96, 32)

	erc20SwapInfo.CallProxy, err = abicoder.ParseStringInData(logData, 128)
	if err != nil {
		return err
	}
	erc20SwapInfo.CallData, err = abicoder.ParseBytesInData(logData, 160)
	if err != nil {
		return err
	}

	tokenCfg := b.GetTokenConfig(erc20SwapInfo.Token)
	if tokenCfg == nil {
		return tokens.ErrMissTokenConfig
	}
	erc20SwapInfo.TokenID = tokenCfg.TokenID

	return nil
}
