package near

import (
	"github.com/deltaswapio/swaprouter/v3/log"
	"github.com/deltaswapio/swaprouter/v3/tokens"
	"github.com/near/borsh-go"
)

// SendTransaction send signed tx
func (b *Bridge) SendTransaction(signedTx interface{}) (txHash string, err error) {
	signTx := signedTx.(*SignedTransaction)
	buf, err := borsh.Serialize(*signTx)
	if err != nil {
		return "", err
	}
	txHash, err = b.BroadcastTxCommit(buf)
	if err != nil {
		return "", err
	}
	return txHash, nil
}

// BroadcastTxCommit broadcast tx
func (b *Bridge) BroadcastTxCommit(signedTx []byte) (result string, err error) {
	urls := b.GatewayConfig.AllGatewayURLs
	var success bool
	for _, url := range urls {
		result, err = BroadcastTxCommit(url, signedTx)
		if err == nil {
			success = true
		} else {
			log.Error("BroadcastTxCommit", "err", err)
		}
	}
	if success {
		return result, nil
	}
	return "", tokens.ErrBroadcastTx
}
