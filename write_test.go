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

	// Do a basic set of spans
	err = writer.AddDurationBeginEvent("Foo", "Root", 3, 45, 200)
	require.NoError(t, err)

	err = writer.AddInstantEvent("OtherThing", "EventHappened", 3, 45, 300)
	require.NoError(t, err)

	err = writer.AddDurationBeginEvent("Foo", "Inner", 3, 45, 400)
	require.NoError(t, err)

	err = writer.AddDurationCompleteEvent("OtherService", "DoStuff", 3, 45, 500, 800)
	require.NoError(t, err)

	err = writer.AddDurationEndEvent("Foo", "Inner", 3, 45, 900)
	require.NoError(t, err)

	err = writer.AddDurationEndEvent("Foo", "Root", 3, 45, 900)
	require.NoError(t, err)

	err = writer.Close()
	closed = true
	require.NoError(t, err)
}
