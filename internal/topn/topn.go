// internal/topn/topn.go
package topn

import "container/heap"

type Item struct {
	Size int64
	Path string
}

type minHeap []Item

/*
Min-heap implementation to keep top-N largest files.
- len(): returns the number of items in the heap
- less(i, j): compares the sizes of items at index i and j
- swap(i, j): swaps the items at index i and j
- push(x): adds a new item to the heap
- pop(): removes and returns the smallest item from the heap
*/
func (h minHeap) Len() int           { return len(h) }
func (h minHeap) Less(i, j int) bool { return h[i].Size < h[j].Size }
func (h minHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }
func (h *minHeap) Push(x any)        { *h = append(*h, x.(Item)) }
func (h *minHeap) Pop() any {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[:n-1]
	return x
}

// Keeper maintains the top N largest items.
type Keeper struct {
	N int
	h *minHeap
}

// Creates a new Keeper that keeps track of the top N largest items.
func NewKeeper(n int) *Keeper {
	h := &minHeap{}
	heap.Init(h)
	return &Keeper{N: n, h: h}
}

// Consider adds a new item to the Keeper if it is among the top N largest items seen so far.
// If the heap has fewer than N items, the item is added directly.
func (k *Keeper) Consider(it Item) {
	if k.h.Len() < k.N {
		heap.Push(k.h, it)
		return
	}

	if (*(k.h))[0].Size < it.Size {
		heap.Pop(k.h)
		heap.Push(k.h, it)
	}
}

// Returns the top N items in descending order.
func (k *Keeper) ItemsDesc() []Item {
	out := make([]Item, k.h.Len())
	for i := len(out) - 1; i >= 0; i-- {
		out[i] = heap.Pop(k.h).(Item)
	}

	// now out is ascending; print descending
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}

	return out
}
