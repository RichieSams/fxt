package fxt

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os"
)

type KernelObjectID uint64

type Thread struct {
	ProcessId KernelObjectID
	ThreadId  KernelObjectID
}

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

type Writer struct {
	file *os.File

	stringTable     map[string]uint16
	nextStringIndex uint16
	threadTable     map[Thread]uint16
	nextThreadIndex uint16
}

func (w *Writer) Close() error {
	return w.file.Close()
}

func (w *Writer) writeMagicNumberRecord() error {
	if _, err := w.file.Write(fxtMagic); err != nil {
		return fmt.Errorf("failed to write magic number record - %w", err)
	}
	return nil
}

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

func (w *Writer) AddProviderSectionRecord(providerId uint32) error {
	sizeInWords := 1
	header := (uint64(providerId) << 20) | (uint64(metadataTypeProviderSection) << 16) | (uint64(sizeInWords) << 4) | uint64(recordTypeMetadata)
	if err := binary.Write(w.file, binary.LittleEndian, header); err != nil {
		return fmt.Errorf("failed to write record header - %w", err)
	}

	return nil
}

func (w *Writer) AddProviderEventRecord(providerId uint32, eventType providerEventType) error {
	sizeInWords := 1
	header := (uint64(eventType) << 52) | (uint64(providerId) << 20) | (uint64(metadataTypeProviderEvent) << 16) | (uint64(sizeInWords) << 4) | uint64(recordTypeMetadata)
	if err := binary.Write(w.file, binary.LittleEndian, header); err != nil {
		return fmt.Errorf("failed to write record header - %w", err)
	}

	return nil
}

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

func (w *Writer) AddInstantEvent(category string, name string, processId KernelObjectID, threadId KernelObjectID, timestamp uint64) error {
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

	sizeInWords := 2
	numArgs := 0
	header := (uint64(nameIndex) << 48) | (uint64(categoryIndex) << 32) | (uint64(threadIndex) << 24) | (uint64(numArgs) << 20) | (uint64(eventTypeInstant) << 16) | (uint64(sizeInWords) << 4) | uint64(recordTypeEvent)
	if err := binary.Write(w.file, binary.LittleEndian, header); err != nil {
		return fmt.Errorf("failed to write record header - %w", err)
	}

	if err := binary.Write(w.file, binary.LittleEndian, timestamp); err != nil {
		return fmt.Errorf("failed to write timestamp - %w", err)
	}

	return nil
}

func (w *Writer) AddDurationBeginEvent(category string, name string, processId KernelObjectID, threadId KernelObjectID, timestamp uint64) error {
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

	sizeInWords := 2
	numArgs := 0
	header := (uint64(nameIndex) << 48) | (uint64(categoryIndex) << 32) | (uint64(threadIndex) << 24) | (uint64(numArgs) << 20) | (uint64(eventTypeDurationBegin) << 16) | (uint64(sizeInWords) << 4) | uint64(recordTypeEvent)
	if err := binary.Write(w.file, binary.LittleEndian, header); err != nil {
		return fmt.Errorf("failed to write record header - %w", err)
	}

	if err := binary.Write(w.file, binary.LittleEndian, timestamp); err != nil {
		return fmt.Errorf("failed to write timestamp - %w", err)
	}

	return nil
}

func (w *Writer) AddDurationEndEvent(category string, name string, processId KernelObjectID, threadId KernelObjectID, timestamp uint64) error {
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

	sizeInWords := 2
	numArgs := 0
	header := (uint64(nameIndex) << 48) | (uint64(categoryIndex) << 32) | (uint64(threadIndex) << 24) | (uint64(numArgs) << 20) | (uint64(eventTypeDurationEnd) << 16) | (uint64(sizeInWords) << 4) | uint64(recordTypeEvent)
	if err := binary.Write(w.file, binary.LittleEndian, header); err != nil {
		return fmt.Errorf("failed to write record header - %w", err)
	}

	if err := binary.Write(w.file, binary.LittleEndian, timestamp); err != nil {
		return fmt.Errorf("failed to write timestamp - %w", err)
	}

	return nil
}

func (w *Writer) AddDurationCompleteEvent(category string, name string, processId KernelObjectID, threadId KernelObjectID, beginTimestamp uint64, endTimestamp uint64) error {
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

	sizeInWords := 3
	numArgs := 0
	header := (uint64(nameIndex) << 48) | (uint64(categoryIndex) << 32) | (uint64(threadIndex) << 24) | (uint64(numArgs) << 20) | (uint64(eventTypeDurationComplete) << 16) | (uint64(sizeInWords) << 4) | uint64(recordTypeEvent)
	if err := binary.Write(w.file, binary.LittleEndian, header); err != nil {
		return fmt.Errorf("failed to write record header - %w", err)
	}

	if err := binary.Write(w.file, binary.LittleEndian, beginTimestamp); err != nil {
		return fmt.Errorf("failed to write timestamp - %w", err)
	}

	if err := binary.Write(w.file, binary.LittleEndian, endTimestamp); err != nil {
		return fmt.Errorf("failed to write timestamp - %w", err)
	}

	return nil
}

func (w *Writer) AddAsyncBeginEvent(category string, name string, processId KernelObjectID, threadId KernelObjectID, timestamp uint64, asyncCorrelationId uint64) error {
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

	sizeInWords := 3
	numArgs := 0
	header := (uint64(nameIndex) << 48) | (uint64(categoryIndex) << 32) | (uint64(threadIndex) << 24) | (uint64(numArgs) << 20) | (uint64(eventTypeAsyncBegin) << 16) | (uint64(sizeInWords) << 4) | uint64(recordTypeEvent)
	if err := binary.Write(w.file, binary.LittleEndian, header); err != nil {
		return fmt.Errorf("failed to write record header - %w", err)
	}

	if err := binary.Write(w.file, binary.LittleEndian, timestamp); err != nil {
		return fmt.Errorf("failed to write timestamp - %w", err)
	}

	if err := binary.Write(w.file, binary.LittleEndian, asyncCorrelationId); err != nil {
		return fmt.Errorf("failed to write async correlation ID - %w", err)
	}

	return nil
}

func (w *Writer) AddAsyncInstantEvent(category string, name string, processId KernelObjectID, threadId KernelObjectID, timestamp uint64, asyncCorrelationId uint64) error {
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

	sizeInWords := 3
	numArgs := 0
	header := (uint64(nameIndex) << 48) | (uint64(categoryIndex) << 32) | (uint64(threadIndex) << 24) | (uint64(numArgs) << 20) | (uint64(eventTypeAsyncInstant) << 16) | (uint64(sizeInWords) << 4) | uint64(recordTypeEvent)
	if err := binary.Write(w.file, binary.LittleEndian, header); err != nil {
		return fmt.Errorf("failed to write record header - %w", err)
	}

	if err := binary.Write(w.file, binary.LittleEndian, timestamp); err != nil {
		return fmt.Errorf("failed to write timestamp - %w", err)
	}

	if err := binary.Write(w.file, binary.LittleEndian, asyncCorrelationId); err != nil {
		return fmt.Errorf("failed to write async correlation ID - %w", err)
	}

	return nil
}

func (w *Writer) AddAsyncEndEvent(category string, name string, processId KernelObjectID, threadId KernelObjectID, timestamp uint64, asyncCorrelationId uint64) error {
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

	sizeInWords := 3
	numArgs := 0
	header := (uint64(nameIndex) << 48) | (uint64(categoryIndex) << 32) | (uint64(threadIndex) << 24) | (uint64(numArgs) << 20) | (uint64(eventTypeAsyncEnd) << 16) | (uint64(sizeInWords) << 4) | uint64(recordTypeEvent)
	if err := binary.Write(w.file, binary.LittleEndian, header); err != nil {
		return fmt.Errorf("failed to write record header - %w", err)
	}

	if err := binary.Write(w.file, binary.LittleEndian, timestamp); err != nil {
		return fmt.Errorf("failed to write timestamp - %w", err)
	}

	if err := binary.Write(w.file, binary.LittleEndian, asyncCorrelationId); err != nil {
		return fmt.Errorf("failed to write async correlation ID - %w", err)
	}

	return nil
}

func (w *Writer) AddFlowBeginEvent(category string, name string, processId KernelObjectID, threadId KernelObjectID, timestamp uint64, flowCorrelationId uint64) error {
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

	sizeInWords := 3
	numArgs := 0
	header := (uint64(nameIndex) << 48) | (uint64(categoryIndex) << 32) | (uint64(threadIndex) << 24) | (uint64(numArgs) << 20) | (uint64(eventTypeFlowBegin) << 16) | (uint64(sizeInWords) << 4) | uint64(recordTypeEvent)
	if err := binary.Write(w.file, binary.LittleEndian, header); err != nil {
		return fmt.Errorf("failed to write record header - %w", err)
	}

	if err := binary.Write(w.file, binary.LittleEndian, timestamp); err != nil {
		return fmt.Errorf("failed to write timestamp - %w", err)
	}

	if err := binary.Write(w.file, binary.LittleEndian, flowCorrelationId); err != nil {
		return fmt.Errorf("failed to write flow correlation ID - %w", err)
	}

	return nil
}

func (w *Writer) AddFlowStepEvent(category string, name string, processId KernelObjectID, threadId KernelObjectID, timestamp uint64, flowCorrelationId uint64) error {
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

	sizeInWords := 3
	numArgs := 0
	header := (uint64(nameIndex) << 48) | (uint64(categoryIndex) << 32) | (uint64(threadIndex) << 24) | (uint64(numArgs) << 20) | (uint64(eventTypeFlowStep) << 16) | (uint64(sizeInWords) << 4) | uint64(recordTypeEvent)
	if err := binary.Write(w.file, binary.LittleEndian, header); err != nil {
		return fmt.Errorf("failed to write record header - %w", err)
	}

	if err := binary.Write(w.file, binary.LittleEndian, timestamp); err != nil {
		return fmt.Errorf("failed to write timestamp - %w", err)
	}

	if err := binary.Write(w.file, binary.LittleEndian, flowCorrelationId); err != nil {
		return fmt.Errorf("failed to write flow correlation ID - %w", err)
	}

	return nil
}

func (w *Writer) AddFlowEndEvent(category string, name string, processId KernelObjectID, threadId KernelObjectID, timestamp uint64, flowCorrelationId uint64) error {
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

	sizeInWords := 3
	numArgs := 0
	header := (uint64(nameIndex) << 48) | (uint64(categoryIndex) << 32) | (uint64(threadIndex) << 24) | (uint64(numArgs) << 20) | (uint64(eventTypeFlowEnd) << 16) | (uint64(sizeInWords) << 4) | uint64(recordTypeEvent)
	if err := binary.Write(w.file, binary.LittleEndian, header); err != nil {
		return fmt.Errorf("failed to write record header - %w", err)
	}

	if err := binary.Write(w.file, binary.LittleEndian, timestamp); err != nil {
		return fmt.Errorf("failed to write timestamp - %w", err)
	}

	if err := binary.Write(w.file, binary.LittleEndian, flowCorrelationId); err != nil {
		return fmt.Errorf("failed to write flow correlation ID - %w", err)
	}

	return nil
}
