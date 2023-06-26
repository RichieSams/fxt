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
