# 发布 aicommit

发布由 Git tag 和 [CHANGELOG.md](../CHANGELOG.md) 中人工维护的说明共同驱动。
tag 是构建版本；CHANGELOG 中对应版本段落是 GitHub Release 的正文来源。

## 日常维护 Unreleased

每完成一项用户可见的变更，就将它写入 `## Unreleased`。可按 Added、
Changed、Fixed 或 Security 等合适的小标题分类。不要等到 tag 推送之后再补写
Release 说明。

## 发布前验证

先运行与本次改动相关的检查：

```bash
go test ./...
go build -o aicommit .
```

## 定版

先预览将要发布的说明，不修改仓库：

```bash
bash scripts/release.sh 0.1.7 --dry-run
```

确认后创建发布提交和带注释的 tag：

```bash
bash scripts/release.sh 0.1.7
```

脚本要求工作区干净、版本合法且未被使用，并且 `Unreleased` 中有实际内容。
它会将说明移动到 `## 0.1.7 - YYYY-MM-DD`，提交 CHANGELOG，并创建
`v0.1.7`。

确认本地提交和 tag 后再推送：

```bash
git push origin <branch>
git push origin v0.1.7
```

也可以在发布命令中加入 `--push`。

## CI 发布内容

推送 tag 后 GitHub Actions 会运行 `go test ./...`，构建 macOS、Linux、
Windows 的 amd64 和 arm64 二进制，从 CHANGELOG 提取对应版本段落并附加安装说明
创建 GitHub Release，最后发布 npm 分发包装。

npm 包会在发布时按 tag 写入版本号；它的 postinstall 脚本下载同版本的 Release
二进制，因此安装指定 npm 版本时得到的也是对应版本的原生二进制。
