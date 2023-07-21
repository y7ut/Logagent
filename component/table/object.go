package table

import (
	"fmt"
	"reflect"
	"sort"
	"strconv"

	"github.com/charmbracelet/bubbles/table"
)

type ObjectGrid[T any] struct {
	d       []T
	headers map[string]string // map[结构体Field]表头翻译
	sort    map[int]string    // 表头字段的排序
	define  map[string]string // 自定义表头
}

// NewMapGird 创建一个网格数据
func NewGrid[T any](items []T) *ObjectGrid[T] {
	return &ObjectGrid[T]{
		d:       items,
		headers: make(map[string]string, 0),
		sort:    make(map[int]string, 0),
	}
}

// 手动设置表头，如果不设置表头,则会自动获取，排序一定是按照对象的Field顺序的
func (o *ObjectGrid[T]) DefineHeader(define map[string]string) *ObjectGrid[T] {
	o.define = define
	return o
}

// 获取表头信息
func (o *ObjectGrid[T]) guessHeaders() {
	// 这里只是为了获取表头信息
	// 分析类型参数类型T的结构
	// 如果单纯的使用new(T) 在T本身就是一个指针的时候就有问题了
	// 所以我们要先判断是不是指针，如果是指针给一次机会
	eg := new(T)
	v := reflect.ValueOf(eg)
	var t reflect.Type
	if v.Kind() == reflect.Ptr {
		t = reflect.TypeOf(*eg)
	} else {
		t = reflect.TypeOf(eg)
	}
	// 再给一次机会，判断是否为指针
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return
	}

	o.sort = make(map[int]string, 0)
	unSort := make([]string, 0)
	for i := 0; i < t.NumField(); i++ {
		currentField := t.Field(i)
		currentFieldTag := t.Field(i).Tag.Get("gird_column")

		if currentFieldTag != "" {
			o.headers[currentField.Name] = currentFieldTag
		}

		currentFieldSort := t.Field(i).Tag.Get("gird_sort")
		if currentFieldSort != "" {
			fieldSort, err := strconv.Atoi(currentFieldSort)
			if err == nil {
				for i := fieldSort; ; i++ {
					if _, get := o.sort[i]; !get {
						o.sort[i] = currentField.Name
						break
					}
				}
			}
		} else {
			unSort = append(unSort, currentField.Name)
		}

		if defineTag, ok := o.define[currentField.Name]; ok {
			o.headers[currentField.Name] = defineTag
		}
	}

	// 最后吧没有指定排序的加上
	for _, v := range unSort {
		o.sort[len(o.sort)] = v
	}
}

func (o *ObjectGrid[T]) Headers() []string {
	if o.sort == nil || len(o.sort) == 0 {
		// 没有初始化排序的话，去初始化排序，有了排序才可以计算列表头
		o.guessHeaders()
	}
	sortColumns := make([]string, 0, len(o.sort))
	// 根据排序顺序，返回表头信息
	sortHelper := make([]int, 0, len(o.sort))
	for s := range o.sort {
		sortHelper = append(sortHelper, s)
	}
	sort.Ints(sortHelper)
	for _, v := range sortHelper {
		sortColumns = append(sortColumns, o.headers[o.sort[v]])
	}

	return sortColumns
}

func sortIntKeyMap[T any](m map[int]T) []T {
	sortColumns := make([]T, 0, len(m))
	// 根据排序顺序，返回表头信息
	sortHelper := make([]int, 0, len(m))
	for s := range m {
		sortHelper = append(sortHelper, s)
	}
	sort.Ints(sortHelper)
	for _, v := range sortHelper {
		sortColumns = append(sortColumns, m[v])
	}
	return sortColumns
}

func (o *ObjectGrid[T]) Rows() [][]any {
	s := make([][]any, 0)

	getIndex := func(s string) int {
		for i, v := range o.sort {
			if v == s {
				return i
			}
		}
		return -1
	}
	// fmt.Println(sortColumns)
	for _, m := range o.d {
		// m 是 T
		t := reflect.TypeOf(m)
		if t.Kind() == reflect.Ptr {
			t = t.Elem()
		}
		if t.Kind() != reflect.Struct {
			continue
		}

		tmp := make(map[int]any, 0)

		v := reflect.ValueOf(m)
		if v.Kind() == reflect.Ptr {
			v = v.Elem()
		}
		for i := 0; i < t.NumField(); i++ {
			field := v.Field(i)

			if field.Kind() == reflect.Ptr {
				field = field.Elem()
			}

			if field.Kind() > reflect.Complex128 && field.Kind() != reflect.String {
				continue
			}

			currentField := t.Field(i)

			if _, ok := o.headers[currentField.Name]; ok {
				if getIndex(currentField.Name) >= 0 {
					tmp[getIndex(currentField.Name)] = field.Interface()
				}
			}
		}

		s = append(s, sortIntKeyMap(tmp))
	}

	return s
}

// Render render一组Table所需的数据格式
func (o *ObjectGrid[T]) Render() (columns []table.Column, rows []table.Row) {
	gopHeaders := o.Headers()
	gopRows := o.Rows()
	columns = make([]table.Column, len(gopHeaders))
	rows = make([]table.Row, 0, len(gopRows))

	maxLen := make(map[int]int, len(gopHeaders))
	for _, v := range gopRows {
		tmpLine := make([]string, 0)
		for k := range v {
			tmpItem := fmt.Sprint(v[k])
			if len(tmpItem) > maxLen[k] {
				maxLen[k] = len(tmpItem)
			}
			tmpLine = append(tmpLine, tmpItem)
		}
		rows = append(rows, tmpLine)
	}
	for i, v := range gopHeaders {
		if len(v) > maxLen[i] {
			maxLen[i] = len(v)
		}
		columns[i] = table.Column{
			Title: v,
			Width: maxLen[i] + 3,
		}
	}
	return
}
