package fxt_test

import (
	"context"
	"os"
	"testing"

	"github.com/richiesams/fxt"
	"github.com/stretchr/testify/require"
)

func TestReader(t *testing.T) {
	ctx := context.Background()

	file, err := os.Open("test_data/trace.fxt")
	require.NoError(t, err)
	defer func() {
		err := file.Close()
		require.NoError(t, err)
	}()

	recordStateByProvider, err := fxt.ParseRecords(ctx, file)
	require.NoError(t, err)

	for _, state := range recordStateByProvider {
		_, err = fxt.TransformRecordsToSpans(state.Records)
		require.NoError(t, err)
	}
}
