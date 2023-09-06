package nft

type MetadataGetter interface {
	GetName() string
	GetImageURL() string
	GetAnimationURL() string
	GetURI() string
	GetCollectionName() string
	GetCollectionDescription() string
}
