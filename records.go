package fxt

// Record is a way to constrain what types can be returned in the record stream
type Record interface {
	fxtRecord()
}

type timestampedRecord interface {
	getTimestampNS() uint64
}

type InstantEventRecord struct {
	TimestampNS uint64
	Category    string
	Name        string
	Thread      Thread
	Args        map[string]any
}

func (InstantEventRecord) fxtRecord() {}
func (r InstantEventRecord) getTimestampNS() uint64 {
	return r.TimestampNS
}

type CounterEventRecord struct {
	TimestampNS uint64
	Category    string
	Name        string
	Thread      Thread
	Args        map[string]any
	CounterID   uint64
}

func (CounterEventRecord) fxtRecord() {}
func (r CounterEventRecord) getTimestampNS() uint64 {
	return r.TimestampNS
}

type DurationBeginEventRecord struct {
	TimestampNS uint64
	Category    string
	Name        string
	Thread      Thread
	Args        map[string]any
}

func (DurationBeginEventRecord) fxtRecord() {}
func (r DurationBeginEventRecord) getTimestampNS() uint64 {
	return r.TimestampNS
}

type DurationEndEventRecord struct {
	TimestampNS uint64
	Category    string
	Name        string
	Thread      Thread
	Args        map[string]any
}

func (DurationEndEventRecord) fxtRecord() {}
func (r DurationEndEventRecord) getTimestampNS() uint64 {
	return r.TimestampNS
}

type DurationCompleteEventRecord struct {
	TimestampNS   uint64
	Category      string
	Name          string
	Thread        Thread
	Args          map[string]any
	DurationNS uint64
}

func (DurationCompleteEventRecord) fxtRecord() {}
func (r DurationCompleteEventRecord) getTimestampNS() uint64 {
	return r.TimestampNS
}

type AsyncBeginEventRecord struct {
	TimestampNS   uint64
	Category      string
	Name          string
	Thread        Thread
	Args          map[string]any
	CorrelationID uint64
}

func (AsyncBeginEventRecord) fxtRecord() {}
func (r AsyncBeginEventRecord) getTimestampNS() uint64 {
	return r.TimestampNS
}

type AsyncInstantEventRecord struct {
	TimestampNS   uint64
	Category      string
	Name          string
	Thread        Thread
	Args          map[string]any
	CorrelationID uint64
}

func (AsyncInstantEventRecord) fxtRecord() {}
func (r AsyncInstantEventRecord) getTimestampNS() uint64 {
	return r.TimestampNS
}

type AsyncEndEventRecord struct {
	TimestampNS   uint64
	Category      string
	Name          string
	Thread        Thread
	Args          map[string]any
	CorrelationID uint64
}

func (AsyncEndEventRecord) fxtRecord() {}
func (r AsyncEndEventRecord) getTimestampNS() uint64 {
	return r.TimestampNS
}

type FlowBeginEvent struct {
	TimestampNS   uint64
	Category      string
	Name          string
	Thread        Thread
	Args          map[string]any
	CorrelationID uint64
}

func (FlowBeginEvent) fxtRecord() {}
func (r FlowBeginEvent) getTimestampNS() uint64 {
	return r.TimestampNS
}

type FlowStepEvent struct {
	TimestampNS   uint64
	Category      string
	Name          string
	Thread        Thread
	Args          map[string]any
	CorrelationID uint64
}

func (FlowStepEvent) fxtRecord() {}
func (r FlowStepEvent) getTimestampNS() uint64 {
	return r.TimestampNS
}

type FlowEndEvent struct {
	TimestampNS   uint64
	Category      string
	Name          string
	Thread        Thread
	Args          map[string]any
	CorrelationID uint64
}

func (FlowEndEvent) fxtRecord() {}
func (r FlowEndEvent) getTimestampNS() uint64 {
	return r.TimestampNS
}

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
	TimestampNS         uint64
	CPUID               uint16
	OutgoingThreadID    KernelObjectID
	OutgoingThreadState ThreadStateType
	IncomingThreadID    KernelObjectID
	Args                map[string]any
}

func (ContextSwitchRecord) fxtRecord() {}
func (r ContextSwitchRecord) getTimestampNS() uint64 {
	return r.TimestampNS
}

type ThreadWakeupRecord struct {
	TimestampNS    uint64
	CPUID          uint16
	WakingThreadID KernelObjectID
	Args           map[string]any
}

func (ThreadWakeupRecord) fxtRecord() {}
func (r ThreadWakeupRecord) getTimestampNS() uint64 {
	return r.TimestampNS
}

type LogRecord struct {
	TimestampNS uint64
	Thread      Thread
	Message     string
}

func (LogRecord) fxtRecord() {}
func (r LogRecord) getTimestampNS() uint64 {
	return r.TimestampNS
}

type LargeBlobWithMetadataRecord struct {
	TimestampNS uint64
	Category    string
	Name        string
	Thread      Thread
	Args        map[string]any
	Payload     []byte
}

func (LargeBlobWithMetadataRecord) fxtRecord() {}
func (r LargeBlobWithMetadataRecord) getTimestampNS() uint64 {
	return r.TimestampNS
}

type LargeBlobNoMetadataRecord struct {
	Category string
	Name     string
	Payload  []byte
}

func (LargeBlobNoMetadataRecord) fxtRecord() {}
