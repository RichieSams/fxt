package fxt

type recordType uint8

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
	recordTypeLarge           recordType = 15
)

type argumentType uint8

const (
	argumentTypeNull    argumentType = 0
	argumentTypeInt32   argumentType = 1
	argumentTypeUInt32  argumentType = 2
	argumentTypeInt64   argumentType = 3
	argumentTypeUInt64  argumentType = 4
	argumentTypeDouble  argumentType = 5
	argumentTypeString  argumentType = 6
	argumentTypePointer argumentType = 7
	argumentTypeKOID    argumentType = 8
	argumentTypeBool    argumentType = 9
)

type metadataType uint8

const (
	metadataTypeProviderInfo    metadataType = 1
	metadataTypeProviderSection metadataType = 2
	metadataTypeProviderEvent   metadataType = 3
	metadataTypeTraceInfo       metadataType = 4
)

type ProviderID uint32

type ProviderEventType uint8

const (
	ProviderEventTypeBufferFilledUp ProviderEventType = 0
)

type traceInfoType uint8

const (
	traceInfoTypeMagicNumber traceInfoType = 0
)

// KernelObjectID is a unique identifier for a kernel object
// for example, a process or thread
type KernelObjectID uint64

// Thread uniquely identifies a thread within a process
type Thread struct {
	ProcessID KernelObjectID
	ThreadID  KernelObjectID
}

type eventType uint8

const (
	eventTypeInstant          eventType = 0
	eventTypeCounter          eventType = 1
	eventTypeDurationBegin    eventType = 2
	eventTypeDurationEnd      eventType = 3
	eventTypeDurationComplete eventType = 4
	eventTypeAsyncBegin       eventType = 5
	eventTypeAsyncInstant     eventType = 6
	eventTypeAsyncEnd         eventType = 7
	eventTypeFlowBegin        eventType = 8
	eventTypeFlowStep         eventType = 9
	eventTypeFlowEnd          eventType = 10
)

type BlobType uint8

const (
	BlobTypeData       BlobType = 1
	BlobTypeLastBranch BlobType = 2
	BlobTypePerfetto   BlobType = 3
)

type KernelObjectType uint8

const (
	kernelObjectTypeProcess KernelObjectType = 1
	kernelObjectTypeThread  KernelObjectType = 2
)

type schedulingRecordType uint8

const (
	schedulingRecordTypeContextSwitch schedulingRecordType = 1
	schedulingRecordTypeThreadWakeup  schedulingRecordType = 2
)

type ThreadStateType uint8

const (
	ThreadStateTypeNew       = 0
	ThreadStateTypeRunning   = 1
	ThreadStateTypeSuspended = 2
	ThreadStateTypeBlocked   = 3
	ThreadStateTypeDying     = 4
	ThreadStateTypeDead      = 5
)

type largeRecordType uint8

const (
	largeRecordTypeLargeBlob = 0
)

type largeBlobType uint8

const (
	largeBlobTypeWithMetadata = 0
	largeBlobTypeNoMetadata   = 1
)
