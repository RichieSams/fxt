package fxt

import (
	"cmp"
	"fmt"
	"slices"
	"strconv"
	"time"
)

type SpansByProcess map[KernelObjectID]*ProcessState

type ProcessState struct {
	Name    string
	Threads map[KernelObjectID]*ThreadState
}

type ThreadState struct {
	Name  string
	Spans []*Span
}

type ThreadSpans struct {
	ThreadID KernelObjectID
	Spans    []*Span
}

type Span struct {
	TimestampNS uint64
	Category    string
	Name        string
	Duration    time.Duration
	Args        map[string]any

	Parent   *Span
	Children []*Span
}

type transformThreadState struct {
	spanStack []*Span

	spans []*Span
}

func TransformRecordsToSpans(records []Record) (SpansByProcess, error) {
	processNames := map[KernelObjectID]string{}
	threadNames := map[KernelObjectID]string{}

	filteredRecords := []timestampedRecord{}

	for _, genericRecord := range records {
		switch record := genericRecord.(type) {
		case KernelObjectRecord:
			switch record.Type {
			case kernelObjectTypeProcess:
				processNames[record.ID] = record.Name
			case kernelObjectTypeThread:
				threadNames[record.ID] = record.Name
			default:
				return nil, fmt.Errorf("unknown Kernel Object Record type: %d", record.Type)
			}
		case DurationBeginEventRecord:
			filteredRecords = append(filteredRecords, record)
		case DurationEndEventRecord:
			filteredRecords = append(filteredRecords, record)
		case DurationCompleteEventRecord:
			filteredRecords = append(filteredRecords, record)
		default:
			// Skip the rest
		}
	}

	slices.SortStableFunc(filteredRecords, func(a timestampedRecord, b timestampedRecord) int {
		return cmp.Compare(a.getTimestampNS(), b.getTimestampNS())
	})

	maxTimestampNS := filteredRecords[len(filteredRecords)-1].getTimestampNS()

	processes := map[KernelObjectID]map[KernelObjectID]*transformThreadState{}
	for _, genericRecord := range filteredRecords {
		switch record := genericRecord.(type) {
		case DurationBeginEventRecord:
			span := &Span{
				TimestampNS: record.TimestampNS,
				Category:    record.Category,
				Name:        record.Name,
				Args:        record.Args,
				Parent:      nil,
				Children:    []*Span{},
			}

			processMap, ok := processes[record.Thread.ProcessID]
			if !ok {
				processMap = map[KernelObjectID]*transformThreadState{}
				processes[record.Thread.ProcessID] = processMap
			}

			threadState, ok := processMap[record.Thread.ThreadID]
			if !ok {
				threadState = &transformThreadState{
					spanStack: []*Span{},
					spans:     []*Span{},
				}
				processMap[record.Thread.ThreadID] = threadState
			}

			// If the stack is empty, we start a new span at the root
			// and add it to the stack
			if len(threadState.spanStack) == 0 {
				threadState.spanStack = append(threadState.spanStack, span)
				threadState.spans = append(threadState.spans, span)
			} else {
				// Otherwise, we create a child span of the span at the top of the stack
				parent := threadState.spanStack[len(threadState.spanStack)-1]
				span.Parent = parent
				parent.Children = append(parent.Children, span)

				// Now we push to the stack
				threadState.spanStack = append(threadState.spanStack, span)
			}
		case DurationEndEventRecord:
			processMap, ok := processes[record.Thread.ProcessID]
			if !ok {
				return nil, fmt.Errorf("invalid DurationEndEventRecord %s:%s - ProcessID %d is not yet known", record.Category, record.Name, record.Thread.ProcessID)
			}

			threadState, ok := processMap[record.Thread.ThreadID]
			if !ok {
				return nil, fmt.Errorf("invalid DurationEndEventRecord %s:%s - ThreadID %d is not yet known", record.Category, record.Name, record.Thread.ThreadID)
			}

			stackLen := len(threadState.spanStack)
			if stackLen == 0 {
				return nil, fmt.Errorf("invalid DurationEndEventRecord %s:%s - no matching DurationBeginEventRecord", record.Category, record.Name)
			}

			// Pop from the stack and set the relevant info
			span := threadState.spanStack[stackLen-1]
			threadState.spanStack = threadState.spanStack[:stackLen-1]

			// Check that we're closing the right span
			if record.Category != span.Category {
				return nil, fmt.Errorf("invalid DurationEndEventRecord %s:%s - no matching DurationBeginEventRecord", record.Category, record.Name)
			}
			if record.Name != span.Name {
				return nil, fmt.Errorf("invalid DurationEndEventRecord %s:%s - no matching DurationBeginEventRecord", record.Category, record.Name)
			}

			span.Duration = time.Duration(record.TimestampNS-span.TimestampNS) * time.Nanosecond
			for key, value := range record.Args {
				span.Args[key] = value
			}
		case DurationCompleteEventRecord:
			span := &Span{
				TimestampNS: record.TimestampNS,
				Category:    record.Category,
				Name:        record.Name,
				Duration:    time.Duration(record.DurationNS) * time.Nanosecond,
				Args:        record.Args,
				Parent:      nil,
				Children:    []*Span{},
			}

			processMap, ok := processes[record.Thread.ProcessID]
			if !ok {
				processMap = map[KernelObjectID]*transformThreadState{}
				processes[record.Thread.ProcessID] = processMap
			}

			threadState, ok := processMap[record.Thread.ThreadID]
			if !ok {
				threadState = &transformThreadState{
					spanStack: []*Span{},
					spans:     []*Span{},
				}
				processMap[record.Thread.ThreadID] = threadState
			}

			// If the stack is empty, we just add our span parented to the root
			if len(threadState.spanStack) == 0 {
				threadState.spans = append(threadState.spans, span)
			} else {
				// Otherwise, we create a child span of the span at the top of the stack
				parent := threadState.spanStack[len(threadState.spanStack)-1]
				span.Parent = parent
				parent.Children = append(parent.Children, span)
			}
		}
	}

	output := SpansByProcess{}
	for processID, spansByThread := range processes {
		processName, ok := processNames[processID]
		if !ok {
			processName = strconv.FormatUint(uint64(processID), 10)
		}

		processState := &ProcessState{
			Name:    processName,
			Threads: map[KernelObjectID]*ThreadState{},
		}

		for threadID, threadState := range spansByThread {
			// Close any dangling spans with the maxTimestampNS value
			for {
				stackLen := len(threadState.spanStack)
				if stackLen == 0 {
					break
				}

				span := threadState.spanStack[stackLen-1]
				threadState.spanStack = threadState.spanStack[:stackLen-1]

				span.Duration = time.Duration(maxTimestampNS-span.TimestampNS) * time.Nanosecond
			}

			threadName, ok := threadNames[threadID]
			if !ok {
				threadName = strconv.FormatUint(uint64(threadID), 10)
			}

			processState.Threads[threadID] = &ThreadState{
				Name:  threadName,
				Spans: threadState.spans,
			}
		}
	}

	return output, nil
}
