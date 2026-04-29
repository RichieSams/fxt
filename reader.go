package fxt

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

// offsetReader wraps an io.Reader and tracks the current read offset
type offsetReader struct {
	reader              io.Reader
	currentRecordOffset int64
	currentOffset       int64
}

// Read implements io.Reader interface and tracks the number of bytes read
func (or *offsetReader) Read(p []byte) (n int, err error) {
	n, err = or.reader.Read(p)
	or.currentOffset += int64(n)
	return n, err
}

func (or *offsetReader) StartRecord() {
	or.currentRecordOffset = or.currentOffset
}

type RecordStateByProvider map[ProviderID]*ProviderRecordState
type ProviderRecordState struct {
	Name    string
	Records []Record
	Events  []ProviderEventType
}

type readState struct {
	numTicksPerSecond uint64
	stringTable       map[uint16]string
	threadTable       map[uint16]Thread
}

func ParseRecords(ctx context.Context, input io.Reader) (RecordStateByProvider, error) {
	recordsByProvider := RecordStateByProvider{}
	stateByProvider := map[ProviderID]*readState{}

	offsetReader := &offsetReader{
		reader:              input,
		currentRecordOffset: 0,
		currentOffset:       0,
	}

	var currentProviderState *ProviderRecordState
	var currentReadState *readState

	for {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		offsetReader.StartRecord()

		var header uint64
		if err := binary.Read(offsetReader, binary.LittleEndian, &header); err != nil {
			if errors.Is(err, io.EOF) {
				// Read to the end
				// Normal exit
				return recordsByProvider, nil
			}

			return nil, fmt.Errorf("failed to read record header at offset 0x%0x - %w", offsetReader.currentRecordOffset, err)
		}

		var record Record
		var err error

		recordType := recordType(getFieldFromValue(0, 3, header))
		switch recordType {
		case recordTypeMetadata:
			metadataType := metadataType(getFieldFromValue(16, 19, header))

			switch metadataType {
			case metadataTypeProviderInfo:
				providerID := getFieldFromValue(20, 51, header)
				nameLen := getFieldFromValue(52, 59, header)

				name, err := readString(offsetReader, int(nameLen))
				if err != nil {
					return nil, fmt.Errorf("failed to read ProviderInfo metadata record at offset 0x%0x - failed to read name at offset 0x%0x - %w", offsetReader.currentRecordOffset, offsetReader.currentOffset, err)
				}

				if _, ok := recordsByProvider[ProviderID(providerID)]; ok {
					return nil, fmt.Errorf("got multiple ProviderInfo metadata records for the same provider ID: %d", providerID)
				}

				newProviderState := &ProviderRecordState{
					Name:    name,
					Records: []Record{},
					Events:  []ProviderEventType{},
				}
				newReadState := &readState{
					stringTable: map[uint16]string{},
					threadTable: map[uint16]Thread{},
				}

				recordsByProvider[ProviderID(providerID)] = newProviderState
				stateByProvider[ProviderID(providerID)] = newReadState

				currentProviderState = newProviderState
				currentReadState = newReadState
			case metadataTypeProviderSection:
				providerID := getFieldFromValue(20, 51, header)

				providerState, ok := recordsByProvider[ProviderID(providerID)]
				if !ok {
					return nil, fmt.Errorf("got ProviderSection metadata record before the provider was defined with a ProviderInfo record - provider ID: %d", providerID)
				}

				readState, ok := stateByProvider[ProviderID(providerID)]
				if !ok {
					return nil, fmt.Errorf("got ProviderSection metadata record before the provider was defined with a ProviderInfo record - provider ID: %d", providerID)
				}

				currentProviderState = providerState
				currentReadState = readState
			case metadataTypeProviderEvent:
				providerID := getFieldFromValue(20, 51, header)
				eventType := ProviderEventType(getFieldFromValue(52, 55, header))

				providerState, ok := recordsByProvider[ProviderID(providerID)]
				if !ok {
					return nil, fmt.Errorf("got ProviderEvent metadata record before the provider was defined with a ProviderInfo record - provider ID: %d", providerID)
				}

				providerState.Events = append(providerState.Events, eventType)
			case metadataTypeTraceInfo:
				traceInfoType := traceInfoType(getFieldFromValue(20, 23, header))

				// MagicNumber is the only supported info type atm
				if traceInfoType != traceInfoTypeMagicNumber {
					return nil, fmt.Errorf("invalid Trace Info Type %d at offset 0x%0x", traceInfoType, offsetReader.currentRecordOffset)
				}

				// Validate the value
				if header != fxtMagic {
					return nil, fmt.Errorf("invalid FXT magic number %0X at offset 0x%0x", header, offsetReader.currentRecordOffset)
				}
			default:
				return nil, fmt.Errorf("invalid Metadata type %d at offset 0x%0x", metadataType, offsetReader.currentRecordOffset)
			}
		case recordTypeInitialization:
			var numTicksPerSecond uint64

			if err := binary.Read(offsetReader, binary.LittleEndian, &numTicksPerSecond); err != nil {
				return nil, fmt.Errorf("failed to read ProviderInfo metadata record at offset 0x%0x - failed to read number of ticks per second at offset 0x%0x - %w", offsetReader.currentRecordOffset, offsetReader.currentOffset, err)
			}

			currentReadState.numTicksPerSecond = numTicksPerSecond
		case recordTypeString:
			if err := currentReadState.parseStringRecord(header, offsetReader); err != nil {
				return nil, fmt.Errorf("failed to read string record at offset 0x%0x - %w", offsetReader.currentRecordOffset, err)
			}
		case recordTypeThread:
			if err = currentReadState.parseThreadRecord(header, offsetReader); err != nil {
				return nil, fmt.Errorf("failed to read thread record at offset 0x%0x - %w", offsetReader.currentRecordOffset, err)
			}
		case recordTypeEvent:
			record, err = currentReadState.parseEventRecord(header, offsetReader)
			if err != nil {
				return nil, fmt.Errorf("failed to read event record at offset 0x%0x - %w", offsetReader.currentRecordOffset, err)
			}
		case recordTypeBlob:
			record, err = currentReadState.parseBlobRecord(header, offsetReader)
			if err != nil {
				return nil, fmt.Errorf("failed to read blob record at offset 0x%0x - %w", offsetReader.currentRecordOffset, err)
			}
		case recordTypeUserspaceObject:
			record, err = currentReadState.parseUserspaceObjectRecord(header, offsetReader)
			if err != nil {
				return nil, fmt.Errorf("failed to read userspace object record at offset 0x%0x - %w", offsetReader.currentRecordOffset, err)
			}
		case recordTypeKernelObject:
			record, err = currentReadState.parseKernelObjectRecord(header, offsetReader)
			if err != nil {
				return nil, fmt.Errorf("failed to read kernel object record at offset 0x%0x - %w", offsetReader.currentRecordOffset, err)
			}
		case recordTypeScheduling:
			record, err = currentReadState.parseSchedulingRecord(header, offsetReader)
			if err != nil {
				return nil, fmt.Errorf("failed to read scheduling record at offset 0x%0x - %w", offsetReader.currentRecordOffset, err)
			}
		case recordTypeLog:
			record, err = currentReadState.parseLogRecord(header, offsetReader)
			if err != nil {
				return nil, fmt.Errorf("failed to read scheduling record at offset 0x%0x - %w", offsetReader.currentRecordOffset, err)
			}
		case recordTypeLarge:
			largeRecordType := largeRecordType(getFieldFromValue(36, 39, header))

			switch largeRecordType {
			case largeRecordTypeLargeBlob:
				record, err = currentReadState.parseLargeBlobRecord(header, offsetReader)
				if err != nil {
					return nil, fmt.Errorf("failed to read large blob record at offset 0x%0x - %w", offsetReader.currentRecordOffset, err)
				}
			default:
				return nil, fmt.Errorf("invalid large record type %d", largeRecordType)
			}
		default:
			return nil, fmt.Errorf("invalid record type %d", recordType)
		}

		// Validate we read the correct amount
		readSize := offsetReader.currentOffset - offsetReader.currentRecordOffset

		var recordSize uint64
		if recordType == recordTypeLarge {
			recordSize = getFieldFromValue(4, 35, header)

		} else {
			recordSize = getFieldFromValue(4, 15, header)
		}
		expectedSize := int64(recordSize * 8)

		// If we read less than expected, this isn't *ideal*. But we can skip over the rest and keep going
		// TODO: Should we have a way to report this somehow?
		if readSize < expectedSize {
			if err := readAndDiscard(offsetReader, expectedSize-readSize); err != nil {
				return nil, fmt.Errorf("failed to read remaining bytes of record at offset 0x%0x - %w", offsetReader.currentRecordOffset, err)
			}
		} else if readSize != expectedSize {
			return nil, fmt.Errorf("read incorrect number of bytes for record starting at offset 0x%0x - Expected to read %d bytes, but read %d", offsetReader.currentRecordOffset, expectedSize, readSize)
		}

		if record != nil {
			currentProviderState.Records = append(currentProviderState.Records, record)
		}
	}
}

func (state *readState) parseStringRecord(header uint64, offsetReader *offsetReader) error {
	strIndex := getFieldFromValue(16, 30, header)
	strLen := getFieldFromValue(32, 60, header)

	strValue, err := readString(offsetReader, int(strLen))
	if err != nil {
		return fmt.Errorf("failed to read string record value at offset 0x%0x - %w", offsetReader.currentOffset, err)
	}

	state.stringTable[uint16(strIndex)] = strValue
	return nil
}

func (state *readState) parseThreadRecord(header uint64, offsetReader *offsetReader) error {
	threadIndex := getFieldFromValue(16, 23, header)

	var processID KernelObjectID
	var threadID KernelObjectID

	if err := binary.Read(offsetReader, binary.LittleEndian, &processID); err != nil {
		return fmt.Errorf("failed to read process ID at offset 0x%0x - %w", offsetReader.currentOffset, err)
	}
	if err := binary.Read(offsetReader, binary.LittleEndian, &threadID); err != nil {
		return fmt.Errorf("failed to read thread ID at offset 0x%0x - %w", offsetReader.currentOffset, err)
	}

	state.threadTable[uint16(threadIndex)] = Thread{
		ProcessID: processID,
		ThreadID:  threadID,
	}
	return nil
}

func (state *readState) parseEventRecord(header uint64, offsetReader *offsetReader) (Record, error) {
	eventType := eventType(getFieldFromValue(16, 19, header))
	numArgs := getFieldFromValue(20, 23, header)
	threadRef := uint16(getFieldFromValue(24, 31, header))
	categoryRef := uint16(getFieldFromValue(32, 47, header))
	nameRef := uint16(getFieldFromValue(48, 63, header))

	var timestamp uint64
	if err := binary.Read(offsetReader, binary.LittleEndian, &timestamp); err != nil {
		return nil, fmt.Errorf("failed to read timestamp at offset 0x%0x - %w", offsetReader.currentOffset, err)
	}

	thread, err := state.getOrReadThread(threadRef, offsetReader)
	if err != nil {
		return nil, err
	}

	category, err := state.getOrReadString(categoryRef, offsetReader)
	if err != nil {
		return nil, err
	}

	name, err := state.getOrReadString(nameRef, offsetReader)
	if err != nil {
		return nil, err
	}

	args := map[string]any{}
	for i := 0; i < int(numArgs); i++ {
		name, value, err := state.parseArgument(offsetReader)
		if err != nil {
			return nil, err
		}

		args[name] = value
	}

	switch eventType {
	case eventTypeInstant:
		return InstantEventRecord{
			TimestampNS: state.ticksToNS(timestamp),
			Category:    category,
			Name:        name,
			Thread:      thread,
			Args:        args,
		}, nil
	case eventTypeCounter:
		var counterID uint64
		if err := binary.Read(offsetReader, binary.LittleEndian, &counterID); err != nil {
			return nil, fmt.Errorf("failed to read counter ID at offset 0x%0x - %w", offsetReader.currentOffset, err)
		}

		return CounterEventRecord{
			TimestampNS: state.ticksToNS(timestamp),
			Category:    category,
			Name:        name,
			Thread:      thread,
			Args:        args,
			CounterID:   counterID,
		}, nil
	case eventTypeDurationBegin:
		return DurationBeginEventRecord{
			TimestampNS: state.ticksToNS(timestamp),
			Category:    category,
			Name:        name,
			Thread:      thread,
			Args:        args,
		}, nil
	case eventTypeDurationEnd:
		return DurationEndEventRecord{
			TimestampNS: state.ticksToNS(timestamp),
			Category:    category,
			Name:        name,
			Thread:      thread,
			Args:        args,
		}, nil
	case eventTypeDurationComplete:
		var numTicks uint64
		if err := binary.Read(offsetReader, binary.LittleEndian, &numTicks); err != nil {
			return nil, fmt.Errorf("failed to read number of ticks at offset 0x%0x - %w", offsetReader.currentOffset, err)
		}

		return DurationCompleteEventRecord{
			TimestampNS: state.ticksToNS(timestamp),
			Category:    category,
			Name:        name,
			Thread:      thread,
			Args:        args,
			DurationNS:  state.ticksToNS(numTicks),
		}, nil
	case eventTypeAsyncBegin:
		var correlationID uint64
		if err := binary.Read(offsetReader, binary.LittleEndian, &correlationID); err != nil {
			return nil, fmt.Errorf("failed to read correlation ID at offset 0x%0x - %w", offsetReader.currentOffset, err)
		}

		return AsyncBeginEventRecord{
			TimestampNS:   state.ticksToNS(timestamp),
			Category:      category,
			Name:          name,
			Thread:        thread,
			Args:          args,
			CorrelationID: correlationID,
		}, nil
	case eventTypeAsyncInstant:
		var correlationID uint64
		if err := binary.Read(offsetReader, binary.LittleEndian, &correlationID); err != nil {
			return nil, fmt.Errorf("failed to read correlation ID at offset 0x%0x - %w", offsetReader.currentOffset, err)
		}

		return AsyncInstantEventRecord{
			TimestampNS:   state.ticksToNS(timestamp),
			Category:      category,
			Name:          name,
			Thread:        thread,
			Args:          args,
			CorrelationID: correlationID,
		}, nil
	case eventTypeAsyncEnd:
		var correlationID uint64
		if err := binary.Read(offsetReader, binary.LittleEndian, &correlationID); err != nil {
			return nil, fmt.Errorf("failed to read correlation ID at offset 0x%0x - %w", offsetReader.currentOffset, err)
		}

		return AsyncEndEventRecord{
			TimestampNS:   state.ticksToNS(timestamp),
			Category:      category,
			Name:          name,
			Thread:        thread,
			Args:          args,
			CorrelationID: correlationID,
		}, nil
	case eventTypeFlowBegin:
		var correlationID uint64
		if err := binary.Read(offsetReader, binary.LittleEndian, &correlationID); err != nil {
			return nil, fmt.Errorf("failed to read correlation ID at offset 0x%0x - %w", offsetReader.currentOffset, err)
		}

		return FlowBeginEvent{
			TimestampNS:   state.ticksToNS(timestamp),
			Category:      category,
			Name:          name,
			Thread:        thread,
			Args:          args,
			CorrelationID: correlationID,
		}, nil
	case eventTypeFlowStep:
		var correlationID uint64
		if err := binary.Read(offsetReader, binary.LittleEndian, &correlationID); err != nil {
			return nil, fmt.Errorf("failed to read correlation ID at offset 0x%0x - %w", offsetReader.currentOffset, err)
		}

		return FlowStepEvent{
			TimestampNS:   state.ticksToNS(timestamp),
			Category:      category,
			Name:          name,
			Thread:        thread,
			Args:          args,
			CorrelationID: correlationID,
		}, nil
	case eventTypeFlowEnd:
		var correlationID uint64
		if err := binary.Read(offsetReader, binary.LittleEndian, &correlationID); err != nil {
			return nil, fmt.Errorf("failed to read correlation ID at offset 0x%0x - %w", offsetReader.currentOffset, err)
		}

		return FlowEndEvent{
			TimestampNS:   state.ticksToNS(timestamp),
			Category:      category,
			Name:          name,
			Thread:        thread,
			Args:          args,
			CorrelationID: correlationID,
		}, nil
	default:
		return nil, fmt.Errorf("invalid event type %d", eventType)
	}
}

func (state *readState) parseBlobRecord(header uint64, offsetReader *offsetReader) (Record, error) {
	nameRef := uint16(getFieldFromValue(16, 31, header))
	payloadSize := getFieldFromValue(32, 46, header)
	blobType := BlobType(getFieldFromValue(48, 55, header))

	name, err := state.getOrReadString(nameRef, offsetReader)
	if err != nil {
		return nil, fmt.Errorf("failed to read blob record name at offset 0x%0x - %w", offsetReader.currentOffset, err)
	}

	payload, err := readBlob(offsetReader, int(payloadSize))
	if err != nil {
		return nil, fmt.Errorf("failed to read blob payload at offset 0x%0x - %w", offsetReader.currentOffset, err)
	}

	return BlobRecord{
		Name:    name,
		Type:    blobType,
		Payload: payload,
	}, nil
}

func (state *readState) parseUserspaceObjectRecord(header uint64, offsetReader *offsetReader) (Record, error) {
	threadRef := uint16(getFieldFromValue(16, 23, header))
	nameRef := uint16(getFieldFromValue(24, 39, header))
	numArgs := getFieldFromValue(40, 43, header)

	var pointerVal uint64
	if err := binary.Read(offsetReader, binary.LittleEndian, &pointerVal); err != nil {
		return nil, fmt.Errorf("failed to read pointer value at offset 0x%0x - %w", offsetReader.currentOffset, err)
	}

	// The inline thread only has the process ID, so we can't use getOrReadThread()
	var processID KernelObjectID

	if threadRef == 0 {
		// Inline

		if err := binary.Read(offsetReader, binary.LittleEndian, &processID); err != nil {
			return nil, fmt.Errorf("failed to read process ID at offset 0x%0x - %w", offsetReader.currentOffset, err)
		}
	} else {
		thread, ok := state.threadTable[threadRef]
		if !ok {
			return nil, fmt.Errorf("record referenced unknown thread index %d", threadRef)
		}

		processID = thread.ProcessID
	}

	name, err := state.getOrReadString(nameRef, offsetReader)
	if err != nil {
		return nil, fmt.Errorf("failed to read userspace object name at offset 0x%0x - %w", offsetReader.currentOffset, err)
	}

	args := map[string]any{}
	for i := 0; i < int(numArgs); i++ {
		name, value, err := state.parseArgument(offsetReader)
		if err != nil {
			return nil, err
		}

		args[name] = value
	}

	return UserspaceObjectRecord{
		Name:      name,
		ProcessID: processID,
		Pointer:   uintptr(pointerVal),
		Args:      args,
	}, nil
}

func (state *readState) parseKernelObjectRecord(header uint64, offsetReader *offsetReader) (Record, error) {
	kernelObjectType := KernelObjectType(getFieldFromValue(16, 23, header))
	nameRef := uint16(getFieldFromValue(24, 39, header))
	numArgs := getFieldFromValue(40, 43, header)

	var koid KernelObjectID
	if err := binary.Read(offsetReader, binary.LittleEndian, &koid); err != nil {
		return nil, fmt.Errorf("failed to read KOID at offset 0x%0x - %w", offsetReader.currentOffset, err)
	}

	name, err := state.getOrReadString(nameRef, offsetReader)
	if err != nil {
		return nil, fmt.Errorf("failed to read kernel object name at offset 0x%0x - %w", offsetReader.currentOffset, err)
	}

	args := map[string]any{}
	for i := 0; i < int(numArgs); i++ {
		name, value, err := state.parseArgument(offsetReader)
		if err != nil {
			return nil, err
		}

		args[name] = value
	}

	return KernelObjectRecord{
		Type: kernelObjectType,
		ID:   koid,
		Name: name,
		Args: args,
	}, nil
}

func (state *readState) parseSchedulingRecord(header uint64, offsetReader *offsetReader) (Record, error) {
	schedulingRecordType := schedulingRecordType(getFieldFromValue(60, 63, header))

	switch schedulingRecordType {
	case schedulingRecordTypeContextSwitch:
		numArgs := getFieldFromValue(16, 19, header)
		cpuNumber := uint16(getFieldFromValue(20, 35, header))
		outThreadState := ThreadStateType(getFieldFromValue(36, 39, header))

		var timestamp uint64
		if err := binary.Read(offsetReader, binary.LittleEndian, &timestamp); err != nil {
			return nil, fmt.Errorf("failed to read timestamp at offset 0x%0x - %w", offsetReader.currentOffset, err)
		}

		var outgoingThreadID KernelObjectID
		if err := binary.Read(offsetReader, binary.LittleEndian, &outgoingThreadID); err != nil {
			return nil, fmt.Errorf("failed to read outgoing thread ID at offset 0x%0x - %w", offsetReader.currentOffset, err)
		}

		var incomingThreadID KernelObjectID
		if err := binary.Read(offsetReader, binary.LittleEndian, &incomingThreadID); err != nil {
			return nil, fmt.Errorf("failed to read incoming thread ID at offset 0x%0x - %w", offsetReader.currentOffset, err)
		}

		args := map[string]any{}
		for i := 0; i < int(numArgs); i++ {
			name, value, err := state.parseArgument(offsetReader)
			if err != nil {
				return nil, err
			}

			args[name] = value
		}

		return ContextSwitchRecord{
			TimestampNS:         state.ticksToNS(timestamp),
			CPUID:               cpuNumber,
			OutgoingThreadID:    outgoingThreadID,
			OutgoingThreadState: outThreadState,
			IncomingThreadID:    incomingThreadID,
			Args:                args,
		}, nil
	case schedulingRecordTypeThreadWakeup:
		numArgs := getFieldFromValue(16, 19, header)
		cpuNumber := uint16(getFieldFromValue(20, 35, header))

		var timestamp uint64
		if err := binary.Read(offsetReader, binary.LittleEndian, &timestamp); err != nil {
			return nil, fmt.Errorf("failed to read timestamp at offset 0x%0x - %w", offsetReader.currentOffset, err)
		}

		var wakingThreadID KernelObjectID
		if err := binary.Read(offsetReader, binary.LittleEndian, &wakingThreadID); err != nil {
			return nil, fmt.Errorf("failed to read waking thread ID at offset 0x%0x - %w", offsetReader.currentOffset, err)
		}

		args := map[string]any{}
		for i := 0; i < int(numArgs); i++ {
			name, value, err := state.parseArgument(offsetReader)
			if err != nil {
				return nil, err
			}

			args[name] = value
		}

		return ThreadWakeupRecord{
			TimestampNS:    state.ticksToNS(timestamp),
			CPUID:          cpuNumber,
			WakingThreadID: wakingThreadID,
			Args:           args,
		}, nil
	default:
		return nil, fmt.Errorf("invalid scheduling type %d", schedulingRecordType)
	}
}

func (state *readState) parseLogRecord(header uint64, offsetReader *offsetReader) (Record, error) {
	logMessageLen := getFieldFromValue(16, 30, header)
	threadRef := uint16(getFieldFromValue(32, 39, header))

	var timestamp uint64
	if err := binary.Read(offsetReader, binary.LittleEndian, &timestamp); err != nil {
		return nil, fmt.Errorf("failed to read timestamp at offset 0x%0x - %w", offsetReader.currentOffset, err)
	}

	thread, err := state.getOrReadThread(threadRef, offsetReader)
	if err != nil {
		return nil, err
	}

	message, err := readString(offsetReader, int(logMessageLen))
	if err != nil {
		return nil, fmt.Errorf("failed to read log data at offset 0x%0x - %w", offsetReader.currentOffset, err)
	}

	return LogRecord{
		TimestampNS: state.ticksToNS(timestamp),
		Thread:      thread,
		Message:     message,
	}, nil
}

func (state *readState) parseLargeBlobRecord(header uint64, offsetReader *offsetReader) (Record, error) {
	largeBlobType := largeBlobType(getFieldFromValue(40, 43, header))

	switch largeBlobType {
	case largeBlobTypeWithMetadata:
		var formatHeader uint64
		if err := binary.Read(offsetReader, binary.LittleEndian, &formatHeader); err != nil {
			return nil, fmt.Errorf("failed to read large record format header at offset 0x%0x - %w", offsetReader.currentRecordOffset, err)
		}

		categoryRef := uint16(getFieldFromValue(0, 15, formatHeader))
		nameRef := uint16(getFieldFromValue(16, 31, formatHeader))
		numArgs := getFieldFromValue(32, 35, formatHeader)
		threadRef := uint16(getFieldFromValue(36, 43, formatHeader))

		category, err := state.getOrReadString(categoryRef, offsetReader)
		if err != nil {
			return nil, err
		}

		name, err := state.getOrReadString(nameRef, offsetReader)
		if err != nil {
			return nil, err
		}

		var timestamp uint64
		if err := binary.Read(offsetReader, binary.LittleEndian, &timestamp); err != nil {
			return nil, fmt.Errorf("failed to read timestamp at offset 0x%0x - %w", offsetReader.currentOffset, err)
		}

		thread, err := state.getOrReadThread(threadRef, offsetReader)
		if err != nil {
			return nil, err
		}

		args := map[string]any{}
		for i := 0; i < int(numArgs); i++ {
			name, value, err := state.parseArgument(offsetReader)
			if err != nil {
				return nil, err
			}

			args[name] = value
		}

		var blobSize uint64
		if err := binary.Read(offsetReader, binary.LittleEndian, &blobSize); err != nil {
			return nil, fmt.Errorf("failed to read blob size at offset 0x%0x - %w", offsetReader.currentOffset, err)
		}

		payload, err := readBlob(offsetReader, int(blobSize))
		if err != nil {
			return nil, fmt.Errorf("failed to read blob payload at offset 0x%0x - %w", offsetReader.currentOffset, err)
		}

		return LargeBlobWithMetadataRecord{
			TimestampNS: state.ticksToNS(timestamp),
			Category:    category,
			Name:        name,
			Thread:      thread,
			Args:        args,
			Payload:     payload,
		}, nil
	case largeBlobTypeNoMetadata:
		var formatHeader uint64
		if err := binary.Read(offsetReader, binary.LittleEndian, &formatHeader); err != nil {
			return nil, fmt.Errorf("failed to read large record format header at offset 0x%0x - %w", offsetReader.currentRecordOffset, err)
		}

		categoryRef := uint16(getFieldFromValue(0, 15, formatHeader))
		nameRef := uint16(getFieldFromValue(16, 31, formatHeader))

		category, err := state.getOrReadString(categoryRef, offsetReader)
		if err != nil {
			return nil, err
		}

		name, err := state.getOrReadString(nameRef, offsetReader)
		if err != nil {
			return nil, err
		}

		var blobSize uint64
		if err := binary.Read(offsetReader, binary.LittleEndian, &blobSize); err != nil {
			return nil, fmt.Errorf("failed to read blob size at offset 0x%0x - %w", offsetReader.currentOffset, err)
		}

		payload, err := readBlob(offsetReader, int(blobSize))
		if err != nil {
			return nil, fmt.Errorf("failed to read blob payload at offset 0x%0x - %w", offsetReader.currentOffset, err)
		}

		return LargeBlobNoMetadataRecord{
			Category: category,
			Name:     name,
			Payload:  payload,
		}, nil
	default:
		return nil, fmt.Errorf("invalid large blob format type %d", largeBlobType)
	}
}

func (state *readState) getOrReadThread(threadRef uint16, offsetReader *offsetReader) (Thread, error) {
	if threadRef == 0 {
		// Inline thread
		var processID KernelObjectID
		var threadID KernelObjectID

		if err := binary.Read(offsetReader, binary.LittleEndian, &processID); err != nil {
			return Thread{}, fmt.Errorf("failed to read process ID at offset 0x%0x - %w", offsetReader.currentOffset, err)
		}
		if err := binary.Read(offsetReader, binary.LittleEndian, &threadID); err != nil {
			return Thread{}, fmt.Errorf("failed to read thread ID at offset 0x%0x - %w", offsetReader.currentOffset, err)
		}

		return Thread{
			ProcessID: processID,
			ThreadID:  threadID,
		}, nil
	}

	thread, ok := state.threadTable[threadRef]
	if !ok {
		return Thread{}, fmt.Errorf("record referenced unknown thread index %d", threadRef)
	}

	return thread, nil
}

func (state *readState) getOrReadString(strRef uint16, offsetReader *offsetReader) (string, error) {
	if strRef == 0 {
		// Empty string
		return "", nil
	}

	if (strRef & 0x8000) == 0x8000 {
		// Inline thread
		return readString(offsetReader, int(strRef&^0x8000))
	}

	str, ok := state.stringTable[strRef]
	if !ok {
		return "", fmt.Errorf("record referenced unknown string index %d", strRef)
	}

	return str, nil
}

func (state *readState) parseArgument(offsetReader *offsetReader) (name string, value any, err error) {
	startOffset := offsetReader.currentOffset

	var header uint64
	if err := binary.Read(offsetReader, binary.LittleEndian, &header); err != nil {
		return "", nil, fmt.Errorf("failed to read argument header at offset 0x%0x - %w", startOffset, err)
	}

	nameRef := getFieldFromValue(16, 31, header)
	name, err = state.getOrReadString(uint16(nameRef), offsetReader)
	if err != nil {
		return "", nil, fmt.Errorf("failed to read argument name for argument at offset 0x%0x - %w", startOffset, err)
	}

	argumentType := argumentType(getFieldFromValue(0, 3, header))
	switch argumentType {
	case argumentTypeNull:
		// Nothing extra to parse
	case argumentTypeInt32:
		value = int32(getFieldFromValue(32, 63, header))
	case argumentTypeUInt32:
		value = uint32(getFieldFromValue(32, 63, header))
	case argumentTypeInt64:
		var i64Value int64
		if err := binary.Read(offsetReader, binary.LittleEndian, &i64Value); err != nil {
			return "", nil, fmt.Errorf("failed to read argument value for argument at offset 0x%0x - %w", startOffset, err)
		}
		value = i64Value
	case argumentTypeUInt64:
		var u64Value uint64
		if err := binary.Read(offsetReader, binary.LittleEndian, &u64Value); err != nil {
			return "", nil, fmt.Errorf("failed to read argument value for argument at offset 0x%0x - %w", startOffset, err)
		}
		value = u64Value
	case argumentTypeDouble:
		var doubleValue float64
		if err := binary.Read(offsetReader, binary.LittleEndian, &doubleValue); err != nil {
			return "", nil, fmt.Errorf("failed to read argument value for argument at offset 0x%0x - %w", startOffset, err)
		}
		value = doubleValue
	case argumentTypeString:
		strRef := uint16(getFieldFromValue(32, 47, header))

		strValue, err := state.getOrReadString(strRef, offsetReader)
		if err != nil {
			return "", nil, fmt.Errorf("failed to read argument value for argument at offset 0x%0x - %w", startOffset, err)
		}
		value = strValue
	case argumentTypePointer:
		var u64Value uint64
		if err := binary.Read(offsetReader, binary.LittleEndian, &u64Value); err != nil {
			return "", nil, fmt.Errorf("failed to read argument value for argument at offset 0x%0x - %w", startOffset, err)
		}
		value = uintptr(u64Value)
	case argumentTypeKOID:
		var u64Value uint64
		if err := binary.Read(offsetReader, binary.LittleEndian, &u64Value); err != nil {
			return "", nil, fmt.Errorf("failed to read argument value for argument at offset 0x%0x - %w", startOffset, err)
		}
		value = KernelObjectType(u64Value)
	case argumentTypeBool:
		value = getFieldFromValue(32, 32, header) == 1
	}

	// Validate we read the correct amount
	readSize := offsetReader.currentOffset - startOffset

	argumentSize := getFieldFromValue(4, 15, header)
	if readSize != int64(argumentSize)*8 {
		return "", nil, fmt.Errorf("read incorrect number of bytes for argument starting at offset 0x%0x - Expected to read %d bytes, but read %d", startOffset, readSize, argumentSize)
	}

	return name, value, nil
}

func (state *readState) ticksToNS(ticks uint64) uint64 {
	nsPerSec := uint64(1000000000)
	return ticks * nsPerSec / state.numTicksPerSecond
}

func readString(input io.Reader, len int) (string, error) {
	blob, err := readBlob(input, len)
	return string(blob), err
}

func readBlob(input io.Reader, len int) ([]byte, error) {
	// Pad out the length to a multiple of 8 byte alignment
	paddedLen := ((len + (8 - 1)) &^ (8 - 1))

	// Read all the bytes
	data := make([]byte, paddedLen)

	_, err := io.ReadFull(input, data)
	if err != nil {
		return nil, fmt.Errorf("failed to read padded string - %w", err)
	}

	return data[:len], nil
}

func readAndDiscard(input io.Reader, len int64) error {
	_, err := io.CopyN(io.Discard, input, len)
	return err
}
