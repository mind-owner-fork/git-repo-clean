package main

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/AlecAivazis/survey/v2/terminal"
)

var qs = []*survey.Question{
	{
		Name: "fileType",
		Prompt: &survey.Input{
			Message: LocalSprintf("select the type of file to scan, such as zip, png:"),
			Default: "*",
			Help:    LocalSprintf("default is all types. If you want to specify a type, you can directly enter the type suffix without prefix '.'"),
		},
		Validate: func(ans interface{}) error {
			str, ok := ans.(string)
			if !ok || len(str) > 10 {
				return errors.New(LocalSprintf("filetype error one"))
			}
			match, _ := regexp.MatchString(`^[a-zA-Z1-9]+[.]?[a-zA-Z1-9]*$|^[a-zA-Z1-9]+$`, str)
			if !match && str != "*" {
				return errors.New(LocalSprintf("filetype error two"))
			}
			return nil
		},
	},
	{
		Name: "fileSize",
		Prompt: &survey.Input{
			Message: LocalSprintf("select the minimum size of the file to scan, such as 1m, 1G:"),
			Default: "1M",
			Help:    LocalSprintf("the size value needs units, such as 10K. The optional units are B, K, m and G, and are not case sensitive"),
		},
		Validate: func(ans interface{}) error {
			str, ok := ans.(string)
			if !ok {
				return errors.New(LocalSprintf("filesize error one"))
			}
			match, _ := regexp.MatchString(`^[1-9]+[0-9]*[bBkKmMgG]$`, str)
			if !match {
				return errors.New(LocalSprintf("filesize error two"))
			}
			return nil
		},
	},
	{
		Name: "fileNumber",
		Prompt: &survey.Input{
			Message: LocalSprintf("select the number of scan results to display, the default is 3:"),
			Default: "3",
			Help:    LocalSprintf("the default display is the first 3. The maximum page size is 10 rows, so it is best not to exceed 10."),
		},
		Validate: func(ans interface{}) error {
			str, ok := ans.(string)
			if !ok {
				return errors.New(LocalSprintf("filenumber error one"))
			}
			match, _ := regexp.MatchString(`^[1-9]+[0-9]*$`, str)
			if !match {
				return errors.New(LocalSprintf("filenumber error two"))
			}
			return nil
		},
	},
}

func (op *Options) SurveyCmd() error {

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
		return err
	}
	op.limit = answers.Threshold
	op.number = answers.ShowNum
	op.types = answers.FileType

	return nil
}

func MultiSelectCmd(list BlobList) []string {

	selected := []string{}
	targets := []string{}

	for _, item := range list {
		ele := item.oid + ": " + item.objectName + "\n"
		targets = append(targets, ele)
	}
	prompt := &survey.MultiSelect{
		Message:  LocalSprintf("multi select message") + "\n",
		Options:  targets,
		PageSize: 10,
		Help:     LocalSprintf("multi select help info"),
	}
	err := survey.AskOne(prompt, &selected, survey.WithHelpInput('?'))
	if err != nil {
		if err == terminal.InterruptErr {
			PrintLocalWithRedln("process interrupted")
			os.Exit(1)
		}
	}
	return selected
}

func Confirm(list []string) (bool, []string) {
	ok := false
	results := []string{}

	prompt := &survey.Confirm{
		Message: LocalSprintf("confirm message") + "\n",
	}

	err := survey.AskOne(prompt, &ok)
	if err != nil {
		if err == terminal.InterruptErr {
			PrintLocalWithRedln("process interrupted")
			os.Exit(1)
		}
	}

	// turn back to name oid only
	for _, item := range list {
		name := strings.Split(item, ":")[0]
		results = append(results, name)
	}

	return ok, results
}

func AskForMigrateToLFS() bool {
	ok := false

	prompt := &survey.Confirm{
		Message: LocalSprintf("ask for migrating big file into LFS") + "\n",
	}
	err := survey.AskOne(prompt, &ok)
	if err != nil {
		if err == terminal.InterruptErr {
			PrintLocalWithRedln("process interrupted")
			os.Exit(1)
		}
	}

	return ok
}

func AskForOverride() bool {
	ok := false

	prompt := &survey.Confirm{
		Message: LocalSprintf("ask for override message") + "\n",
	}
	err := survey.AskOne(prompt, &ok)
	if err != nil {
		if err == terminal.InterruptErr {
			PrintLocalWithRedln("process interrupted")
			os.Exit(1)
		}
	}

	return ok
}

func AskForUpdate() bool {
	ok := false
	fmt.Println()
	prompt := &survey.Confirm{
		Message: LocalSprintf("ask for update message") + "\n",
	}
	err := survey.AskOne(prompt, &ok)
	if err != nil {
		if err == terminal.InterruptErr {
			PrintLocalWithRedln("process interrupted")
			os.Exit(1)
		}
	}

	return ok
}
