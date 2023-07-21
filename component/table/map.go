package table

import (
	"fmt"

	"github.com/charmbracelet/bubbles/table"
)

type GridAccess interface {
	~int | ~uint | ~int8 | ~uint8 | ~int16 | ~uint16 | ~int32 | ~uint32 | ~int64 | ~uint64 | ~uintptr | ~float32 | ~float64 | ~string
}

// MapGrid 网格数据
type MapGrid[H, T GridAccess] struct {
	d       []map[H]T
	headers []H
}

// NewMapGrid 创建一个网格数据
func NewMapGrid[H, T GridAccess](items []map[H]T) *MapGrid[H, T] {
	return &MapGrid[H, T]{
		d:       items,
		headers: make([]H, 0),
	}
}

// 手动设置表头，如果不设置表头,则会自动获取，但是不保证顺序,可恶...
func (i *MapGrid[H, T]) SetHeaders(headers ...H) *MapGrid[H, T] {
	i.headers = headers
	return i
}

// 获取表头信息
func (i *MapGrid[H, T]) Headers() []H {
	if len(i.headers) > 0 {
		return i.headers
	}
	set := make(map[H]bool)
	for _, m := range i.d {
		for k := range m {
			set[k] = true
		}
	}
	s := MapKeys(set)
	return s
}

// Index 获取某一个表头的下标位置
func (i *MapGrid[H, T]) Index(header H) (result int, err error) {
	err = fmt.Errorf("not found")
	for k, v := range i.Headers() {
		if v == header {
			result, err = k, nil
		}
	}
	return result, err
}

// Rows 获取全部的行数据，并按照表头的index的来排序
func (i *MapGrid[H, T]) Rows() [][]T {
	s := make([][]T, 0)
	headers := i.Headers()
	count := len(headers)

	for _, m := range i.d {
		// m 是 map[H]T
		line := make([]T, count)
		for k := 0; k < count; k++ {
			if v, ok := m[headers[k]]; ok {
				line[k] = v
			}
		}
		s = append(s, line)
	}
	return s
}

// Render render一组Table所需的数据格式
func (i *MapGrid[H, T]) Render() (columns []table.Column, rows []table.Row) {
	gmpHeaders := i.Headers()
	gmpRows := i.Rows()
	columns = make([]table.Column, 0, len(gmpHeaders))
	rows = make([]table.Row, 0, len(gmpRows))
	for _, v := range gmpHeaders {

		header := fmt.Sprint(v)

		column := table.Column{
			Title: header,
			Width: len(header) * 2,
		}
		columns = append(columns, column)
	}

	for _, v := range gmpRows {
		tmpLine := make([]string, 0)
		for k := range v {
			tmpLine = append(tmpLine, fmt.Sprint(v[k]))
		}
		rows = append(rows, tmpLine)
	}
	return
}
