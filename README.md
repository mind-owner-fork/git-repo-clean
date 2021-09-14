**Command Usage:**

`git clean-repo`
> show usage

`git clean-repo -h[--help]`
> show help info

`git clean-repo -v[--verbose]`
> verbose output

`git clean-repo -V[--version]`
> show application version

`git clean-repo --scan --range=full/blobs/commits/trees/refs`
> scan range can be: full|blobs|commits|trees|refs|tags

`git clean-repo --scan --range=blobs --limit=10m --type=jpg --number=5`
> scan top 5 file(.jpg type, if have any) which size larger than 10M

`git clean-repo --delete filepath [OID]`
> without --scan option, app will execute clean action
> clean file by its filepath or OID(which type and size limitation are specified before)


**UI Usage:**

`git clean-repo -i[--interactive]`
> interactive with user end, guide user step by step
> this contains serveral commands below:

`git clean-repo --scan --range=blobs --number=N --limit=50m`
> show a list of top N biggest files(blobs), which type are unspecified but size larger than 50M
> hint user to specify file type by using commands

> hint user can upload bigfile to Gitee LFS server or just delete
> * let user choose which one to delete, or delete at on time
> `git clean-repo --delete filepath_0 [index_0]`
> `git clean-repo --delete filepath_1 [index_1]`
> `git clean-repo --delete filepath_2 [index_2]`
> * let user use LFS tool to upload bigfile to LFS server
> `git clean-repo --lfs --add url -- filepath`
> this will add the large file to remote LFS repo
> `git clean-repo --delete filepath`
> then delete from local repo


**LFS使用流程**

+ download and install
> https://github.com/git-lfs/git-lfs/releases
+ set up in machine
> git lfs install
+ track file
> git lfs track "*.mp4"
+ modify .gitattributes
> git add .gitattributes
+ normal git operation and the tracked file will upload to LFS server
> git add && git commit && git push

git lfs可以跟踪仓库中新加入的文件，而不会追踪历史提交中的文件
已经存在于提交历史中的大文件，如果想要使用LFS，需要用迁移：
+ mirate existing file in history
> git lfs migrate import --include="*.psd" --everything
+ push to remote
> git push origin main



**Code Struct**

+ main.go       | application layer
+ options.go    | app options, arguments
+ dialog.go     | interface with user
+ repository.go | maintain a repo
+ scan.go       | scan repo
+ actions.go    | execute some actions with repo
+ lfs.go        | handle LFS


## TODO
+ suppoort refs filter
+ support multi file type in one option
