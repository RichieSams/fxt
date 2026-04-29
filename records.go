package fxt

// FXTRecord is a way to constrain what types can be returned in the record stream
type FXTRecord interface {
	fxtRecord()
}

type timestampedRecord interface {
	getTimestampNS() uint64
}

type InstantEventRecord struct {
	Timestamp uint64
	Category  string
	Name      string
	Thread    Thread
	Args      map[string]any
}

func (InstantEventRecord) fxtRecord() {}

type CounterEventRecord struct {
	Timestamp uint64
	Category  string
	Name      string
	Thread    Thread
	Args      map[string]any
	CounterID uint64
}

func (CounterEventRecord) fxtRecord() {}

type DurationBeginEventRecord struct {
	Timestamp uint64
	Category  string
	Name      string
	Thread    Thread
	Args      map[string]any
}

func (DurationBeginEventRecord) fxtRecord() {}

type DurationEndEventRecord struct {
	Timestamp uint64
	Category  string
	Name      string
	Thread    Thread
	Args      map[string]any
}

func (DurationEndEventRecord) fxtRecord() {}

type DurationCompleteEventRecord struct {
	Timestamp     uint64
	Category      string
	Name          string
	Thread        Thread
	Args          map[string]any
	NumberOfTicks uint64
}

func (DurationCompleteEventRecord) fxtRecord() {}

type AsyncBeginEventRecord struct {
	Timestamp     uint64
	Category      string
	Name          string
	Thread        Thread
	Args          map[string]any
	CorrelationID uint64
}

func (AsyncBeginEventRecord) fxtRecord() {}

type AsyncInstantEventRecord struct {
	Timestamp     uint64
	Category      string
	Name          string
	Thread        Thread
	Args          map[string]any
	CorrelationID uint64
}

func (AsyncInstantEventRecord) fxtRecord() {}

type AsyncEndEventRecord struct {
	Timestamp     uint64
	Category      string
	Name          string
	Thread        Thread
	Args          map[string]any
	CorrelationID uint64
}

func (AsyncEndEventRecord) fxtRecord() {}

type FlowBeginEvent struct {
	Timestamp     uint64
	Category      string
	Name          string
	Thread        Thread
	Args          map[string]any
	CorrelationID uint64
}

func (FlowBeginEvent) fxtRecord() {}

type FlowStepEvent struct {
	Timestamp     uint64
	Category      string
	Name          string
	Thread        Thread
	Args          map[string]any
	CorrelationID uint64
}

func (FlowStepEvent) fxtRecord() {}

type FlowEndEvent struct {
	Timestamp     uint64
	Category      string
	Name          string
	Thread        Thread
	Args          map[string]any
	CorrelationID uint64
}

func (FlowEndEvent) fxtRecord() {}

type BlobRecord struct {
	Name    string
	Type    BlobType
	Payload []byte
}

func (BlobRecord) fxtRecord() {}

type UserspaceObjectRecord struct {
	Name      string
	ProcessID KernelObjectID
	Pointer   uintptr
	Args      map[string]any
}

func (UserspaceObjectRecord) fxtRecord() {}

type KernelObjectRecord struct {
	Type KernelObjectType
	ID   KernelObjectID
	Name string
	Args map[string]any
}

func (KernelObjectRecord) fxtRecord() {}

type ContextSwitchRecord struct {
	Timestamp           uint64
	CPUID               uint16
	OutgoingThreadID    KernelObjectID
	OutgoingThreadState ThreadStateType
	IncomingThreadID    KernelObjectID
	Args                map[string]any
}

func (ContextSwitchRecord) fxtRecord() {}

type ThreadWakeupRecord struct {
	Timestamp      uint64
	CPUID          uint16
	WakingThreadID KernelObjectID
	Args           map[string]any
}

func (ThreadWakeupRecord) fxtRecord() {}

type LogRecord struct {
	Timestamp uint64
	Thread    Thread
	Message   string
}

func (LogRecord) fxtRecord() {}

type LargeBlobWithMetadataRecord struct {
	Timestamp uint64
	Category  string
	Name      string
	Thread    Thread
	Args      map[string]any
	Payload   []byte
}

func (LargeBlobWithMetadataRecord) fxtRecord() {}

type LargeBlobNoMetadataRecord struct {
	Category string
	Name     string
	Payload  []byte
}

func (LargeBlobNoMetadataRecord) fxtRecord() {}
