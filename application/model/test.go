package model

import (
	"teamide/application/base"
)

type TestModel struct {
	Name           string        `json:"name,omitempty" yaml:"name,omitempty"`                 // 名称，同一个应用中唯一
	Comment        string        `json:"comment,omitempty" yaml:"name,omitempty"`              // 注释说明
	Description    string        `json:"description,omitempty" yaml:"description,omitempty"`   // 注释说明
	ThreadNumber   int           `json:"threadNumber,omitempty" yaml:"threadNumber,omitempty"` // 线程数量
	ForNumber      int           `json:"forNumber,omitempty" yaml:"forNumber,omitempty"`       // 循环次数
	Steps          []ServiceStep `json:"steps,omitempty" yaml:"steps,omitempty"`
	TestJavascript string        `json:"-" yaml:"-"` // Javascript
}

func TextToTestModel(namePath string, text string) (model *TestModel, err error) {
	var modelMap map[string]interface{}
	var name string
	name, modelMap, err = TextToModelMap(namePath, text)
	if err != nil {
		return
	}
	model = &TestModel{
		Name: name,
	}
	model.Steps, err = getStepsByValue(modelMap["steps"])
	if err != nil {
		return
	}
	delete(modelMap, "steps")

	err = base.ToBean([]byte(base.ToJSON(modelMap)), model)
	if err != nil {
		return
	}
	return
}