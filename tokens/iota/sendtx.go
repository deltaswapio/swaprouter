package iota

import (
	"github.com/deltaswapio/swaprouter/v3/log"
	"github.com/deltaswapio/swaprouter/v3/tokens"
	iotago "github.com/iotaledger/iota.go/v2"
)

// SendTransaction send signed tx
func (b *Bridge) SendTransaction(signedTx interface{}) (txHash string, err error) {
	message := signedTx.(*iotago.Message)
	urls := b.GetGatewayConfig().AllGatewayURLs
	for _, url := range urls {
		if txHash, err = CommitMessage(url, message); err == nil {
			return txHash, nil
		} else {
			log.Warn("CommitMessage", "err", err)
		}
	}
	return "", tokens.ErrCommitMessage
}
