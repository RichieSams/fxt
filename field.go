package fxt

func getFieldFromValue(begin uint64, end uint64, value uint64) uint64 {
	mask := uint64((1 << (end - begin + 1)) - 1)
	return (value >> begin) & mask
}
