package fxt

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetFieldFromValue(t *testing.T) {
	tests := []struct {
		name     string
		begin    uint64
		end      uint64
		value    uint64
		expected uint64
	}{
		{
			name:     "extract single bit at position 0",
			begin:    0,
			end:      0,
			value:    0b1010,
			expected: 0,
		},
		{
			name:     "extract single bit at position 1",
			begin:    1,
			end:      1,
			value:    0b1010,
			expected: 1,
		},
		{
			name:     "extract single bit at position 3",
			begin:    3,
			end:      3,
			value:    0b1010,
			expected: 1,
		},
		{
			name:     "extract 2 bits from middle",
			begin:    1,
			end:      2,
			value:    0b1110,
			expected: 0b11,
		},
		{
			name:     "extract 4 bits from beginning",
			begin:    0,
			end:      3,
			value:    0b11110000,
			expected: 0b0000,
		},
		{
			name:     "extract 4 bits from end",
			begin:    4,
			end:      7,
			value:    0b11110000,
			expected: 0b1111,
		},
		{
			name:     "extract entire 8-bit value",
			begin:    0,
			end:      7,
			value:    0b10101010,
			expected: 0b10101010,
		},
		{
			name:     "extract from larger value",
			begin:    8,
			end:      15,
			value:    0xABCD,
			expected: 0xAB,
		},
		{
			name:     "extract single bit from large value",
			begin:    16,
			end:      16,
			value:    0x10000,
			expected: 1,
		},
		{
			name:     "extract zero from zero value",
			begin:    0,
			end:      7,
			value:    0,
			expected: 0,
		},
		{
			name:     "extract from max uint64",
			begin:    60,
			end:      63,
			value:    0xFFFFFFFFFFFFFFFF,
			expected: 0xF,
		},
		{
			name:     "extract 16-bit field",
			begin:    16,
			end:      31,
			value:    0x12345678,
			expected: 0x1234,
		},
		{
			name:     "extract overlapping middle bits",
			begin:    2,
			end:      5,
			value:    0b11111100,
			expected: 0b1111,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getFieldFromValue(tt.begin, tt.end, tt.value)
			require.Equal(t, tt.expected, result,
				"getFieldFromValue(%d, %d, 0x%X) = 0x%X, expected 0x%X",
				tt.begin, tt.end, tt.value, result, tt.expected)
		})
	}
}

func TestGetFieldFromValueEdgeCases(t *testing.T) {
	t.Run("same begin and end position", func(t *testing.T) {
		// Testing various single bit extractions
		for i := uint64(0); i < 8; i++ {
			value := uint64(1 << i)
			result := getFieldFromValue(i, i, value)
			require.Equal(t, uint64(1), result,
				"extracting bit %d from value with only bit %d set should return 1", i, i)
		}
	})

	t.Run("extract all bits", func(t *testing.T) {
		value := uint64(0x123456789ABCDEF)
		result := getFieldFromValue(0, 63, value)
		require.Equal(t, value, result, "extracting all 64 bits should return original value")
	})

	t.Run("high bit positions", func(t *testing.T) {
		// Test extracting from high bit positions
		value := uint64(0x8000000000000000) // MSB set
		result := getFieldFromValue(63, 63, value)
		require.Equal(t, uint64(1), result, "should extract MSB correctly")
	})
}

func TestGetFieldFromValueBitMaskValidation(t *testing.T) {
	// Test that the function properly masks bits outside the range
	t.Run("mask validation", func(t *testing.T) {
		// Value has bits set outside our extraction range
		value := uint64(0xFFFFFFFF) // All lower 32 bits set

		// Extract only bits 4-7 (should be 0xF)
		result := getFieldFromValue(4, 7, value)
		require.Equal(t, uint64(0xF), result,
			"should only extract bits 4-7, ignoring other set bits")

		// Extract bits 16-19 (should be 0xF)
		result = getFieldFromValue(16, 19, value)
		require.Equal(t, uint64(0xF), result,
			"should only extract bits 16-19, ignoring other set bits")
	})
}
