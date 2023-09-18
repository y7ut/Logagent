package agent

type Collector struct {
	Style string `json:"style" gird_column:"日志规则" gird_sort:"4"`
	Path  string `json:"path" gird_column:"路径" gird_sort:"1"`
	Topic string `json:"topic" gird_column:"日志主题" gird_sort:"2"`
	Exist string `json:"_" gird_column:"是否存在" gird_sort:"4"`
}
