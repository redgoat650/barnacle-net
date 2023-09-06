package loopring

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"

	"github.com/redgoat650/barnacle-net/internal/blockchain/nft"
)

type UserNFTBalancesPayload struct {
	TotalNum int       `json:"totalNum"`
	Data     []NFTData `json:"data,omitempty"`
}

type NFTData struct {
	ID             string         `json:"id"`
	AccountID      string         `json:"accountId"`
	TokenID        string         `json:"tokenId"`
	NFTType        string         `json:"nftType"`
	Metadata       *Metadata      `json:"metadata,omitempty"`
	CollectionInfo CollectionInfo `json:"collectionInfo,omitempty"`
}

func (d NFTData) GetName() string {
	return d.Metadata.Base.Name
}

func (d NFTData) GetURI() string {
	return d.Metadata.URI
}

func (d NFTData) GetImageURL() string {
	return d.Metadata.Base.Image
}

func (d NFTData) GetAnimationURL() string {
	return d.Metadata.Extra.AnimationURL
}

func (d NFTData) GetCollectionName() string {
	return d.CollectionInfo.Name
}

func (d NFTData) GetCollectionDescription() string {
	return d.CollectionInfo.Description
}

type Metadata struct {
	URI   string `json:"uri,omitempty"`
	Base  Base   `json:"base,omitempty"`
	Extra Extra  `json:"extra,omitempty"`
}

type Base struct {
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	Image       string `json:"image,omitempty"`
	Properties  string `json:"properties,omitempty"`
}

type CollectionInfo struct {
	ID          string `json:"id"`
	Owner       string `json:"owner,omitempty"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	NFTType     string `json:"nftType,omitempty"`
}

type Extra struct {
	AnimationURL string `json:"animationURL,omitempty"`
}

const (
	loopringAPIBaseURL = "https://api3.loopring.io"
	loopringAPIV3      = "api/v3"
	userResource       = "user"
	nftResource        = "nft"
	balancesResource   = "balances"

	addressParamKey  = "address"
	metadataParamKey = "metadata"
	limitParamKey    = "limit"
	offsetParamKey   = "offset"

	apiKeyHeaderKey = "X-API-KEY"
	maxRecordLimit  = 50
)

func GetNFTMetadata(walletID string, apiKey []byte) ([]nft.MetadataGetter, error) {
	pl, err := getBatchNFTMetadata(walletID, apiKey, maxRecordLimit, 0)
	if err != nil {
		return nil, err
	}

	nftData := pl.Data

	for pl.TotalNum > len(nftData) {
		pl, err = getBatchNFTMetadata(walletID, apiKey, 0, len(nftData))
		if err != nil {
			return nil, err
		}

		nftData = append(nftData, pl.Data...)
	}

	var ret []nft.MetadataGetter

	for _, d := range nftData {
		ret = append(ret, nft.MetadataGetter(d))
	}

	return ret, nil
}

func getBatchNFTMetadata(walletID string, apiKey []byte, limit, offset int) (*UserNFTBalancesPayload, error) {
	fullURI, err := url.JoinPath(loopringAPIBaseURL, loopringAPIV3, userResource, nftResource, balancesResource)
	if err != nil {
		return nil, err
	}

	params := url.Values{}
	params.Add(addressParamKey, walletID)
	params.Add(metadataParamKey, strconv.FormatBool(true))
	params.Add(limitParamKey, strconv.Itoa(limit))
	params.Add(offsetParamKey, strconv.Itoa(offset))

	u, err := url.ParseRequestURI(fullURI)
	if err != nil {
		return nil, err
	}

	u.RawQuery = params.Encode()

	log.Println("issuing loopring API call:", u.String())

	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set(apiKeyHeaderKey, string(apiKey))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	payload := UserNFTBalancesPayload{}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(b, &payload)
	if err != nil {
		return nil, err
	}

	return &payload, nil
}
