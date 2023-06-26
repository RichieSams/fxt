package fxt_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/richiesams/fxt"

	"github.com/stretchr/testify/require"
)

func TestWrite(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "")
	require.NoError(t, err)

	defer func() {
		err := os.RemoveAll(tempDir)
		require.NoError(t, err)
	}()

	writer, err := fxt.NewWriter(filepath.Join(tempDir, "test.fxt"))
	require.NoError(t, err)

	closed := false
	defer func() {
		if !closed {
			err := writer.Close()
			require.NoError(t, err)
		}
	}()

	// Set up the provider info
	err = writer.AddProviderInfoRecord(1234, "Test Provider")
	require.NoError(t, err)
	err = writer.AddProviderSectionRecord(1234)
	require.NoError(t, err)
	err = writer.AddInitializationRecord(1000)
	require.NoError(t, err)

	// Name the processes / threads
	err = writer.SetProcessName(3, "Test.exe")
	require.NoError(t, err)
	err = writer.SetThreadName(3, 45, "Main")
	require.NoError(t, err)
	err = writer.SetThreadName(3, 87, "Worker0")
	require.NoError(t, err)
	err = writer.SetProcessName(4, "Server.exe")
	require.NoError(t, err)
	err = writer.SetThreadName(4, 50, "ServerThread")
	require.NoError(t, err)

	// Do a basic set of spans
	// And throw in some async
	err = writer.AddDurationBeginEvent("Foo", "Root", 3, 45, 200)
	require.NoError(t, err)

	err = writer.AddInstantEvent("OtherThing", "EventHappened", 3, 45, 300)
	require.NoError(t, err)

	err = writer.AddDurationBeginEvent("Foo", "Inner", 3, 45, 400)
	require.NoError(t, err)

	err = writer.AddAsyncBeginEvent("Asdf", "AsyncThing", 3, 45, 450, 111)
	require.NoError(t, err)

	err = writer.AddDurationCompleteEvent("OtherService", "DoStuff", 3, 45, 500, 800)
	require.NoError(t, err)

	err = writer.AddAsyncInstantEvent("Asdf", "AsyncInstant", 3, 87, 825, 111)
	require.NoError(t, err)

	err = writer.AddAsyncEndEvent("Asdf", "AsyncThing", 3, 87, 850, 111)
	require.NoError(t, err)

	err = writer.AddDurationEndEvent("Foo", "Inner", 3, 45, 900)
	require.NoError(t, err)

	err = writer.AddDurationEndEvent("Foo", "Root", 3, 45, 900)
	require.NoError(t, err)

	// Test out flows
	err = writer.AddDurationBeginEvent("CategoryA", "REST Request to server", 3, 45, 950)
	require.NoError(t, err)

	err = writer.AddFlowBeginEvent("CategoryA", "AwesomeFlow", 3, 45, 955, 123)
	require.NoError(t, err)

	err = writer.AddDurationEndEvent("CategoryA", "REST Request to server", 3, 45, 1000)
	require.NoError(t, err)

	err = writer.AddDurationBeginEvent("CategoryA", "Server process request", 4, 50, 1000)
	require.NoError(t, err)

	err = writer.AddFlowStepEvent("CategoryA", "Server request handler", 4, 50, 1005, 123)
	require.NoError(t, err)

	err = writer.AddDurationEndEvent("CategoryA", "Server process request", 4, 50, 1100)
	require.NoError(t, err)

	err = writer.AddDurationBeginEvent("CategoryA", "Process server response", 3, 45, 1150)
	require.NoError(t, err)

	err = writer.AddFlowEndEvent("CategoryA", "AwesomeFlow", 3, 45, 1155, 123)
	require.NoError(t, err)

	err = writer.AddDurationEndEvent("CategoryA", "Process server response", 3, 45, 1200)
	require.NoError(t, err)

	// Add some counter events
	err = writer.AddCounterEvent(
		"Bar", "CounterA", 3, 45, 250,
		map[string]interface{}{
			"int_arg":    int32(111),
			"uint_arg":   uint32(984),
			"double_arg": float64(1.0),
			"int64_arg":  int64(851),
			"uint64_arg": uint64(35),
		},
		555,
	)
	require.NoError(t, err)

	err = writer.AddCounterEvent(
		"Bar", "CounterA", 3, 45, 500,
		map[string]interface{}{
			"int_arg":    int32(784),
			"uint_arg":   uint32(561),
			"double_arg": float64(4.0),
			"int64_arg":  int64(445),
			"uint64_arg": uint64(95),
		},
		555,
	)
	require.NoError(t, err)

	err = writer.AddCounterEvent(
		"Bar", "CounterA", 3, 45, 1000,
		map[string]interface{}{
			"int_arg":    int32(333),
			"uint_arg":   uint32(845),
			"double_arg": float64(9.0),
			"int64_arg":  int64(521),
			"uint64_arg": uint64(24),
		},
		555,
	)
	require.NoError(t, err)

	err = writer.Close()
	closed = true
	require.NoError(t, err)
}
