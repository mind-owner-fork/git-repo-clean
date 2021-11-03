package main

import (
	"fmt"
	"strings"

	"github.com/AlecAivazis/survey/v2"
)

var qs = []*survey.Question{
	{
		Name: "fileType",
		Prompt: &survey.Input{
			Message: "选择要扫描的文件的类型，如：zip, png:",
			Default: "*",
			Help:    "默认无类型，即查找所有类型。如果想指定类型，则直接输入类型后缀名即可, 不需要加'.'",
		},
		Validate:  survey.Required,
		Transform: survey.Title,
	},
	{
		Name: "fileSize",
		Prompt: &survey.Input{
			Message: "选择要扫描文件的最低大小，如：1M, 1g:",
			Default: "1M",
			Help:    "大小数值需要单位，如: 10K. 可选单位有B,K,M,G, 且不区分大小写",
		},
		Validate: survey.Required,
	},
	{
		Name: "fileNumber",
		Prompt: &survey.Input{
			Message: "选择要显示扫描结果的数量，默认3:",
			Default: "3",
			Help:    "默认显示前3个，单页最大显示为10行，所以最好不超过10",
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
	err := survey.Ask(qs, &answers, survey.WithHelpInput('?'))
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	op.limit = answers.Threshold
	op.number = answers.ShowNum
	op.types = answers.FileType
}

func MultiSelectCmd(list BlobList) []string {

	selected := []string{}
	options := []string{}

	for _, target := range list {
		opt := target.oid + ": " + target.objectName + "\n"
		options = append(options, opt)
	}
	prompt := &survey.MultiSelect{
		Message:  "请选择你要删除的文件(可多选):\n",
		Options:  options,
		PageSize: 10,
		Help:     "使用键盘的上下左右，可进行上下换行、全选、全取消，使用空格建选中单个，使用Enter键确认选择",
	}
	survey.AskOne(prompt, &selected, survey.WithHelpInput('?'))

	return selected
}

func Confirm(list []string) (bool, []string) {
	ok := false
	results := []string{}

	prompt := &survey.Confirm{
		Message: "以上是你要删除的文件，确定要删除吗?\n",
	}
	for _, item := range list {
		fmt.Println(item)
	}
	survey.AskOne(prompt, &ok)

	// turn back to name oid only
	for _, item := range list {
		name := strings.Split(item, ":")[0]
		results = append(results, name)
	}

	return ok, results
}
