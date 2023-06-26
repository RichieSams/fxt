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

	err = writer.Close()
	closed = true
	require.NoError(t, err)
}
