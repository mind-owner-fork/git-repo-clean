package main

import (
	"fmt"
	"io"

	"github.com/cloudfoundry/jibber_jabber"

	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

func init() {
	initEnglish()
	initChinese()
}

func initEnglish() {
	// main.go
	message.SetString(language.English, "parse Option error", "Parse Option error")
	message.SetString(language.English, "couldn't open Git repository", "Couldn't open Git repository")
	message.SetString(language.English, "couldn't find Git execute program", "Couldn't find Git execute program")
	message.SetString(language.English, "sorry, this tool requires Git version at least 2.24.0",
		"Sorry, this tool requires Git version at least 2.24.0")
	message.SetString(language.English, "couldn't support running in bare repository", "Couldn't support running in bare repository")
	message.SetString(language.English, "couldn't support running in shallow repository", "Couldn't support running in shallow repository")
	message.SetString(language.English, "scanning repository error: %s", "Scanning repository error: %s")
	message.SetString(language.English, "no files were scanned",
		"According to the filter conditions you selected, no files were filtered out. Please adjust the filter criteria and try again.")
	message.SetString(language.English, "no files were selected",
		"You haven't selected any files. Please select at least one file")
	message.SetString(language.English, "operation aborted", "The operation has been aborted. Please reconfirm the file and try again.")
	message.SetString(language.English, "cleaning completed", "Local repository cleaning up completed!")
	message.SetString(language.English, "current repository size", "Current repository size: ")
	message.SetString(language.English, "execute force push",
		"The following two commands will be executed, then the remote commit will be overwritten:")
	message.SetString(language.English, "suggest operations header",
		"Since the history of the local repository has been modified, if there is no new commit, it is recommended to complete the following work first:")
	message.SetString(language.English, "1. (Done!)", "1. (Done!) update remote repository. Push local cleaned repository to remote repository:")
	message.SetString(language.English, "1. (Undo)", "1. (Undo) update remote repository. Push local cleaned repository to remote repository:")
	message.SetString(language.English, "2. (Undo)",
		"2. (Undo) clean up the remote repository. After successful push, please go to your corresponding repository management page to perform GC operation.")
	message.SetString(language.English, "3. (Undo)",
		"3. (Undo) process the associated repository. Process other repository in the clone under the same remote repository to ensure that the same file won't be submitted to the remote repository again. ")
	message.SetString(language.English, "gitee GC page link", "If you are a Gitee repository and have admin right, please check the link: ")
	message.SetString(language.English, "for detailed documentation, see", "For detailed documentation, see: ")
	message.SetString(language.English, "suggest operations done", "After completing the above three steps, Congratulations, all the cleaning work has been done!")
	message.SetString(language.English, "introduce GIT LFS",
		"If you need to store large files, please use the GIT LFS to avoid the size of the repository exceed the limit again.")
	message.SetString(language.English, "for the use of Gitee LFS, see", "For the use of Gitee LFS, see: ")
	message.SetString(language.English, "init repo filter error", "Init repo Filter error")

	// options.go
	message.SetString(language.English, "help info", Usage)
	message.SetString(language.English, "option format error: %s", "Option format error: %s")
	message.SetString(language.English, "build version: %s", "Build version: %s")
	message.SetString(language.English, "single parameter is invalid", "This single parameter is invalid, please combine with other parameter.")
	// parser.go
	message.SetString(language.English, "unsupported filechange type", "Unsupported filechange type")
	message.SetString(language.English, "nested tags error",
		"The operation has been aborted because nested tags. It is recommended to use the '--branch=<branch>' option to specify a single branch.")
	message.SetString(language.English, "no match mark id", "No match mark id")
	message.SetString(language.English, "no match original-oid", "No match original-oid")
	message.SetString(language.English, "no match data size", "No match data size")
	message.SetString(language.English, "failed to write data", "Failed to write data")
	message.SetString(language.English, "start to clean up specified files",
		"Start to clean up the specified file from the history (if the repository is too large, the execution time will be long, please wait a few minutes)...")
	message.SetString(language.English, "run git-fast-import process failed", "Run git-fast-import process failed")
	// utils.go
	message.SetString(language.English, "expected a value followed by --limit option, but you are: %s",
		"Expected a value followed by --limit option, but you are: %s")
	message.SetString(language.English, "expected format: --limit=<n>b|k|m|g, but you are: --limit=%s",
		"Expected format: --limit=<n>b|k|m|g, but you are: --limit=%s")
	message.SetString(language.English, "scan done!", "Scan done!")
	message.SetString(language.English, "note that there may be multiple versions of the same file",
		"Note that there may be multiple versions of the same file, which are the main reasons for wasting git repository storage")
	message.SetString(language.English, "please delete selectively according to its Blob ID",
		"Please delete selectively according to its Blob ID, if you are sure that all files can be deleted, select all.")
	// repository.go
	message.SetString(language.English, "start scanning", "Start scanning(if the repository is too large, the scanning time will be long, please wait a few minutes)...")
	message.SetString(language.English, "run GetBlobName error: %s", "Run GetBlobName error: %s")
	message.SetString(language.English, "run getblobsize error: %s", "Run getblobsize error: %s")
	message.SetString(language.English, "expected blob object type, but got: %s", "Expected blob object type, but got: %s")
	message.SetString(language.English, "could not run 'git rev-parse --is-bare-repository': %s", "Could not run 'git rev-parse --is-bare-repository': %s")
	message.SetString(language.English, "could not run 'git rev-parse --is-shallow-repository': %s", "Could not run 'git rev-parse --is-shallow-repository': %s")
	message.SetString(language.English, "could not run 'git reflog show': %s", "Could not run 'git reflog show': %s")
	message.SetString(language.English, "could not run 'git lfs version': %s", "Could not run 'git lfs version': %s")
	message.SetString(language.English, "could not run 'git version': %s", "Could not run 'git version': %s")
	message.SetString(language.English, "match git version wrong", "Match git version wrong")
	message.SetString(language.English, "could not run 'git symbolic-ref HEAD --short': %s", "Could not run 'git symbolic-ref HEAD --short': %s")
	message.SetString(language.English, "could not run 'git status'", "Could not run 'git status'")
	message.SetString(language.English, "git status clean", "git status clean")
	message.SetString(language.English, "could not run 'du -hs .git'", "Could not run 'du -hs .git'")
	message.SetString(language.English, "start preparing repository data", "Start preparing repository data")
	message.SetString(language.English, "backup canceled", "Backup canceled")
	message.SetString(language.English, "start backup", "Start backup...")
	message.SetString(language.English, "clone error", "git clone --no-local error")
	message.SetString(language.English, "run filepach.Abs error", "Run filepach.Abs error")

	message.SetString(language.English, "backup done! Backup file path is: %s", "Backup done! Backup file path is: %s")
	message.SetString(language.English, "push failed",
		"Push failed. You may not have permission to push, or the repository does not have a remote repository")
	message.SetString(language.English, "done", "Done")
	message.SetString(language.English, "file cleanup is complete. Start cleaning the repository", "File cleanup is complete. Start cleaning the repository...")

	// cmd.go
	message.SetString(language.English, "select the type of file to scan, such as zip, png:", "Select the type of file to scan, such as zip, png:")
	message.SetString(language.English, "default is all types. If you want to specify a type, you can directly enter the type suffix without prefix '.'",
		"Default is all types. If you want to specify a type, you can directly enter the type suffix without prefix '.'")
	message.SetString(language.English, "filetype error one", "Sorry, the type name you entered is too long, more than 10 characters")
	message.SetString(language.English, "filetype error two", "The type must be a letter. It can contain '.' in the middle, but it doesn't need to contain '.' at the beginning")

	message.SetString(language.English, "select the minimum size of the file to scan, such as 1m, 1G:", "Select the minimum size of the file to scan, such as 1m, 1G:")
	message.SetString(language.English, "the size value needs units, such as 10K. The optional units are B, K, m and G, and are not case sensitive",
		"The size value needs units, such as 10K. The optional units are B, K, m and G, and are not case sensitive")
	message.SetString(language.English, "filesize error one", "input error")
	message.SetString(language.English, "filesize error two", "Must be a combination of numbers + unit characters (B, K, m, g), and the units are not case sensitive")

	message.SetString(language.English, "select the number of scan results to display, the default is 3:", "Select the number of scan results to display. The default value is 3:")
	message.SetString(language.English, "the default display is the first 3. The maximum page size is 10 rows, so it is best not to exceed 10.",
		"The default display is the first 3. The maximum page size is 10 rows, so it is best not to exceed 10.")
	message.SetString(language.English, "filenumber error one", "input error")
	message.SetString(language.English, "filenumber error two", "Must be a pure number")

	message.SetString(language.English, "multi select message", "Please select the file you want to delete (multiple choices are allowed):")
	message.SetString(language.English, "multi select help info", "Use <Up/Down> arrows to move, <space> to select, <right> to all, <left> to none, type to filter, ? for more help")

	message.SetString(language.English, "confirm message", "The above is the file you want to delete. Are you sure you want to delete it ?")
	message.SetString(language.English, "ask for backup message", "Do you want to back up the repository before deleting your files ?")
	message.SetString(language.English, "ask for override message", "A folder with the same name exists in the current directory. Do you want to overwrite it (if no, will cancel the backup) ?")
	message.SetString(language.English, "ask for update message", "Your local commit history has changed. Do you want to force push to the remote repository now ?")
	message.SetString(language.English, "process interrupted", "process interrupted")

	message.SetString(language.English, "convert uint error: %s", "Convert uint error: %s")
	message.SetString(language.English, "parse uint error: %s", "Parse uint error: %s")

}

func initChinese() {
	// main.go
	message.SetString(language.Chinese, "parse Option error", "解析参数错误。")
	message.SetString(language.Chinese, "couldn't open Git repository", "无法打开Git仓库。")
	message.SetString(language.Chinese, "couldn't find Git execute program", "无法找到Git可执行文件。")
	message.SetString(language.Chinese, "sorry, this tool requires Git version at least 2.24.0",
		"抱歉，这个工具需要Git的最低版本为 2.24.0")
	message.SetString(language.Chinese, "couldn't support running in bare repository", "不支持在裸仓库中执行。")
	message.SetString(language.Chinese, "couldn't support running in shallow repository", "不支持在浅仓库中执行。")
	message.SetString(language.Chinese, "scanning repository error: %s", "扫描仓库出错: %s")
	message.SetString(language.Chinese, "no files were scanned", "根据你所选择的筛选条件，没有扫描到任何文件，请调整筛选条件再试一次。")
	message.SetString(language.Chinese, "no files were selected", "你没有选择任何文件，请至少选择一个文件。")
	message.SetString(language.Chinese, "operation aborted", "操作已中止，请重新确认文件后再次尝试。")
	message.SetString(language.Chinese, "cleaning completed", "本地仓库清理完成！")
	message.SetString(language.Chinese, "current repository size", "当前仓库大小：")
	message.SetString(language.Chinese, "execute force push", "将会执行如下两条命令，远端的的提交将会被覆盖:")
	message.SetString(language.Chinese, "suggest operations header", "由于本地仓库的历史已经被修改，如果没有新的提交，建议先完成如下工作：")
	message.SetString(language.Chinese, "1. (Done!)", "1. (已完成！)更新远程仓库。将本地清理后的仓库推送到远程仓库：")
	message.SetString(language.Chinese, "1. (Undo)", "1. (待完成)更新远程仓库。将本地清理后的仓库推送到远程仓库：")
	message.SetString(language.Chinese, "2. (Undo)", "2. (待完成)清理远程仓库。提交成功后，请前往你对应的仓库管理页面，执行GC操作。")
	message.SetString(language.Chinese, "3. (Undo)", "3. (待完成)处理关联仓库。处理同一个远程仓库下clone的其它仓库，确保不会将同样的文件再次提交到远程仓库。")
	message.SetString(language.Chinese, "gitee GC page link", "如果是 Gitee 仓库，且有管理权限，请点击链接: ")
	message.SetString(language.Chinese, "for detailed documentation, see", "详细文档请参阅: ")
	message.SetString(language.Chinese, "suggest operations done", "完成以上三步后，恭喜你，所有的清理工作已经完成！")
	message.SetString(language.Chinese, "introduce GIT LFS", "如果有大文件的存储需求，请使用Git-LFS功能，避免仓库体积再次膨胀。")
	message.SetString(language.Chinese, "for the use of Gitee LFS, see", "Gite LFS 的使用请参阅：")
	message.SetString(language.Chinese, "init repo filter error", "初始化仓库过滤器失败")

	// options.go
	message.SetString(language.Chinese, "help info", Usage_ZH)
	message.SetString(language.Chinese, "option format error: %s", "选项格式错误: %s")
	message.SetString(language.Chinese, "build version: %s", "版本编号: %s")
	message.SetString(language.Chinese, "single parameter is invalid", "该单项参数无效，请结合其它参数使用")
	// parser.go
	message.SetString(language.Chinese, "unsupported filechange type", "不支持的filechange类型")
	message.SetString(language.Chinese, "nested tags error", "处理过程中断，因为仓库中存在嵌套式tag，建议使用'--branch=<branch>'参数指定单个分支。")
	message.SetString(language.Chinese, "no match mark id", "没有匹配到mark id字段")
	message.SetString(language.Chinese, "no match original-oid", "没有匹配到original oid字段")
	message.SetString(language.Chinese, "no match data size", "没有匹配到数据大小字段")
	message.SetString(language.Chinese, "failed to write data", "写数据失败")
	message.SetString(language.Chinese, "start to clean up specified files", "开始从历史中清理指定的文件(如果仓库过大，执行时间会比较长，请耐心等待)...")
	message.SetString(language.Chinese, "run git-fast-import process failed", "运行git fast-import过程出错")
	// utils.go
	message.SetString(language.Chinese, "expected a value followed by --limit option, but you are: %s", "'--limit'选项后面需要跟一个数值，但是你给的是: %s")
	message.SetString(language.Chinese, "expected format: --limit=<n>b|k|m|g, but you are: --limit=%s", "希望的格式为: --limit=<n>b|k|m|g, 但是你给的是: --limit=%s")
	message.SetString(language.Chinese, "scan done!", "扫描完成!")
	message.SetString(language.Chinese, "note that there may be multiple versions of the same file", "注意，同一个文件因为版本不同可能会存在多个，这些是占用 Git 仓库存储的主要原因。")
	message.SetString(language.Chinese, "please delete selectively according to its Blob ID", "请根据需要，通过其对应的ID进行选择性删除，如果确认文件可以全部删除，全选即可。")
	// repository.go
	message.SetString(language.Chinese, "start scanning", "开始扫描(如果仓库过大，扫描时间会比较长，请耐心等待)...")
	message.SetString(language.Chinese, "run GetBlobName error: %s", "运行 GetBlobName 错误: %s")
	message.SetString(language.English, "run getblobsize error: %s", "运行 getblobsize 错误: %s")
	message.SetString(language.Chinese, "expected blob object type, but got: %s", "期望blob类型数据，但实际得到: %s")
	message.SetString(language.Chinese, "could not run 'git rev-parse --is-bare-repository': %s", "无法运行'git rev-parse --is-bare-repository': %s")
	message.SetString(language.Chinese, "could not run 'git rev-parse --is-shallow-repository': %s", "无法运行'git rev-parse --is-shallow-repository': %s")
	message.SetString(language.Chinese, "could not run 'git reflog show': %s", "无法运行'git reflog show': %s")
	message.SetString(language.Chinese, "could not run 'git lfs version': %s", "无法运行'git lfs version': %s")
	message.SetString(language.Chinese, "could not run 'git version': %s", "无法运行'git version': %s")
	message.SetString(language.Chinese, "match git version wrong", "Git版本号匹配错误")
	message.SetString(language.Chinese, "could not run 'git symbolic-ref HEAD --short': %s", "无法运行'git symbolic-ref HEAD --short': %s")
	message.SetString(language.Chinese, "could not run 'git status'", "无法运行'git status'")
	message.SetString(language.Chinese, "git status clean", "git status为空")
	message.SetString(language.Chinese, "could not run 'du -hs .git'", "无法运行'du -hs .git'")
	message.SetString(language.Chinese, "start preparing repository data", "开始准备仓库数据")
	message.SetString(language.Chinese, "backup canceled", "已取消备份")
	message.SetString(language.Chinese, "start backup", "开始备份...")
	message.SetString(language.Chinese, "clone error", "git clone --no-local 错误")
	message.SetString(language.Chinese, "run filepach.Abs error", "运行 filepach.Abs 错误")
	message.SetString(language.Chinese, "backup done! Backup file path is: %s", "备份完毕! 备份文件路径为：%s")
	message.SetString(language.Chinese, "Push failed", "推送失败，可能是没有权限推送，或者该仓库没有设置远程仓库。")
	message.SetString(language.Chinese, "done", "完成")
	message.SetString(language.Chinese, "file cleanup is complete. Start cleaning the repository", "文件清理完毕，开始清理仓库...")

	// cmd.go
	message.SetString(language.Chinese, "select the type of file to scan, such as zip, png:", "选择要扫描的文件的类型，如：zip, png:")
	message.SetString(language.Chinese, "filetype help message", "默认无类型，即查找所有类型。如果想指定类型，则直接输入类型后缀名即可, 不需要加'.'")
	message.SetString(language.Chinese, "filetype error one", "抱歉，输入的类型名过长，超过10个字符")
	message.SetString(language.Chinese, "filetype error two", "类型必须是字母，中间可以包含'.'，但是开头不需要包含'.'")

	message.SetString(language.Chinese, "select the minimum size of the file to scan, such as 1m, 1G:", "选择要扫描文件的最低大小，如：1M, 1g:")
	message.SetString(language.Chinese, "filesize help message", "大小数值需要单位，如: 10K. 可选单位有B,K,M,G, 且不区分大小写")
	message.SetString(language.Chinese, "filesize error one", "输入错误")
	message.SetString(language.Chinese, "filesize error two", "必须以数字+单位字符(b,k,m,g)组合，且单位不区分大小写")

	message.SetString(language.Chinese, "select the number of scan results to display, the default is 3:", "选择要显示扫描结果的数量，默认值是3:")
	message.SetString(language.Chinese, "filenumber help message", "默认显示前3个，单页最大显示为10行，所以最好不超过10。")
	message.SetString(language.Chinese, "filenumber error one", "输入错误")
	message.SetString(language.Chinese, "filenumber error two", "必须是纯数字")

	message.SetString(language.Chinese, "multi select message", "请选择你要删除的文件(可多选):")
	message.SetString(language.Chinese, "multi select help info", "使用键盘的上下左右，可进行上下换行、全选、全取消，使用空格建选中单个，使用Enter键确认选择。")

	message.SetString(language.Chinese, "confirm message", "以上是你要删除的文件，确定要删除吗?")
	message.SetString(language.Chinese, "ask for backup message", "在删除你的文件之前，是否需要备份仓库?")
	message.SetString(language.Chinese, "ask for override message", "当前目录下存在同名文件夹，是否需要覆盖(回答否，则取消备份)?")
	message.SetString(language.Chinese, "ask for update message", "你的本地提交历史已经更改，是否现在强制推送到远程仓库？")
	message.SetString(language.Chinese, "process interrupted", "过程中断")

	message.SetString(language.Chinese, "convert uint error: %s", "转换大小单位出错: %s")
	message.SetString(language.Chinese, "parse uint error: %s", "解析无符号整数出错: %s")

}

// find local languange type. LC_ALL > LANG > LANGUAGE
func Local() language.Tag {
	// when LC_ALL && LANG is none, throw panic
	userLanguage, err := jibber_jabber.DetectLanguage()
	if err != nil {
		fmt.Println("try to set: export LC_ALL=zh_CN.UTF-8")
		panic(err)
	}
	// fix LC_ALL=C.UTF-8
	if userLanguage == "C" {
		userLanguage = "zh"
	}
	tagLanguage := language.Make(userLanguage)
	return tagLanguage
}

// set languange
func SetLang() language.Tag {
	return Local()
}

// local printer
func LocalPrinter() *message.Printer {
	tag := SetLang()
	return message.NewPrinter(tag)
}

// local fmt.Sprintf
func LocalSprintf(key message.Reference, a ...interface{}) string {
	return LocalPrinter().Sprintf(key, a)
}

// local fmt.Fprintf
func LocalFprintf(w io.Writer, key message.Reference, a ...interface{}) (int, error) {
	return LocalPrinter().Fprintf(w, key, a)
}

// local fmt.Printf
func LocalPrintf(key message.Reference, a ...interface{}) (int, error) {
	return LocalPrinter().Printf(key, a)
}

/*  PRINT WITH COLOR */

func PrintLocalWithRed(key message.Reference, a ...interface{}) {
	f := LocalSprintf(key, a)
	PrintRed(f)
}

func PrintLocalWithGreen(key message.Reference, a ...interface{}) {
	f := LocalSprintf(key, a)
	PrintGreen(f)
}

func PrintLocalWithYellow(key message.Reference, a ...interface{}) {
	f := LocalSprintf(key, a)
	PrintYellow(f)
}

func PrintLocalWithBlue(key message.Reference, a ...interface{}) {
	f := LocalSprintf(key, a)
	PrintBlue(f)
}

func PrintLocalWithPlain(key message.Reference, a ...interface{}) {
	f := LocalSprintf(key, a)
	PrintPlain(f)
}

func PrintLocalWithRedln(key message.Reference, a ...interface{}) {
	f := LocalSprintf(key, a)
	PrintRedln(f)
}

func PrintLocalWithGreenln(key message.Reference, a ...interface{}) {
	f := LocalSprintf(key, a)
	PrintGreenln(f)
}

func PrintLocalWithYellowln(key message.Reference, a ...interface{}) {
	f := LocalSprintf(key, a)
	PrintYellowln(f)
}

func PrintLocalWithBlueln(key message.Reference, a ...interface{}) {
	f := LocalSprintf(key, a)
	PrintBlueln(f)
}

func PrintLocalWithPlainln(key message.Reference, a ...interface{}) {
	f := LocalSprintf(key, a)
	PrintPlainln(f)
}
