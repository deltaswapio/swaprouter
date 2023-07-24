package bridge

import (
	"math/big"

	"github.com/deltaswapio/swaprouter/v3/log"
	"github.com/deltaswapio/swaprouter/v3/tokens"
	"github.com/deltaswapio/swaprouter/v3/tokens/aptos"
	"github.com/deltaswapio/swaprouter/v3/tokens/btc"
	"github.com/deltaswapio/swaprouter/v3/tokens/cardano"
	"github.com/deltaswapio/swaprouter/v3/tokens/cosmos"
	"github.com/deltaswapio/swaprouter/v3/tokens/eth"
	"github.com/deltaswapio/swaprouter/v3/tokens/flow"
	"github.com/deltaswapio/swaprouter/v3/tokens/iota"
	"github.com/deltaswapio/swaprouter/v3/tokens/near"
	"github.com/deltaswapio/swaprouter/v3/tokens/reef"
	"github.com/deltaswapio/swaprouter/v3/tokens/ripple"
	"github.com/deltaswapio/swaprouter/v3/tokens/solana"
	"github.com/deltaswapio/swaprouter/v3/tokens/stellar"
	"github.com/deltaswapio/swaprouter/v3/tokens/tron"
)

// NewCrossChainBridge new bridge
func NewCrossChainBridge(chainID *big.Int) tokens.IBridge {
	switch {
	case reef.SupportsChainID(chainID):
		return reef.NewCrossChainBridge()
	case solana.SupportChainID(chainID):
		return solana.NewCrossChainBridge()
	case cosmos.SupportsChainID(chainID):
		return cosmos.NewCrossChainBridge()
	case btc.SupportsChainID(chainID):
		return btc.NewCrossChainBridge()
	case cardano.SupportsChainID(chainID):
		return cardano.NewCrossChainBridge()
	case aptos.SupportsChainID(chainID):
		return aptos.NewCrossChainBridge()
	case tron.SupportsChainID(chainID):
		return tron.NewCrossChainBridge()
	case near.SupportsChainID(chainID):
		return near.NewCrossChainBridge()
	case iota.SupportsChainID(chainID):
		return iota.NewCrossChainBridge()
	case ripple.SupportsChainID(chainID):
		return ripple.NewCrossChainBridge()
	case stellar.SupportsChainID(chainID):
		return stellar.NewCrossChainBridge(chainID.String())
	case flow.SupportsChainID(chainID):
		return flow.NewCrossChainBridge()
	case chainID.Sign() <= 0:
		log.Fatal("wrong chainID", "chainID", chainID)
	default:
		return eth.NewCrossChainBridge()
	}
	return nil
}
