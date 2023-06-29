package fxt

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os"
)

// KernelObjectID is a unique identifier for a kernel object
// for example, a process or thread
type KernelObjectID uint64

// Thread uniquely identifies a thread within a process
type Thread struct {
	ProcessId KernelObjectID
	ThreadId  KernelObjectID
}

// NewWriter creates a new FXT file at `filePath` and initializes it with the FXT header
// It returns a Writer instance which can be used to add records to the file
func NewWriter(filePath string) (*Writer, error) {
	file, err := os.Create(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open dest file %s - %w", filePath, err)
	}

	writer := &Writer{
		file:            file,
		stringTable:     map[string]uint16{},
		nextStringIndex: 1,
		threadTable:     map[Thread]uint16{},
		nextThreadIndex: 1,
	}

	if err := writer.writeMagicNumberRecord(); err != nil {
		return nil, err
	}

	return writer, nil
}

// Writer is a struct for writing an FXT file. It has methods for adding records to the file
type Writer struct {
	file *os.File

	stringTable     map[string]uint16
	nextStringIndex uint16
	threadTable     map[Thread]uint16
	nextThreadIndex uint16
}

// Close closes the underlying file
func (w *Writer) Close() error {
	return w.file.Close()
}

func (w *Writer) writeMagicNumberRecord() error {
	if _, err := w.file.Write(fxtMagic); err != nil {
		return fmt.Errorf("failed to write magic number record - %w", err)
	}
	return nil
}

// AddProviderInfoRecord adds a provider info metadata record to the file
//
// https://fuchsia.googlesource.com/fuchsia/+/refs/heads/main/docs/reference/tracing/trace-format.md#provider-info-metadata
func (w *Writer) AddProviderInfoRecord(providerId uint32, providerName string) error {
	nameBytes := []byte(providerName)
	nameLen := len(nameBytes)
	if nameLen > math.MaxUint8 {
		return fmt.Errorf("provider name is too long")
	}

	paddedNameLen := (nameLen + 8 - 1) & (-8)
	diff := paddedNameLen - nameLen

	sizeInWords := 1 + (paddedNameLen / 8)

	header := (uint64(nameLen) << 52) | (uint64(providerId) << 20) | (uint64(metadataTypeProviderInfo) << 16) | (uint64(sizeInWords) << 4) | uint64(recordTypeMetadata)
	if err := binary.Write(w.file, binary.LittleEndian, header); err != nil {
		return fmt.Errorf("failed to write record header - %w", err)
	}

	if _, err := w.file.Write(nameBytes); err != nil {
		return fmt.Errorf("failed to write provider name data - %w", err)
	}
	if diff > 0 {
		buffer := make([]byte, diff)
		if _, err := w.file.Write(buffer); err != nil {
			return fmt.Errorf("failed to write provider name padding - %w", err)
		}
	}

	n, err := w.file.Seek(0, io.SeekCurrent)
	if err != nil {
		return err
	}
	fmt.Print(n)

	return nil
}

// AddProviderSectionRecord adds a provider section metadata record to the file
//
// https://fuchsia.googlesource.com/fuchsia/+/refs/heads/main/docs/reference/tracing/trace-format.md#provider-section-metadata
func (w *Writer) AddProviderSectionRecord(providerId uint32) error {
	sizeInWords := 1
	header := (uint64(providerId) << 20) | (uint64(metadataTypeProviderSection) << 16) | (uint64(sizeInWords) << 4) | uint64(recordTypeMetadata)
	if err := binary.Write(w.file, binary.LittleEndian, header); err != nil {
		return fmt.Errorf("failed to write record header - %w", err)
	}

	return nil
}

// AddProviderEventRecord adds a provider event metadata record to the file
//
// https://fuchsia.googlesource.com/fuchsia/+/refs/heads/main/docs/reference/tracing/trace-format.md#provider-event-metadata
func (w *Writer) AddProviderEventRecord(providerId uint32, eventType providerEventType) error {
	sizeInWords := 1
	header := (uint64(eventType) << 52) | (uint64(providerId) << 20) | (uint64(metadataTypeProviderEvent) << 16) | (uint64(sizeInWords) << 4) | uint64(recordTypeMetadata)
	if err := binary.Write(w.file, binary.LittleEndian, header); err != nil {
		return fmt.Errorf("failed to write record header - %w", err)
	}

	return nil
}

// AddInitializationRecord adds an initialization record to the file
//
// https://fuchsia.googlesource.com/fuchsia/+/refs/heads/main/docs/reference/tracing/trace-format.md#initialization-record
//
// This specifies the number of ticks per second for all event records after this
func (w *Writer) AddInitializationRecord(numTicksPerSecond uint64) error {
	sizeInWords := 2
	header := (uint64(sizeInWords) << 4) | uint64(recordTypeInitialization)
	if err := binary.Write(w.file, binary.LittleEndian, header); err != nil {
		return fmt.Errorf("failed to write record header - %w", err)
	}

	if err := binary.Write(w.file, binary.LittleEndian, numTicksPerSecond); err != nil {
		return fmt.Errorf("failed to write number of ticks per second - %w", err)
	}

	return nil
}

func (w *Writer) addStringRecord(stringIndex uint16, str string) error {
	strBytes := []byte(str)
	strLen := len(strBytes)
	if strLen > math.MaxUint8 {
		return fmt.Errorf("string is too long")
	}

	paddedStrLen := (strLen + 8 - 1) & (-8)
	diff := paddedStrLen - strLen

	sizeInWords := 1 + (paddedStrLen / 8)
	header := (uint64(strLen) << 32) | (uint64(stringIndex) << 16) | (uint64(sizeInWords) << 4) | uint64(recordTypeString)
	if err := binary.Write(w.file, binary.LittleEndian, header); err != nil {
		return fmt.Errorf("failed to write record header - %w", err)
	}

	if _, err := w.file.Write(strBytes); err != nil {
		return fmt.Errorf("failed to write string data - %w", err)
	}
	if diff > 0 {
		buffer := make([]byte, diff)
		if _, err := w.file.Write(buffer); err != nil {
			return fmt.Errorf("failed to write string padding - %w", err)
		}
	}

	return nil
}

func (w *Writer) addThreadRecord(threadIndex uint16, processId KernelObjectID, threadId KernelObjectID) error {
	sizeInWords := 3
	header := (uint64(threadIndex) << 16) | (uint64(sizeInWords) << 4) | uint64(recordTypeThread)
	if err := binary.Write(w.file, binary.LittleEndian, header); err != nil {
		return fmt.Errorf("failed to write record header - %w", err)
	}

	if err := binary.Write(w.file, binary.LittleEndian, processId); err != nil {
		return fmt.Errorf("failed to write process ID - %w", err)
	}

	if err := binary.Write(w.file, binary.LittleEndian, threadId); err != nil {
		return fmt.Errorf("failed to write thread ID - %w", err)
	}

	return nil
}

func (w *Writer) getStringIndex(str string) (uint16, error) {
	index, ok := w.stringTable[str]
	if !ok {
		return 0, fmt.Errorf("`%s` does not exist in the string table", str)
	}

	return index, nil
}

func (w *Writer) getOrCreateStringIndex(str string) (uint16, error) {
	index, ok := w.stringTable[str]
	if !ok {
		index = w.nextStringIndex
		w.nextStringIndex++
		w.stringTable[str] = index
		if err := w.addStringRecord(index, str); err != nil {
			return 0, fmt.Errorf("failed to add string record for `%s` - %w", str, err)
		}
	}

	return index, nil
}

func (w *Writer) getOrCreateThreadIndex(processId KernelObjectID, threadId KernelObjectID) (uint16, error) {
	thread := Thread{ProcessId: processId, ThreadId: threadId}
	threadIndex, ok := w.threadTable[thread]
	if !ok {
		threadIndex = w.nextThreadIndex
		w.nextThreadIndex++
		w.threadTable[thread] = threadIndex
		if err := w.addThreadRecord(threadIndex, processId, threadId); err != nil {
			return 0, fmt.Errorf("failed to add thread record - %w", err)
		}
	}

	return threadIndex, nil
}

// SetProcessName adds a kernel object record to give a human-readable name to a process ID
//
// https://fuchsia.googlesource.com/fuchsia/+/refs/heads/main/docs/reference/tracing/trace-format.md#kernel-object-record
func (w *Writer) SetProcessName(processId KernelObjectID, name string) error {
	nameIndex, err := w.getOrCreateStringIndex(name)
	if err != nil {
		return err
	}

	sizeInWords := /* header */ 1 + /* processID */ 1
	numArgs := 0
	header := (uint64(numArgs) << 40) | (uint64(nameIndex) << 24) | (uint64(koidTypeProcess) << 16) | (uint64(sizeInWords) << 4) | uint64(recordTypeKernelObject)
	if err := binary.Write(w.file, binary.LittleEndian, header); err != nil {
		return fmt.Errorf("failed to write record header - %w", err)
	}

	if err := binary.Write(w.file, binary.LittleEndian, processId); err != nil {
		return fmt.Errorf("failed to write process ID - %w", err)
	}

	return nil
}

// SetThreadName adds a kernel object record
//
// https://fuchsia.googlesource.com/fuchsia/+/refs/heads/main/docs/reference/tracing/trace-format.md#kernel-object-record
func (w *Writer) SetThreadName(processId KernelObjectID, threadId KernelObjectID, name string) error {
	nameIndex, err := w.getOrCreateStringIndex(name)
	if err != nil {
		return err
	}

	processIndex, err := w.getOrCreateStringIndex("process")
	if err != nil {
		return err
	}

	argumentSizeInWords := 2

	sizeInWords := /* header */ 1 + /* threadID */ 1 + /* argument data */ argumentSizeInWords
	numArgs := 1
	header := (uint64(numArgs) << 40) | (uint64(nameIndex) << 24) | (uint64(koidTypeThread) << 16) | (uint64(sizeInWords) << 4) | uint64(recordTypeKernelObject)
	if err := binary.Write(w.file, binary.LittleEndian, header); err != nil {
		return fmt.Errorf("failed to write record header - %w", err)
	}

	if err := binary.Write(w.file, binary.LittleEndian, threadId); err != nil {
		return fmt.Errorf("failed to write thread ID - %w", err)
	}

	// Write KIOD Argument to reference the process ID
	argHeader := (uint64(processIndex) << 16) | (uint64(argumentSizeInWords) << 4) | uint64(argumentTypeKOID)
	if err := binary.Write(w.file, binary.LittleEndian, argHeader); err != nil {
		return fmt.Errorf("failed to write argument header - %w", err)
	}

	if err := binary.Write(w.file, binary.LittleEndian, processId); err != nil {
		return fmt.Errorf("failed to write process ID - %w", err)
	}

	return nil
}

// writeEventHeaderAndGenericData is a helper function for all event record methods
// All events share the same basic header and initial data sections
//
// https://fuchsia.googlesource.com/fuchsia/+/refs/heads/main/docs/reference/tracing/trace-format.md#event-record
//
// This function writes the header and the common data
func (w *Writer) writeEventHeaderAndGenericData(eventType eventType, category string, name string, processId KernelObjectID, threadId KernelObjectID, timestamp uint64, arguments map[string]interface{}, extraSizeInWords int) error {
	categoryIndex, err := w.getOrCreateStringIndex(category)
	if err != nil {
		return err
	}

	nameIndex, err := w.getOrCreateStringIndex(name)
	if err != nil {
		return err
	}

	threadIndex, err := w.getOrCreateThreadIndex(processId, threadId)
	if err != nil {
		return err
	}

	// Add up the argument word size
	// And ensure the argument keys (and string values) are in the string table
	argumentSizeInWords := 0
	for key, value := range arguments {
		size, err := getArgumentSizeInWords(value)
		if err != nil {
			return err
		}
		argumentSizeInWords += size

		if err := w.addArgumentStringsToTable(key, value); err != nil {
			return err
		}
	}

	sizeInWords := /* Header */ 1 + /* timestamp */ 1 + /* argument data */ argumentSizeInWords + /* extra stuff */ extraSizeInWords
	numArgs := len(arguments)
	header := (uint64(nameIndex) << 48) | (uint64(categoryIndex) << 32) | (uint64(threadIndex) << 24) | (uint64(numArgs) << 20) | (uint64(eventType) << 16) | (uint64(sizeInWords) << 4) | uint64(recordTypeEvent)
	if err := binary.Write(w.file, binary.LittleEndian, header); err != nil {
		return fmt.Errorf("failed to write record header - %w", err)
	}

	if err := binary.Write(w.file, binary.LittleEndian, timestamp); err != nil {
		return fmt.Errorf("failed to write timestamp - %w", err)
	}

	wordsWritten := 0
	for key, value := range arguments {
		size, err := w.writeArgument(key, value)
		if err != nil {
			return err
		}
		wordsWritten += size
	}
	if wordsWritten != argumentSizeInWords {
		return fmt.Errorf("Expected to write %d words of argument data, but actually wrote %d", argumentSizeInWords, wordsWritten)
	}

	return nil
}

func getArgumentSizeInWords(value interface{}) (int, error) {
	if value == nil {
		return 1, nil
	}

	switch value.(type) {
	case int32:
		return 1, nil
	case uint32:
		return 1, nil
	case int64:
		return 2, nil
	case uint64:
		return 2, nil
	case float64:
		return 2, nil
	case string:
		return 1, nil
	case uintptr:
		return 2, nil
	case KernelObjectID:
		return 2, nil
	case bool:
		return 1, nil
	default:
		return 0, fmt.Errorf("invalid value type `%v` for argument", value)
	}
}

func (w *Writer) addArgumentStringsToTable(key string, value interface{}) error {
	_, err := w.getOrCreateStringIndex(key)
	if err != nil {
		return err
	}

	if v, ok := value.(string); ok {
		_, err := w.getOrCreateStringIndex(v)
		if err != nil {
			return err
		}
	}

	return nil
}

// writeArgument will write out a single argument data record
//
// https://fuchsia.googlesource.com/fuchsia/+/refs/heads/main/docs/reference/tracing/trace-format.md#argument-types
func (w *Writer) writeArgument(key string, value interface{}) (numWordsWritten int, err error) {
	keyIndex, err := w.getStringIndex(key)
	if err != nil {
		return 0, err
	}

	// Check for nil. That will create a null argument
	if value == nil {
		sizeInWords := 1
		header := (uint64(keyIndex) << 16) | (uint64(sizeInWords) << 4) | uint64(argumentTypeNull)
		if err := binary.Write(w.file, binary.LittleEndian, header); err != nil {
			return 0, fmt.Errorf("failed to write argument header - %w", err)
		}

		return sizeInWords, nil
	}

	switch v := value.(type) {
	case int32:
		sizeInWords := 1
		header := (uint64(v) << 32) | (uint64(keyIndex) << 16) | (uint64(sizeInWords) << 4) | uint64(argumentTypeInt32)
		if err := binary.Write(w.file, binary.LittleEndian, header); err != nil {
			return 0, fmt.Errorf("failed to write argument header - %w", err)
		}

		return sizeInWords, nil
	case uint32:
		sizeInWords := 1
		header := (uint64(v) << 32) | (uint64(keyIndex) << 16) | (uint64(sizeInWords) << 4) | uint64(argumentTypeUInt32)
		if err := binary.Write(w.file, binary.LittleEndian, header); err != nil {
			return 0, fmt.Errorf("failed to write argument header - %w", err)
		}

		return sizeInWords, nil
	case int64:
		sizeInWords := 2
		header := (uint64(keyIndex) << 16) | (uint64(sizeInWords) << 4) | uint64(argumentTypeInt64)
		if err := binary.Write(w.file, binary.LittleEndian, header); err != nil {
			return 0, fmt.Errorf("failed to write argument header - %w", err)
		}

		if err := binary.Write(w.file, binary.LittleEndian, v); err != nil {
			return 0, fmt.Errorf("failed to write argument value - %w", err)
		}

		return sizeInWords, nil
	case uint64:
		sizeInWords := 2
		header := (uint64(keyIndex) << 16) | (uint64(sizeInWords) << 4) | uint64(argumentTypeUInt64)
		if err := binary.Write(w.file, binary.LittleEndian, header); err != nil {
			return 0, fmt.Errorf("failed to write argument header - %w", err)
		}

		if err := binary.Write(w.file, binary.LittleEndian, v); err != nil {
			return 0, fmt.Errorf("failed to write argument value - %w", err)
		}

		return sizeInWords, nil
	case float64:
		sizeInWords := 2
		header := (uint64(keyIndex) << 16) | (uint64(sizeInWords) << 4) | uint64(argumentTypeDouble)
		if err := binary.Write(w.file, binary.LittleEndian, header); err != nil {
			return 0, fmt.Errorf("failed to write argument header - %w", err)
		}

		if err := binary.Write(w.file, binary.LittleEndian, v); err != nil {
			return 0, fmt.Errorf("failed to write argument value - %w", err)
		}

		return sizeInWords, nil
	case string:
		valueIndex, err := w.getStringIndex(v)
		if err != nil {
			return 0, err
		}

		sizeInWords := 1
		header := (uint64(valueIndex) << 32) | (uint64(keyIndex) << 16) | (uint64(sizeInWords) << 4) | uint64(argumentTypeString)
		if err := binary.Write(w.file, binary.LittleEndian, header); err != nil {
			return 0, fmt.Errorf("failed to write argument header - %w", err)
		}

		return sizeInWords, nil
	case uintptr:
		sizeInWords := 2
		header := (uint64(keyIndex) << 16) | (uint64(sizeInWords) << 4) | uint64(argumentTypePointer)
		if err := binary.Write(w.file, binary.LittleEndian, header); err != nil {
			return 0, fmt.Errorf("failed to write argument header - %w", err)
		}

		if err := binary.Write(w.file, binary.LittleEndian, uint64(v)); err != nil {
			return 0, fmt.Errorf("failed to write argument value - %w", err)
		}

		return sizeInWords, nil
	case KernelObjectID:
		sizeInWords := 2
		header := (uint64(keyIndex) << 16) | (uint64(sizeInWords) << 4) | uint64(argumentTypeKOID)
		if err := binary.Write(w.file, binary.LittleEndian, header); err != nil {
			return 0, fmt.Errorf("failed to write argument header - %w", err)
		}

		if err := binary.Write(w.file, binary.LittleEndian, v); err != nil {
			return 0, fmt.Errorf("failed to write argument value - %w", err)
		}

		return sizeInWords, nil
	case bool:
		valueBit := 0
		if v {
			valueBit = 1
		}

		sizeInWords := 1
		header := (uint64(valueBit) << 32) | (uint64(keyIndex) << 16) | (uint64(sizeInWords) << 4) | uint64(argumentTypeBool)
		if err := binary.Write(w.file, binary.LittleEndian, header); err != nil {
			return 0, fmt.Errorf("failed to write argument header - %w", err)
		}

		return sizeInWords, nil
	default:
		return 0, fmt.Errorf("invalid value type `%v` for argument `%s`", value, key)
	}
}

// AddInstantEvent adds an instant event record to the file
//
// https://fuchsia.googlesource.com/fuchsia/+/refs/heads/main/docs/reference/tracing/trace-format.md#instant-event
//
// If strings and/or process/thread IDs aren't already in the string / thread tables respectively,
// string and thread records will be automatically created. Any future events will use the table
// references.
//
// https://fuchsia.googlesource.com/fuchsia/+/refs/heads/main/docs/reference/tracing/trace-format.md#string-record
// https://fuchsia.googlesource.com/fuchsia/+/refs/heads/main/docs/reference/tracing/trace-format.md#thread-record
func (w *Writer) AddInstantEvent(category string, name string, processId KernelObjectID, threadId KernelObjectID, timestamp uint64) error {
	return w.AddInstantEventWithArgs(category, name, processId, threadId, timestamp, map[string]interface{}{})
}

// AddInstantEventWithArgs is the same as AddInstantEvent, but it allows you to additionally include
// arguments within the event record
func (w *Writer) AddInstantEventWithArgs(category string, name string, processId KernelObjectID, threadId KernelObjectID, timestamp uint64, arguments map[string]interface{}) error {
	extraSizeInWords := 0
	if err := w.writeEventHeaderAndGenericData(eventTypeInstant, category, name, processId, threadId, timestamp, arguments, extraSizeInWords); err != nil {
		return err
	}

	return nil
}

// AddCounterEvent adds a counter event record to the file
//
// https://fuchsia.googlesource.com/fuchsia/+/refs/heads/main/docs/reference/tracing/trace-format.md#counter-event
//
// If strings and/or process/thread IDs aren't already in the string / thread tables respectively,
// string and thread records will be automatically created. Any future events will use the table
// references.
//
// https://fuchsia.googlesource.com/fuchsia/+/refs/heads/main/docs/reference/tracing/trace-format.md#string-record
// https://fuchsia.googlesource.com/fuchsia/+/refs/heads/main/docs/reference/tracing/trace-format.md#thread-record
func (w *Writer) AddCounterEvent(category string, name string, processId KernelObjectID, threadId KernelObjectID, timestamp uint64, arguments map[string]interface{}, counterId uint64) error {
	extraSizeInWords := 1
	if err := w.writeEventHeaderAndGenericData(eventTypeCounter, category, name, processId, threadId, timestamp, arguments, extraSizeInWords); err != nil {
		return err
	}

	if err := binary.Write(w.file, binary.LittleEndian, counterId); err != nil {
		return fmt.Errorf("failed to write counter ID - %w", err)
	}

	return nil
}

// AddDurationBeginEvent adds a duration begin event record to the file
//
// https://fuchsia.googlesource.com/fuchsia/+/refs/heads/main/docs/reference/tracing/trace-format.md#duration-begin-event
//
// If strings and/or process/thread IDs aren't already in the string / thread tables respectively,
// string and thread records will be automatically created. Any future events will use the table
// references.
//
// https://fuchsia.googlesource.com/fuchsia/+/refs/heads/main/docs/reference/tracing/trace-format.md#string-record
// https://fuchsia.googlesource.com/fuchsia/+/refs/heads/main/docs/reference/tracing/trace-format.md#thread-record
func (w *Writer) AddDurationBeginEvent(category string, name string, processId KernelObjectID, threadId KernelObjectID, timestamp uint64) error {
	return w.AddDurationBeginEventWithArgs(category, name, processId, threadId, timestamp, map[string]interface{}{})
}

// AddDurationBeginEventWithArgs is the same as AddDurationBeginEvent, but it allows you to additionally include
// arguments within the event record
func (w *Writer) AddDurationBeginEventWithArgs(category string, name string, processId KernelObjectID, threadId KernelObjectID, timestamp uint64, arguments map[string]interface{}) error {
	extraSizeInWords := 0
	if err := w.writeEventHeaderAndGenericData(eventTypeDurationBegin, category, name, processId, threadId, timestamp, arguments, extraSizeInWords); err != nil {
		return err
	}

	return nil
}

// AddDurationEndEvent adds a duration end event record to the file
//
// https://fuchsia.googlesource.com/fuchsia/+/refs/heads/main/docs/reference/tracing/trace-format.md#duration-end-event
//
// If strings and/or process/thread IDs aren't already in the string / thread tables respectively,
// string and thread records will be automatically created. Any future events will use the table
// references.
//
// https://fuchsia.googlesource.com/fuchsia/+/refs/heads/main/docs/reference/tracing/trace-format.md#string-record
// https://fuchsia.googlesource.com/fuchsia/+/refs/heads/main/docs/reference/tracing/trace-format.md#thread-record
func (w *Writer) AddDurationEndEvent(category string, name string, processId KernelObjectID, threadId KernelObjectID, timestamp uint64) error {
	return w.AddDurationEndEventWithArgs(category, name, processId, threadId, timestamp, map[string]interface{}{})
}

// AddDurationEndEventWithArgs is the same as AddDurationEndEvent, but it allows you to additionally include
// arguments within the event record
func (w *Writer) AddDurationEndEventWithArgs(category string, name string, processId KernelObjectID, threadId KernelObjectID, timestamp uint64, arguments map[string]interface{}) error {
	extraSizeInWords := 0
	if err := w.writeEventHeaderAndGenericData(eventTypeDurationEnd, category, name, processId, threadId, timestamp, arguments, extraSizeInWords); err != nil {
		return err
	}

	return nil
}

// AddDurationCompleteEvent adds a duration complete event record to the file
//
// https://fuchsia.googlesource.com/fuchsia/+/refs/heads/main/docs/reference/tracing/trace-format.md#duration-complete-event
//
// If strings and/or process/thread IDs aren't already in the string / thread tables respectively,
// string and thread records will be automatically created. Any future events will use the table
// references.
//
// https://fuchsia.googlesource.com/fuchsia/+/refs/heads/main/docs/reference/tracing/trace-format.md#string-record
// https://fuchsia.googlesource.com/fuchsia/+/refs/heads/main/docs/reference/tracing/trace-format.md#thread-record
func (w *Writer) AddDurationCompleteEvent(category string, name string, processId KernelObjectID, threadId KernelObjectID, beginTimestamp uint64, endTimestamp uint64) error {
	return w.AddDurationCompleteEventWithArgs(category, name, processId, threadId, beginTimestamp, endTimestamp, map[string]interface{}{})
}

// AddDurationCompleteEventWithArgs is the same as AddDurationCompleteEvent, but it allows you to additionally include
// arguments within the event record
func (w *Writer) AddDurationCompleteEventWithArgs(category string, name string, processId KernelObjectID, threadId KernelObjectID, beginTimestamp uint64, endTimestamp uint64, arguments map[string]interface{}) error {
	extraSizeInWords := 1
	if err := w.writeEventHeaderAndGenericData(eventTypeDurationComplete, category, name, processId, threadId, beginTimestamp, arguments, extraSizeInWords); err != nil {
		return err
	}

	if err := binary.Write(w.file, binary.LittleEndian, endTimestamp); err != nil {
		return fmt.Errorf("failed to write end timestamp - %w", err)
	}

	return nil
}

// AddAsyncBeginEvent adds an async begin event record to the file
//
// https://fuchsia.googlesource.com/fuchsia/+/refs/heads/main/docs/reference/tracing/trace-format.md#async-begin-event
//
// If strings and/or process/thread IDs aren't already in the string / thread tables respectively,
// string and thread records will be automatically created. Any future events will use the table
// references.
//
// https://fuchsia.googlesource.com/fuchsia/+/refs/heads/main/docs/reference/tracing/trace-format.md#string-record
// https://fuchsia.googlesource.com/fuchsia/+/refs/heads/main/docs/reference/tracing/trace-format.md#thread-record
func (w *Writer) AddAsyncBeginEvent(category string, name string, processId KernelObjectID, threadId KernelObjectID, timestamp uint64, asyncCorrelationId uint64) error {
	return w.AddAsyncBeginEventWithArgs(category, name, processId, threadId, timestamp, asyncCorrelationId, map[string]interface{}{})
}

// AddAsyncBeginEventWithArgs is the same as AddAsyncBeginEvent, but it allows you to additionally include
// arguments within the event record
func (w *Writer) AddAsyncBeginEventWithArgs(category string, name string, processId KernelObjectID, threadId KernelObjectID, timestamp uint64, asyncCorrelationId uint64, arguments map[string]interface{}) error {
	extraSizeInWords := 1
	if err := w.writeEventHeaderAndGenericData(eventTypeAsyncBegin, category, name, processId, threadId, timestamp, arguments, extraSizeInWords); err != nil {
		return err
	}

	if err := binary.Write(w.file, binary.LittleEndian, asyncCorrelationId); err != nil {
		return fmt.Errorf("failed to write async correlation ID - %w", err)
	}

	return nil
}

// AddAsyncInstantEvent adds an async instant event record to the file
//
// https://fuchsia.googlesource.com/fuchsia/+/refs/heads/main/docs/reference/tracing/trace-format.md#async-instant-event
//
// If strings and/or process/thread IDs aren't already in the string / thread tables respectively,
// string and thread records will be automatically created. Any future events will use the table
// references.
//
// https://fuchsia.googlesource.com/fuchsia/+/refs/heads/main/docs/reference/tracing/trace-format.md#string-record
// https://fuchsia.googlesource.com/fuchsia/+/refs/heads/main/docs/reference/tracing/trace-format.md#thread-record
func (w *Writer) AddAsyncInstantEvent(category string, name string, processId KernelObjectID, threadId KernelObjectID, timestamp uint64, asyncCorrelationId uint64) error {
	return w.AddAsyncInstantEventWithArgs(category, name, processId, threadId, timestamp, asyncCorrelationId, map[string]interface{}{})
}

// AddAsyncInstantEventWithArgs is the same as AddAsyncInstantEvent, but it allows you to additionally include
// arguments within the event record
func (w *Writer) AddAsyncInstantEventWithArgs(category string, name string, processId KernelObjectID, threadId KernelObjectID, timestamp uint64, asyncCorrelationId uint64, arguments map[string]interface{}) error {
	extraSizeInWords := 1
	if err := w.writeEventHeaderAndGenericData(eventTypeAsyncInstant, category, name, processId, threadId, timestamp, arguments, extraSizeInWords); err != nil {
		return err
	}

	if err := binary.Write(w.file, binary.LittleEndian, asyncCorrelationId); err != nil {
		return fmt.Errorf("failed to write async correlation ID - %w", err)
	}

	return nil
}

// AddAsyncEndEvent adds an async end event record to the file
//
// https://fuchsia.googlesource.com/fuchsia/+/refs/heads/main/docs/reference/tracing/trace-format.md#async-end-event
//
// If strings and/or process/thread IDs aren't already in the string / thread tables respectively,
// string and thread records will be automatically created. Any future events will use the table
// references.
//
// https://fuchsia.googlesource.com/fuchsia/+/refs/heads/main/docs/reference/tracing/trace-format.md#string-record
// https://fuchsia.googlesource.com/fuchsia/+/refs/heads/main/docs/reference/tracing/trace-format.md#thread-record
func (w *Writer) AddAsyncEndEvent(category string, name string, processId KernelObjectID, threadId KernelObjectID, timestamp uint64, asyncCorrelationId uint64) error {
	return w.AddAsyncEndEventWithArgs(category, name, processId, threadId, timestamp, asyncCorrelationId, map[string]interface{}{})
}

// AddAsyncEndEventWithArgs is the same as AddAsyncEndEvent, but it allows you to additionally include
// arguments within the event record
func (w *Writer) AddAsyncEndEventWithArgs(category string, name string, processId KernelObjectID, threadId KernelObjectID, timestamp uint64, asyncCorrelationId uint64, arguments map[string]interface{}) error {
	extraSizeInWords := 1
	if err := w.writeEventHeaderAndGenericData(eventTypeAsyncEnd, category, name, processId, threadId, timestamp, arguments, extraSizeInWords); err != nil {
		return err
	}

	if err := binary.Write(w.file, binary.LittleEndian, asyncCorrelationId); err != nil {
		return fmt.Errorf("failed to write async correlation ID - %w", err)
	}

	return nil
}

// AddFlowBeginEvent adds an flow begin event record to the file
//
// https://fuchsia.googlesource.com/fuchsia/+/refs/heads/main/docs/reference/tracing/trace-format.md#flow-begin-event
//
// If strings and/or process/thread IDs aren't already in the string / thread tables respectively,
// string and thread records will be automatically created. Any future events will use the table
// references.
//
// https://fuchsia.googlesource.com/fuchsia/+/refs/heads/main/docs/reference/tracing/trace-format.md#string-record
// https://fuchsia.googlesource.com/fuchsia/+/refs/heads/main/docs/reference/tracing/trace-format.md#thread-record
func (w *Writer) AddFlowBeginEvent(category string, name string, processId KernelObjectID, threadId KernelObjectID, timestamp uint64, flowCorrelationId uint64) error {
	return w.AddFlowBeginEventWithArgs(category, name, processId, threadId, timestamp, flowCorrelationId, map[string]interface{}{})
}

// AddFlowBeginEventWithArgs is the same as AddFlowBeginEvent, but it allows you to additionally include
// arguments within the event record
func (w *Writer) AddFlowBeginEventWithArgs(category string, name string, processId KernelObjectID, threadId KernelObjectID, timestamp uint64, flowCorrelationId uint64, arguments map[string]interface{}) error {
	extraSizeInWords := 1
	if err := w.writeEventHeaderAndGenericData(eventTypeFlowBegin, category, name, processId, threadId, timestamp, arguments, extraSizeInWords); err != nil {
		return err
	}

	if err := binary.Write(w.file, binary.LittleEndian, flowCorrelationId); err != nil {
		return fmt.Errorf("failed to write async correlation ID - %w", err)
	}

	return nil
}

// AddFlowStepEvent adds an flow step event record to the file
//
// https://fuchsia.googlesource.com/fuchsia/+/refs/heads/main/docs/reference/tracing/trace-format.md#flow-step-event
//
// If strings and/or process/thread IDs aren't already in the string / thread tables respectively,
// string and thread records will be automatically created. Any future events will use the table
// references.
//
// https://fuchsia.googlesource.com/fuchsia/+/refs/heads/main/docs/reference/tracing/trace-format.md#string-record
// https://fuchsia.googlesource.com/fuchsia/+/refs/heads/main/docs/reference/tracing/trace-format.md#thread-record
func (w *Writer) AddFlowStepEvent(category string, name string, processId KernelObjectID, threadId KernelObjectID, timestamp uint64, flowCorrelationId uint64) error {
	return w.AddFlowStepEventWithArgs(category, name, processId, threadId, timestamp, flowCorrelationId, map[string]interface{}{})
}

// AddFlowStepEventWithArgs is the same as AddFlowStepEvent, but it allows you to additionally include
// arguments within the event record
func (w *Writer) AddFlowStepEventWithArgs(category string, name string, processId KernelObjectID, threadId KernelObjectID, timestamp uint64, flowCorrelationId uint64, arguments map[string]interface{}) error {
	extraSizeInWords := 1
	if err := w.writeEventHeaderAndGenericData(eventTypeFlowStep, category, name, processId, threadId, timestamp, arguments, extraSizeInWords); err != nil {
		return err
	}

	if err := binary.Write(w.file, binary.LittleEndian, flowCorrelationId); err != nil {
		return fmt.Errorf("failed to write async correlation ID - %w", err)
	}

	return nil
}

// AddFlowEndEvent adds an flow end event record to the file
//
// https://fuchsia.googlesource.com/fuchsia/+/refs/heads/main/docs/reference/tracing/trace-format.md#flow-end-event
//
// If strings and/or process/thread IDs aren't already in the string / thread tables respectively,
// string and thread records will be automatically created. Any future events will use the table
// references.
//
// https://fuchsia.googlesource.com/fuchsia/+/refs/heads/main/docs/reference/tracing/trace-format.md#string-record
// https://fuchsia.googlesource.com/fuchsia/+/refs/heads/main/docs/reference/tracing/trace-format.md#thread-record
func (w *Writer) AddFlowEndEvent(category string, name string, processId KernelObjectID, threadId KernelObjectID, timestamp uint64, flowCorrelationId uint64) error {
	return w.AddFlowEndEventWithArgs(category, name, processId, threadId, timestamp, flowCorrelationId, map[string]interface{}{})
}

// AddFlowEndEventWithArgs is the same as AddFlowEndEvent, but it allows you to additionally include
// arguments within the event record
func (w *Writer) AddFlowEndEventWithArgs(category string, name string, processId KernelObjectID, threadId KernelObjectID, timestamp uint64, flowCorrelationId uint64, arguments map[string]interface{}) error {
	extraSizeInWords := 1
	if err := w.writeEventHeaderAndGenericData(eventTypeFlowEnd, category, name, processId, threadId, timestamp, arguments, extraSizeInWords); err != nil {
		return err
	}

	if err := binary.Write(w.file, binary.LittleEndian, flowCorrelationId); err != nil {
		return fmt.Errorf("failed to write async correlation ID - %w", err)
	}

	return nil
}

// AddBlobRecord adds a blob record to the file
//
// https://fuchsia.googlesource.com/fuchsia/+/refs/heads/main/docs/reference/tracing/trace-format.md#blob-record
func (w *Writer) AddBlobRecord(name string, data []byte, blobType BlobType) error {
	nameIndex, err := w.getOrCreateStringIndex(name)
	if err != nil {
		return err
	}

	blobSize := len(data)
	paddedSize := (blobSize + 8 - 1) & (-8)
	diff := paddedSize - blobSize

	sizeInWords := 1 + (paddedSize / 8)
	header := (uint64(blobType) << 48) | (uint64(blobSize) << 32) | (uint64(nameIndex) << 16) | (uint64(sizeInWords) << 4) | uint64(recordTypeBlob)
	if err := binary.Write(w.file, binary.LittleEndian, header); err != nil {
		return fmt.Errorf("failed to write record header - %w", err)
	}

	if _, err := w.file.Write(data); err != nil {
		return fmt.Errorf("failed to write blob data - %w", err)
	}

	if diff > 0 {
		buffer := make([]byte, diff)
		if _, err := w.file.Write(buffer); err != nil {
			return fmt.Errorf("failed to write blob data padding - %w", err)
		}
	}

	return nil
}

// AddUserspaceObjectRecord adds a userspace object record to the file
//
// https://fuchsia.googlesource.com/fuchsia/+/refs/heads/main/docs/reference/tracing/trace-format.md#userspace-object-record
func (w *Writer) AddUserspaceObjectRecord(name string, processId KernelObjectID, pointerValue uintptr, arguments map[string]interface{}) error {
	nameIndex, err := w.getOrCreateStringIndex(name)
	if err != nil {
		return err
	}

	// Add up the argument word size
	// And ensure the argument keys (and string values) are in the string table
	argumentSizeInWords := 0
	for key, value := range arguments {
		size, err := getArgumentSizeInWords(value)
		if err != nil {
			return err
		}
		argumentSizeInWords += size

		if err := w.addArgumentStringsToTable(key, value); err != nil {
			return err
		}
	}

	sizeInWords := /* Header */ 1 + /* pointer value */ 1 + /* process ID */ 1 + /* argument data */ argumentSizeInWords
	threadIndex := 0
	numArgs := len(arguments)
	header := (uint64(numArgs) << 40) | (uint64(nameIndex) << 24) | (uint64(threadIndex) << 16) | (uint64(sizeInWords) << 4) | uint64(recordTypeUserspaceObject)
	if err := binary.Write(w.file, binary.LittleEndian, header); err != nil {
		return fmt.Errorf("failed to write record header - %w", err)
	}

	if err := binary.Write(w.file, binary.LittleEndian, uint64(pointerValue)); err != nil {
		return fmt.Errorf("failed to write pointer value - %w", err)
	}

	if err := binary.Write(w.file, binary.LittleEndian, processId); err != nil {
		return fmt.Errorf("failed to write process ID - %w", err)
	}

	wordsWritten := 0
	for key, value := range arguments {
		size, err := w.writeArgument(key, value)
		if err != nil {
			return err
		}
		wordsWritten += size
	}
	if wordsWritten != argumentSizeInWords {
		return fmt.Errorf("Expected to write %d words of argument data, but actually wrote %d", argumentSizeInWords, wordsWritten)
	}

	return nil
}

// AddContextSwitchRecord adds a context switch scheduling record to the file
//
// https://fuchsia.googlesource.com/fuchsia/+/refs/heads/main/docs/reference/tracing/trace-format.md#context-switch-record-scheduling-event-record-type-1
func (w *Writer) AddContextSwitchRecord(cpuNumber uint16, outgoingThreadState uint8, outgoingThreadId KernelObjectID, incomingThreadId KernelObjectID, timestamp uint64) error {
	return w.AddContextSwitchRecordWithArgs(cpuNumber, outgoingThreadState, outgoingThreadId, incomingThreadId, timestamp, map[string]interface{}{})
}

// AddContextSwitchRecordWithArgs is the same as AddContextSwitchRecord, but it allows you to additionally include
// arguments within the scheduling record
func (w *Writer) AddContextSwitchRecordWithArgs(cpuNumber uint16, outgoingThreadState uint8, outgoingThreadId KernelObjectID, incomingThreadId KernelObjectID, timestamp uint64, arguments map[string]interface{}) error {
	// Sanity check
	// Ideally we'd find out the actual ENUM of valid states
	if outgoingThreadState > 0xF {
		return fmt.Errorf("invalid outgoingThreadState - %d is too large", outgoingThreadState)
	}

	// Add up the argument word size
	// And ensure the argument keys (and string values) are in the string table
	argumentSizeInWords := 0
	for key, value := range arguments {
		size, err := getArgumentSizeInWords(value)
		if err != nil {
			return err
		}
		argumentSizeInWords += size

		if err := w.addArgumentStringsToTable(key, value); err != nil {
			return err
		}
	}

	sizeInWords := /* Header */ 1 + /* timestamp */ 1 + /* outgoing thread ID */ 1 + /* incoming thread ID */ 1 + /* argument data */ argumentSizeInWords
	numArgs := len(arguments)
	header := (uint64(schedulingRecordTypeContextSwitch) << 60) | (uint64(outgoingThreadState) << 36) | (uint64(cpuNumber) << 20) | (uint64(numArgs) << 16) | (uint64(sizeInWords) << 4) | uint64(recordTypeScheduling)
	if err := binary.Write(w.file, binary.LittleEndian, header); err != nil {
		return fmt.Errorf("failed to write record header - %w", err)
	}

	if err := binary.Write(w.file, binary.LittleEndian, timestamp); err != nil {
		return fmt.Errorf("failed to write timestamp - %w", err)
	}

	if err := binary.Write(w.file, binary.LittleEndian, outgoingThreadId); err != nil {
		return fmt.Errorf("failed to write outgoing thread ID - %w", err)
	}

	if err := binary.Write(w.file, binary.LittleEndian, incomingThreadId); err != nil {
		return fmt.Errorf("failed to write incoming thread ID - %w", err)
	}

	wordsWritten := 0
	for key, value := range arguments {
		size, err := w.writeArgument(key, value)
		if err != nil {
			return err
		}
		wordsWritten += size
	}
	if wordsWritten != argumentSizeInWords {
		return fmt.Errorf("Expected to write %d words of argument data, but actually wrote %d", argumentSizeInWords, wordsWritten)
	}

	return nil
}

// AddContextSwitchRecord adds a thread wakeup scheduling record to the file
//
// https://fuchsia.googlesource.com/fuchsia/+/refs/heads/main/docs/reference/tracing/trace-format.md#thread-wakeup-record-scheduling-event-record-type-2
func (w *Writer) AddThreadWakeupRecord(cpuNumber uint16, wakingThreadId KernelObjectID, timestamp uint64) error {
	return w.AddThreadWakeupRecordWithArgs(cpuNumber, wakingThreadId, timestamp, map[string]interface{}{})
}

// AddThreadWakeupRecordWithArgs is the same as AddThreadWakeupRecord, but it allows you to additionally include
// arguments within the scheduling record
func (w *Writer) AddThreadWakeupRecordWithArgs(cpuNumber uint16, wakingThreadId KernelObjectID, timestamp uint64, arguments map[string]interface{}) error {
	// Add up the argument word size
	// And ensure the argument keys (and string values) are in the string table
	argumentSizeInWords := 0
	for key, value := range arguments {
		size, err := getArgumentSizeInWords(value)
		if err != nil {
			return err
		}
		argumentSizeInWords += size

		if err := w.addArgumentStringsToTable(key, value); err != nil {
			return err
		}
	}

	sizeInWords := /* Header */ 1 + /* timestamp */ 1 + /* waking thread ID */ 1 + /* argument data */ argumentSizeInWords
	numArgs := len(arguments)
	header := (uint64(schedulingRecordTypeThreadWakeup) << 60) | (uint64(cpuNumber) << 20) | (uint64(numArgs) << 16) | (uint64(sizeInWords) << 4) | uint64(recordTypeScheduling)
	if err := binary.Write(w.file, binary.LittleEndian, header); err != nil {
		return fmt.Errorf("failed to write record header - %w", err)
	}

	if err := binary.Write(w.file, binary.LittleEndian, timestamp); err != nil {
		return fmt.Errorf("failed to write timestamp - %w", err)
	}

	if err := binary.Write(w.file, binary.LittleEndian, wakingThreadId); err != nil {
		return fmt.Errorf("failed to write waking thread ID - %w", err)
	}

	wordsWritten := 0
	for key, value := range arguments {
		size, err := w.writeArgument(key, value)
		if err != nil {
			return err
		}
		wordsWritten += size
	}
	if wordsWritten != argumentSizeInWords {
		return fmt.Errorf("Expected to write %d words of argument data, but actually wrote %d", argumentSizeInWords, wordsWritten)
	}

	return nil
}
