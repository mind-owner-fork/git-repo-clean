**远程仓库更新了，如何更新本地关联仓库?**



####  背景

假设远程服务器端存在一个仓库repo-server, 开发者A和B分别克隆了repo-a, repo-b进行分布式协同开发。

开发者A在本地repo_a中提交了一些commit，但是提交中不小心包含了比较大的非代码文件，这导致了仓库超出了最大容量限额，于是推送失败。于是他选择使用git-clean-repo工具清理提交历史中的大文件。

清理完成后，按照提示运行`git push origin --all --force`命令顺利推送到远程仓库。



#### 问题

开发者B本地也进行了一些提交，但是此时如果想推送到远程，肯定会失败的，因为远程的合并基准以及发生改变：

```bash
$ git push origin master
Enter passphrase for key '/home/git/.ssh/id_ed25519':
To gitee.com:cactusinhand/my-repo.git
 ! [rejected]          master -> master (fetch first)
error: failed to push some refs to 'gitee.com:cactusinhand/my-repo.git'
hint: Updates were rejected because the remote contains work that you do
hint: not have locally. This is usually caused by another repository pushing
hint: to the same ref. You may want to first integrate the remote changes
hint: (e.g., 'git pull ...') before pushing again.
hint: See the 'Note about fast-forwards' in 'git push --help' for details.
```



####  解决方法

**git pull --rebase=true**



```bash
$ git pull --rebase=true                                                                 
Enter passphrase for key '/home/git/.ssh/id_ed25519':
remote: Enumerating objects: 8, done.
remote: Counting objects: 100% (8/8), done.
remote: Total 15 (delta 8), reused 8 (delta 8), pack-reused 7
Unpacking objects: 100% (15/15), 2.21 KiB | 188.00 KiB/s, done.
From gitee.com:cactusinhand/my-repo
 + f4a81e18...b83e4cb8 master     -> origin/master  (forced update)
   d485b7a5..baa5db10  test       -> origin/test (forced update)
Successfully rebased and updated refs/heads/master.
```

这个命令会把远程所有的新增的数据fetch下来，并进行rebase操作。结果看起来就是repo_b的提交线性地在repo_a的提交(远程仓库的最新提交)之后，成功之后开发者B就可以正常推送他的提交了。

```bash
$ git status
On branch master
Your branch is ahead of 'origin/master' by 1 commit.
  (use "git push" to publish your local commits)

nothing to commit, working tree clean
```



为了展示更为复杂一点的情况，上面的过程表明远程有两个分支master, test进行了更新。

因为当前分支(即master分支)已经rebase更新成功，这个时候需要切换到test分支继续rebase操作：

```bash
$ git checkout test
$ git pull --rebase=true
Enter passphrase for key '/home/git/.ssh/id_ed25519':
Updating d485b7a58..baa5db108
Fast-forward
 files/4004.c | 0
 files/4005.c | 0
 2 files changed, 0 insertions(+), 0 deletions(-)
 create mode 100644 files/4004.c
 create mode 100644 files/4005.c
```



操作成功之后，本地仓库repo_b的test分支就与远程的test分支也保持一致了：

```bash
$ git status
On branch test
Your branch is up to date with 'origin/test'.

nothing to commit, working tree clean
```



**rebase冲突处理**

如果这个过程出现合并冲突，则需要先解决冲突。解决的原则就是以远程的数据为主，我们需要丢弃掉本地不需要的文件：

```bash
# 如果暂存区存在待提交但没有提交的文件，需要丢掉：
$ git restore --staged files
$ git restore files

# 如果有Untracked files，需要删除：
$ rm files

# 冲突解决之后，继续rebase操作：
$ git rebase --continue

# ok之后查看：
$ git status
On branch test
Your branch is up to date with 'origin/test'.  #与远程端保持一致

nothing to commit, working tree clean
```



所有工作完成之后，本地仓库repo_b就可以向远程repo-server正常推送了。