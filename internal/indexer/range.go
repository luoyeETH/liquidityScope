package indexer

import "fmt"

// BlockRange represents an inclusive block range.
type BlockRange struct {
	From uint64
	To   uint64
}

// SplitRange splits a block range into batches of size batchSize.
func SplitRange(from, to, batchSize uint64) ([]BlockRange, error) {
	if batchSize == 0 {
		return nil, fmt.Errorf("batch size must be greater than zero")
	}
	if to < from {
		return nil, fmt.Errorf("to block must be >= from block")
	}

	ranges := make([]BlockRange, 0)
	start := from
	for start <= to {
		remaining := to - start + 1
		var end uint64
		if remaining <= batchSize {
			end = to
		} else {
			end = start + batchSize - 1
		}
		ranges = append(ranges, BlockRange{From: start, To: end})
		if end == to {
			break
		}
		start = end + 1
	}

	return ranges, nil
}
