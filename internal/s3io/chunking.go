package s3io

func AlignRange(offset, size, block, prefetch int64) (start, end int64) {
	if block <= 0 {
		block = 8 << 20
	}
	if prefetch < block {
		prefetch = block
	}
	start = (offset / block) * block
	end = start + prefetch - 1
	if max := offset + size - 1; max < end {
		end = max
	}
	return
}
