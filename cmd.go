package main

import (
	"fmt"

	"github.com/AlecAivazis/survey/v2"
)

var qs = []*survey.Question{
	{
		Name: "fileType",
		Prompt: &survey.Input{
			Message: "输入想要扫描的文件类型:",
			Default: "默认为任意类型",
		},
		Validate:  survey.Required,
		Transform: survey.Title,
	},
	{
		Name: "fileSize",
		Prompt: &survey.Input{
			Message: "选择文件最低大小:",
			Default: "需要指定单位(b,k,m,g)且不区分大小写",
		},
		Validate: survey.Required,
	},
	{
		Name: "fileNumber",
		Prompt: &survey.Input{
			Message: "显示扫描结果的数量",
			Default: "默认显示前三个",
		},
	},
}

func (op *Options) Cmd() {

	// the answers will be written to this struct
	// you can tag fields to match a specific name
	answers := struct {
		FileType  string `survey:"fileType"`
		Threshold string `survey:"fileSize"`
		ShowNum   uint32 `survey:"fileNumber"`
	}{}

	// perform the questions
	err := survey.Ask(qs, &answers)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	op.limit = answers.Threshold
	op.number = answers.ShowNum
	op.types = answers.FileType
}

func Select() {

}
