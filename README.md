**Command Usage:**

`git clean-repo`
> show usage

`git clean-repo -h[--help]`
> show help info

`git clean-repo -v[--verbose]`
> verbose output

`git clean-repo -V[--version]`
> show application version

<!-- `git clean-repo --scan --range=full/blobs/commits/trees/refs` -->
<!-- > scan range can be: full|blobs|commits|trees|refs|tags -->

`git clean-repo --scan --limit=10m --type=jpg --number=5`
> scan top 5 file(.jpg type, if have any) which size larger than 10M

`git clean-repo --delete filepath [OID]`
> without --scan option, app will execute clean action
> clean file by its filepath or OID(which type and size limitation are specified before)


**UI Usage:**

`git clean-repo -i[--interactive]`
> interactive with user end, guide user step by step
> this contains serveral commands below:

`git clean-repo --scan --number=N --limit=50m`
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
# https://help.aliyun.com/document_detail/206889.html
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
# https://help.aliyun.com/document_detail/206890.html?spm=a2c4g.11186623.0.nextDoc.778d3f107TbPkx
+ mirate existing file in history
> git lfs migrate import --include="*.psd" --everything
+ push to remote
> git push origin main

使用LFS将历史中的某个文件纳入到LFS的追踪管理，此时会生成`.git/lfs/objects`保存该文件对象
然后对仓库过滤，删除文件
然后强制推送到远程仓库


**git-filter-repo**
https://thoughts.aliyun.com/sharespace/5e8c37eb546fd9001aee8242/docs/5e8c37ea546fd9001aee823d
https://htmlpreview.github.io/?https://github.com/newren/git-filter-repo/blob/docs/html/git-filter-repo.html


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


**NOTE**
+ 目前只关注文件本身，所以扫描时只关注blob类型对象
+ 考虑到有种情况是扫描出来的大文件(blob)只存在历史中，此时如果想删除，指定文件名是找不到该文件的。因此，实际在做文件删除时，应该指定为blob hash值，也就是虽然看起来用户选择的是文件名，实际上使用它对应的blob hash。
+ 由于需要先将大文件扫描处理，交给用户选择，所以整个过程需要经过两次扫描。一次是根据用户选项，筛选出符合条件的文件，二是在删除重写历史时，使用fast-export进行扫描。
+ fast-export的输出结果需要结构化解析，然后才方便过滤
+ 需要提示说明同一个文件多次出现在扫描结果中的现象：同一个大文件存在不同版本，他们的文件名相同，但OID不同。
在删除时，有两种情况：一是根据文件名，一次性删除其所有版本(delete by name)，二是根据OID不同，一次删除一个版本(delete by oid)。
+ 为了防止用户没有对仓库进行备份，当用户在进行删除重、写历史操作时，有一种策略是首先进行`fresh clone safety check`，即检查正在操作的仓库是不是刚克隆的，如果不是新鲜克隆的，则很有可能该仓库是单例仓库，没有副本，没有原始仓库，所以进行一些操作是非常危险的，此时需要提示用户正在非副本仓库中进行危险操作。不是完全拒绝的，用户还可以使用选项`--force`强制进行操作 ([参考](https://htmlpreview.github.io/?https://github.com/newren/git-filter-repo/blob/docs/html/git-filter-repo.html#FRESHCLONE))。
虽然这种检测不是绝对准确的，但它是有用的。
具体检测方法是查看`git reflog`命令的结果，是否只包含一项。如果超过一项，则会被认为不是刚克隆的仓库。

**技术原理**
见 [doc](docs/technical.md)




**测试项：**

- [x] 单分支，最末端的文件及其commit
- [x] 单分支，中间单个文件及其commit
- [x] 单分支，中间连续多个文件及其commit
- [x] 单分支，中间间断的文件及其commit
- [x] 多分支，first parent(and its blob)(from-ref)
- [x] 多分支，second parent(and its blob)(merge-ref)
- [x] 多分支，all parents
- [x] 多分支，after merge point
- [x] 可执行文件
- [x] 压缩文件
- [x] 多媒体文件


极端情况下，在仓库中加入一个文件大小为1216179567 byte(1.2G)的压缩文件, 作为仓库最近一次提交(最后被扫描)，从仓库删除，最快不到10s。
```bash
$ time git clean-repo -s -v --limit=1g -n=3
Start to scan repository:
[0]: 449a189d6fb67b3dc0cfcce086847fc93ac86fd0 1216179567 gitaly-dev.tar.gz
git clean-repo -s -v --limit=1g -n=3  9.87s user 7.62s system 150% cpu 11.651 total
```

以上是理想情况，即在仓库历史中没有加入其它二进制文件，否则过程也会比较长，这取决于仓库中的数据大小。