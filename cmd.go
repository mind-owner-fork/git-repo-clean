package main

import (
	"fmt"

	"github.com/AlecAivazis/survey/v2"
)

var qs = []*survey.Question{
	{
		Name: "fileType",
		Prompt: &survey.Input{
			Message: "选择要扫描的文件的类型:",
			Default: "*",
		},
		Validate:  survey.Required,
		Transform: survey.Title,
	},
	{
		Name: "fileSize",
		Prompt: &survey.Input{
			Message: "选择要扫描文件的最低大小:",
			Default: "1M",
		},
		Validate: survey.Required,
	},
	{
		Name: "fileNumber",
		Prompt: &survey.Input{
			Message: "选择要显示扫描结果的数量:",
			Default: "3",
		},
	},
}

func (op *Options) PreCmd() {

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

func PostCmd(list BlobList) []string {

	selected := []string{}
	options := []string{}

	for _, target := range list {
		options = append(options, target.objectName)
	}
	prompt := &survey.MultiSelect{
		Message:  "请选择你要删除的文件(可多选):",
		Options:  options,
		Help:     "空格键选中，向左全选，向右全不选",
		PageSize: 10,
	}
	survey.AskOne(prompt, &selected)

	return selected
}

func DoubleCheckCmd(list []string) []string {
	selected := []string{}

	// options := []string{}
	// options = list
	// for _, target := range list {
	// 	options = append(options, target)
	// }
	prompt := &survey.MultiSelect{
		Message: "以下是你要删除的文件ID，确定要删除吗?",
		Options: list,
	}

	survey.AskOne(prompt, &selected)

	return selected
}
