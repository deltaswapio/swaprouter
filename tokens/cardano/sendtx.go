package cardano

import "github.com/deltaswapio/swaprouter/v3/log"

// SendTransaction send signed tx
func (b *Bridge) SendTransaction(signedTx interface{}) (string, error) {
	signedTransaction := signedTx.(*SignedTransaction)

	txhash, err := b.RpcClient.SubmitTx(signedTransaction.Tx)
	if err != nil {
		return "", err
	}
	log.Info("CardanoSubmitTx", "txhash", txhash.String(), "savedTxHash", signedTransaction.TxHash)

	TransactionChaining.InputKey.TxHash = signedTransaction.TxHash
	TransactionChaining.InputKey.TxIndex = signedTransaction.TxIndex
	TransactionChaining.AssetsMap = signedTransaction.AssetsMap
	AddTransactionChainingKeyCache(signedTransaction.TxHash, &signedTransaction.TxIns)
	return signedTransaction.TxHash, nil
}
