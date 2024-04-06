package utils

import (
	"container/heap"
	"fmt"
	"google.golang.org/grpc"
	"os"
	"strings"
)

func ListProtoMethods(desc grpc.ServiceDesc) (result string) {
	var methods = make([]string, len(desc.Methods))
	for i, method := range desc.Methods {
		methods[i] = method.MethodName
	}
	return fmt.Sprintf("[%s]", strings.Join(methods, ","))
}

func FileExists(filename string) bool {
	_, err := os.Stat(filename)
	return !os.IsNotExist(err)
}

// A PriorityQueue implements heap.Interface and holds elements of any comparable type.
type PriorityQueue[T any] struct {
	items []T
	less  func(i, j T) bool // Custom less function
}

func (pq *PriorityQueue[T]) Len() int { return len(pq.items) }

func (pq *PriorityQueue[T]) Less(i, j int) bool {
	return pq.less(pq.items[i], pq.items[j])
}

func (pq *PriorityQueue[T]) Swap(i, j int) {
	pq.items[i], pq.items[j] = pq.items[j], pq.items[i]
}

func (pq *PriorityQueue[T]) Push(x interface{}) {
	item := x.(T)
	pq.items = append(pq.items, item)
}

func (pq *PriorityQueue[T]) Pop() interface{} {
	old := pq.items
	n := len(old)
	item := old[n-1]
	pq.items = old[0 : n-1]
	return item
}

// MergeSortedLists merges multiple sorted lists into one sorted list using a custom less function.
func MergeSortedLists[T comparable](lists [][]T, less func(i, j T) bool, cap int) []T {
	var result []T
	pq := PriorityQueue[T]{less: less}
	heap.Init(&pq)

	// Initialize the priority queue with the first element of each list.
	for _, list := range lists {
		if len(list) > 0 {
			heap.Push(&pq, list[0])
		}
	}

	// Continuously extract the smallest item and add it to the result.
	for pq.Len() > 0 && len(result) < cap {
		item := heap.Pop(&pq).(T)
		result = append(result, item)
		// Remove the used item from its original list and push the next item into the heap.
		for i, list := range lists {
			if len(list) > 0 && item == list[0] {
				lists[i] = list[1:]
				if len(lists[i]) > 0 {
					heap.Push(&pq, lists[i][0])
				}
				break
			}
		}
	}

	return result
}
