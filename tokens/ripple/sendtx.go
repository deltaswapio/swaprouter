package ripple

import (
	"fmt"
	"time"

	"github.com/deltaswapio/swaprouter/v3/log"
	"github.com/deltaswapio/swaprouter/v3/rpc/client"
	"github.com/deltaswapio/swaprouter/v3/tokens"
	"github.com/deltaswapio/swaprouter/v3/tokens/ripple/rubblelabs/ripple/data"
	"github.com/deltaswapio/swaprouter/v3/tokens/ripple/rubblelabs/ripple/websockets"
)

// SendTransaction send signed tx
func (b *Bridge) SendTransaction(signedTx interface{}) (txHash string, err error) {
	tx, ok := signedTx.(data.Transaction)
	if !ok {
		return "", tokens.ErrWrongRawTx
	}
	_, raw, err := data.Raw(tx)
	if err != nil {
		return "", err
	}
	rpcParams := map[string]interface{}{
		"tx_blob": fmt.Sprintf("%X", raw),
	}
	var success bool
	urls := b.GetGatewayConfig().AllGatewayURLs
	for i := 0; i < rpcRetryTimes; i++ {
		// try send to all remotes
		for _, url := range urls {
			var resp *websockets.SubmitResult
			err = client.RPCPostWithTimeout(b.RPCClientTimeout, &resp, url, "submit", rpcParams)
			if err != nil || resp == nil {
				log.Warn("Try sending transaction failed", "error", err)
				continue
			}
			if !resp.EngineResult.Success() {
				log.Warn("send tx with error result", "result", resp.EngineResult, "message", resp.EngineResultMessage)
			}
			txHash = tx.GetBase().Hash.String()
			success = true
		}
		if success {
			break
		}
		time.Sleep(rpcRetryInterval)
	}
	if success {
		return txHash, nil
	}
	return "", err
}
