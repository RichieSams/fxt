package fxt

var (
	fxtMagic = []byte{0x10, 0x00, 0x04, 0x46, 0x78, 0x54, 0x16, 0x00}
)

type recordType int

const (
	recordTypeMetadata        recordType = 0
	recordTypeInitialization  recordType = 1
	recordTypeString          recordType = 2
	recordTypeThread          recordType = 3
	recordTypeEvent           recordType = 4
	recordTypeBlob            recordType = 5
	recordTypeUserspaceObject recordType = 6
	recordTypeKernelObject    recordType = 7
	recordTypeScheduling      recordType = 8
	recordTypeLog             recordType = 9
	recordTypeLargeBlob       recordType = 15
)

type metadataType int

const (
	metadataTypeProviderInfo    metadataType = 1
	metadataTypeProviderSection metadataType = 2
	metadataTypeProviderEvent   metadataType = 3
)

type providerEventType int

const (
	providerEventTypeBufferFilledUp providerEventType = 0
)
