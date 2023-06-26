package fxt

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStringTableFetchNonExistantEntryFails(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "")
	require.NoError(t, err)

	defer func() {
		err := os.RemoveAll(tempDir)
		require.NoError(t, err)
	}()

	writer, err := NewWriter(filepath.Join(tempDir, "test.fxt"))
	require.NoError(t, err)

	closed := false
	defer func() {
		if !closed {
			err := writer.Close()
			require.NoError(t, err)
		}
	}()

	// If we try to fetch before it's in the table, it will fail
	_, err = writer.getStringIndex("test")
	require.Error(t, err)

	// If we add to the table and try again, it should succeed
	_, err = writer.getOrCreateStringIndex("test")
	require.NoError(t, err)

	index, err := writer.getStringIndex("test")
	require.NoError(t, err)
	require.Equal(t, uint16(1), index)

	err = writer.Close()
	closed = true
	require.NoError(t, err)
}
