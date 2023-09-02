package nft

type MetadataGetter interface {
	GetName() string
	GetImageURI() string
	GetURI() string
	GetCollectionName() string
	GetCollectionDescription() string
}
