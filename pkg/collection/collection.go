package collection

import (
	"container/list"
	"reflect"
)

type item interface {
	any
}

// jian
type Collection[T item] struct {
	data      *list.List
	itemIsPtr bool
}

func New[T item](d []T) *Collection[T] {

	t := reflect.TypeOf(*new(T))
	var isPtr = t.Kind() == reflect.Ptr

	l := list.New()
	for _, item := range d {
		t := item
		l.PushBack(t)
	}

	return &Collection[T]{data: l, itemIsPtr: isPtr}
}

// Each can be used to iterate over the collection
func (c *Collection[T]) Each(f func(k int, i T)) *Collection[T] {
	for current, i := c.data.Front(), 0; current != nil; current, i = current.Next(), i+1 {
		var currentItem T = current.Value.(T)
		f(i, currentItem)
	}
	return c
}

// Map can be used to map the collection
func (c *Collection[T]) Map(f func(k int, i T) T) *Collection[T] {
	for current, i := c.data.Front(), 0; current != nil; current, i = current.Next(), i+1 {
		var currentItem T = current.Value.(T)
		currentItem = f(i, currentItem)
		current.Value = currentItem
	}
	return c
}

// Filter can be used to filter the collection only when the function returns true
func (c *Collection[T]) Filter(f func(i T) bool) *Collection[T] {
	match := list.New()
	for current := c.data.Back(); current != nil; current = current.Prev() {
		var currentItem T = current.Value.(T)
		if f(currentItem) {
			match.PushFront(currentItem)
		}
	}
	c.data = match
	return c
}

// Sort sorts the elements in the Collection using the provided function.
func (c *Collection[T]) Sort(f func(i, j T) bool) *Collection[T] {
	c.data = mergeSort(c.data, f)
	return c
}

// Len can be used to get the length
func (c *Collection[T]) Len() int {
	return c.data.Len()
}

// Value can be used to get the value slice
func (c *Collection[T]) Value() []T {
	lt := make([]T, 0)
	for current := c.data.Front(); current != nil; current = current.Next() {
		var currentItem T = current.Value.(T)
		lt = append(lt, currentItem)
	}
	return lt
}

func (c *Collection[T]) Merge(other *Collection[T]) {
	c.data.PushBackList(other.data)
}
