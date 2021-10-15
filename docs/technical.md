**Git仓库数据过滤的大概流程**

```
fast-export
    |
    | output stream
    |
    ---> parser(blob, commit, directives...)
            |
            |
            |
            ---> filter(blob size, blob oid)
                    |
                    | append to
                    |
                    ---> temp file
                            |
                            | output stream
                            |
                            --->  fast-import
```

+ parser的解析数据类型很多，这些类型取决于fast-export的输出内容，且根据fast-export使用的选项不同而不同。
但是，最好考虑到所有可能出现的数据类型。

+ filter目前考虑blob类型和commit类型，过滤维度包括blob size, blob oid, blob name等。

+ export和import中间需要临时文件缓存内容, 也是给解析过滤处理空间。

**git fast-export 输出分析**


`$ git fast-export --all`

```bash
blob                                                            # 类型：blob
mark :1                                                         # 序号：1
data 11                                                         # 文件（大小）：11 bytes
"11111111"                                                      # 文件内容
                                                                # 换行 LF (必须单行)
reset refs/heads/main                                           # ？？？ 表示当前分支(ref)为main
commit refs/heads/main                                          # 类型：commit
mark :2                                                         # 序号：2
author Li Linchao <lilinchao@oschina.cn> 1633749662 +0800       # author
committer Li Linchao <lilinchao@oschina.cn> 1633749662 +0800    # commiter
data 16                                                         # 数据（大小）：16
第一个commit                                                     # commit message(header, body)
M 100644 :1 README.md                                           # filechang: M(modify), D(delete), :1表示该commit修改了序号1中的文件
                                                                # 换行
blob
mark :3
data 33
CopyRight@2021
Author: lilinchao

commit refs/heads/main
mark :4
author Li Linchao <lilinchao@oschina.cn> 1633749750 +0800
committer Li Linchao <lilinchao@oschina.cn> 1633749750 +0800
data 21
add new LICENSE file
from :2                                                         # 表示该commit的parent是序号为2的commit
M 100644 :3 LICENSE                                             # 表示对序号3中的文件LICENSE进行了修改

blob
mark :5
data 22
"11111111"
"22222222"

commit refs/heads/main
mark :6
author Li Linchao <lilinchao@oschina.cn> 1633749780 +0800
committer Li Linchao <lilinchao@oschina.cn> 1633749780 +0800
data 21
修改 README 文件
from :4
M 100644 :5 README.md

reset refs/remotes/origin/main                                 # ？？？ 表示追踪的远程分支为main
from :6                                                        # 表示远程分支的commit对应本地序号6的commit
```
> 测试仓库: https://gitee.com/cactusinhand/fast-export-test.git

加上`--full-tree`选项：
`$ git fast-export --all --full-tree`:

```diff

@@ -10,6 +10,7 @@ author Li Linchao <lilinchao@oschina.cn> 1633749662 +0800
 committer Li Linchao <lilinchao@oschina.cn> 1633749662 +0800
 data 16
 第一个commit
+deleteall
 M 100644 :1 README.md

 blob
@@ -25,7 +26,9 @@ committer Li Linchao <lilinchao@oschina.cn> 1633749750 +0800
 data 21
 add new LICENSE file
 from :2
+deleteall
 M 100644 :3 LICENSE
+M 100644 :1 README.md

 blob
 mark :5
@@ -40,6 +43,8 @@ committer Li Linchao <lilinchao@oschina.cn> 1633749780 +0800
 data 21
 修改 README 文件
 from :4
+deleteall
+M 100644 :3 LICENSE
 M 100644 :5 README.md

 reset refs/remotes/origin/main
```

加上`--show-original-ids`选项：
`$ git fast-export --all --show-original-ids`:

```diff
@@ -1,11 +1,13 @@
 blob
 mark :1
+original-oid 3e7aae957d47a5bab6b32ca8878527b187f3081d
 data 11
 "11111111"

 reset refs/heads/main
 commit refs/heads/main
 mark :2
+original-oid c25c298f8980f596a9561de6b8097c2b8702e01f
 author Li Linchao <lilinchao@oschina.cn> 1633749662 +0800
 committer Li Linchao <lilinchao@oschina.cn> 1633749662 +0800
 data 16
@@ -14,12 +16,14 @@ M 100644 :1 README.md

 blob
 mark :3
+original-oid 36153aa46cfc504b14887cd7f18325ddb3d0d180
 data 33
 CopyRight@2021
 Author: lilinchao

 commit refs/heads/main
 mark :4
+original-oid 1a6699d9a2e860ee36d9abca224622139e5fda82
 author Li Linchao <lilinchao@oschina.cn> 1633749750 +0800
 committer Li Linchao <lilinchao@oschina.cn> 1633749750 +0800
 data 21
@@ -29,12 +33,14 @@ M 100644 :3 LICENSE

 blob
 mark :5
+original-oid bea31b52a7f5d38ef0ddc0b1ec35c2caeb26006a
 data 22
 "11111111"
 "22222222"

 commit refs/heads/main
 mark :6
+original-oid fc556bc71a844cdec3eb878fd489fe59c469e7e9
 author Li Linchao <lilinchao@oschina.cn> 1633749780 +0800
 committer Li Linchao <lilinchao@oschina.cn> 1633749780 +0800
 data 21
```



**部分选项说明**

**--full-tree** 选项会在输出中的每个commit中标记一个deleteall`指令`，后面跟着这个commit中修改过的文件

（而不仅是列出于该提交的第一个父提交相同的文件）

**--show-original-ids** 选项 会在输出中加入original-oid <SHA1SUM>`指令`, 这个对于重写commit历史，或者通过ID裁剪blob有帮助

**--reencode=(yes|no|abort)** 选项 用于处理commit信息中的编码问题， yes表示将commit message重新编码为UTF-8


---
tag的处理情况有点特殊：

对于轻量级tag，首次引入时的变化是：
```diff
diff --git a/tmp b/tmp
index 4100964..d9ffcc7 100644
--- a/tmp
+++ b/tmp
@@ -12,3 +12,6 @@ data 11
 add file a
 M 100644 :1 a.txt

+reset refs/tags/v1.0.a
+from :2
+
(END)
```

之后在进行commit提交，则变化如下：
```diff
diff --git a/tmp b/tmp
index d9ffcc7..16fe281 100644
--- a/tmp
+++ b/tmp
@@ -3,8 +3,8 @@ mark :1
 data 9
 "file a"

-reset refs/heads/main
-commit refs/heads/main
+reset refs/tags/v1.0.a
+commit refs/tags/v1.0.a
 mark :2
 author Li Linchao <lilinchao@oschina.cn> 1633931981 +0800
 committer Li Linchao <lilinchao@oschina.cn> 1633931981 +0800
@@ -12,6 +12,17 @@ data 11
 add file a
 M 100644 :1 a.txt

-reset refs/tags/v1.0.a
+blob
+mark :3
+data 9
+"file b"
+
+commit refs/heads/main
+mark :4
+author Li Linchao <lilinchao@oschina.cn> 1633932075 +0800
+committer Li Linchao <lilinchao@oschina.cn> 1633932075 +0800
+data 11
+add file b
 from :2
+M 100644 :3 b.txt

(END)
```

再次进行commit提交：
```diff
diff --git a/tmp b/tmp
index 16fe281..b1ca763 100644
--- a/tmp
+++ b/tmp
@@ -26,3 +26,17 @@ add file b
 from :2
 M 100644 :3 b.txt

+blob
+mark :5
+data 9
+"file c"
+
+commit refs/heads/main
+mark :6
+author Li Linchao <lilinchao@oschina.cn> 1633932124 +0800
+committer Li Linchao <lilinchao@oschina.cn> 1633932124 +0800
+data 11
+add file c
+from :4
+M 100644 :5 c.txt
+
(END)
```

继续加上轻量级tag：
```diff
diff --git a/tmp b/tmp
index b1ca763..e384d3d 100644
--- a/tmp
+++ b/tmp
@@ -40,3 +40,6 @@ add file c
 from :4
 M 100644 :5 c.txt

+reset refs/tags/v1.0.c
+from :6
+
(END)
```

所以`reset`是一段commit范围内的基准，reset会重置commit所在的引用。

如果一个commit没有parent，则会在它前面加上reset字段, 该reset即为当前分支名


对于标注型tag， 则会在最后加上tag的详细信息，类似于commit：
```diff
diff --git a/tag.txt b/tag.txt
index e06b3ce..46f05fa 100644
--- a/tag.txt
+++ b/tag.txt
@@ -45,3 +45,9 @@ M 100644 :5 README.md
 reset refs/remotes/origin/main
 from :6
                                                              # LF
+tag v1.1                                                     # tag name
+from :6                                                      # tag from mark_6
+tagger Li Linchao <lilinchao@oschina.cn> 1633761415 +0800    # tagger
+data 10                                                      # tag size
+noted tag                                                    # tag message
+
(END)
```

> 如果要对tag进行序号标记，则需要加上`--mark-tags`选项。


---

blob类型数据包含的字段：

+ blob
+ mark
+ data
+ original-oid

如果删除blob, 则也会涉及到commit， 所以也要解析commit

fast-export输出流中，commit类型数据包含的字段：

+ reset
+ commit
+ mark
+ author
+ commiter
+ encoding
+ from
+ merge
+ filechange
+ original-oid
+ deleteall


如果想删除某个文件(blob)以及其涉及的提交(commit)，对输出流的改动如下：

```diff
diff --git a/all.txt b/all.txt
index e4a9cba..2bd2161 100644
--- a/all.txt
+++ b/all.txt
@@ -1,47 +1,17 @@
 blob
 mark :1
-data 11
-"11111111"
-
-reset refs/heads/main
-commit refs/heads/main
-mark :2
-author Li Linchao <lilinchao@oschina.cn> 1633749662 +0800
-committer Li Linchao <lilinchao@oschina.cn> 1633749662 +0800
-data 16
-第一个commit
-M 100644 :1 README.md
-
-blob
-mark :3
 data 33
 CopyRight@2021
 Author: lilinchao

+reset refs/heads/main
 commit refs/heads/main
-mark :4
+mark :2
 author Li Linchao <lilinchao@oschina.cn> 1633749750 +0800
 committer Li Linchao <lilinchao@oschina.cn> 1633749750 +0800
 data 21
 add new LICENSE file
-from :2
-M 100644 :3 LICENSE
-
-blob
-mark :5
-data 22
-"11111111"
-"22222222"
-
-commit refs/heads/main
-mark :6
-author Li Linchao <lilinchao@oschina.cn> 1633749780 +0800
-committer Li Linchao <lilinchao@oschina.cn> 1633749780 +0800
-data 21
-修改 README 文件
-from :4
-M 100644 :5 README.md
+M 100644 :1 LICENSE

 reset refs/remotes/origin/main
-from :6
+from :2

(END)
```

完整数据如下：
```bash
blob
mark :1
data 33
CopyRight@2021
Author: lilinchao

reset refs/heads/main
commit refs/heads/main
mark :2
author Li Linchao <lilinchao@oschina.cn> 1633749750 +0800
committer Li Linchao <lilinchao@oschina.cn> 1633749750 +0800
data 21
add new LICENSE file
M 100644 :1 LICENSE

reset refs/remotes/origin/main
from :2
```

最后，将该文件作为输入流，传递给fast-import则可以得到一个新的完整的仓库，里面不包含前面删除的文件
```bash
$ git init new-repo
$ cd new-repo
$ git fast-import <../output
$ git reset --hard
```

解析过程就是逐行读取数据流，并识别出不同的数据类型，该过程伴随着数据格式检验
过滤过程就是删除指定的blob，以及对应的commit，并且更新所有的mark序号(否则fast-import解析出错，达不到预期的效果)。

通过使用`--show-original-ids`选项，可以得到所有对象的oid, 然后可以进行过滤。
blob后面会紧跟一个commit，可以是多个blob跟一个commit




```go

package main

import "io"

func do() {
        reader := git-fast-export
        r1, w1 := io.Pipe()
        writer := git-fast-import

        go func() {
        // run git-fast-export process
        // parse output
        // filter out some objects
        // write to pipe: w1.Write(reader)
        // # if parsed "done" flag, then w1.Close()
        }()

        go func() {
                // run git-fast-import process
                // copy Pipe output to writer
                // io.Copy(writer, r1)
        } ()
}

```

**NOTE**

+ 全局mark_id处理问题，mark_id, hash_id
+ 考虑到流式处理，不用对mark_id做减法，每次都顺序做加法

> 原始id序号为：A:1, B:2, C:3
> 加上过滤筛选条件后，按顺序A还是为1，B需要被过滤，跳过，C则自动加1，变为2
> commitD 依赖B，则commitD 也会修改。

> Blob, Commit, Reset, Tag, Filechange类型数据，底层都嵌套有`GitElements`结构，其中有`dumped`这个字段，
一旦检测到改字段为`false`，则意味着改类型需要被过滤，整个条数据不再写入流中，同时直接跳到下一行继续解析其它类型数据。
