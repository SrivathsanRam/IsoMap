package walking

import (
	"context"
	"math/bits"
)

func countWalkable(network *Network) int {
	if network == nil {
		return 0
	}
	total := 0
	for _, word := range network.WalkableBits {
		total += bits.OnesCount64(word)
	}
	return total
}

func connectedComponentStats(ctx context.Context, network *Network) (componentCount, largest int, err error) {
	cellCount := network.CellCount()
	visited := make([]uint64, (cellCount+63)/64)
	queue := make([]int, 0, 4096)
	for candidate := 0; candidate < cellCount; candidate++ {
		if candidate&65535 == 0 {
			if err := ctx.Err(); err != nil {
				return 0, 0, err
			}
		}
		mask := uint64(1) << uint(candidate&63)
		if !network.IsWalkable(candidate) || visited[candidate>>6]&mask != 0 {
			continue
		}
		componentCount++
		visited[candidate>>6] |= mask
		queue = append(queue[:0], candidate)
		componentSize := 0
		for head := 0; head < len(queue); head++ {
			index := queue[head]
			componentSize++
			row, column, _ := network.RowColumn(index)
			for deltaRow := -1; deltaRow <= 1; deltaRow++ {
				for deltaColumn := -1; deltaColumn <= 1; deltaColumn++ {
					if deltaRow == 0 && deltaColumn == 0 {
						continue
					}
					neighbor, ok := network.Index(row+deltaRow, column+deltaColumn)
					if !ok || !network.IsWalkable(neighbor) {
						continue
					}
					neighborMask := uint64(1) << uint(neighbor&63)
					if visited[neighbor>>6]&neighborMask != 0 {
						continue
					}
					visited[neighbor>>6] |= neighborMask
					queue = append(queue, neighbor)
				}
			}
		}
		if componentSize > largest {
			largest = componentSize
		}
	}
	return componentCount, largest, nil
}
