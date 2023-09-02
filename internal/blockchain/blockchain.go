package blockchain

import (
	"fmt"
	"strings"

	"github.com/redgoat650/barnacle-net/internal/blockchain/loopring"
	"github.com/redgoat650/barnacle-net/internal/blockchain/nft"
)

const (
	LoopringChainName = "loopring"
)

type Profile struct {
	Name   string
	Chain  string
	APIKey []byte
}

func GetNFTMetadata(walletID string, prof Profile) ([]nft.MetadataGetter, error) {
	switch strings.ToLower(prof.Chain) {
	case LoopringChainName:
		return loopring.GetNFTMetadata(walletID, prof.APIKey)
	default:
		return nil, fmt.Errorf("unsupported blockchain %s referenced in profile %s", prof.Chain, prof.Name)
	}
}
