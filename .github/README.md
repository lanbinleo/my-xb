# GitHub Workflows

## Release Workflow

### 触发条件

当推送带有 `v` 前缀的 Git tag 时自动触发，例如 `v1.0.0`、`v2.1.3` 等。

### 构建平台

该 workflow 会为以下平台构建二进制文件：

- Linux (amd64, arm64)
- Windows (amd64)
- macOS/Darwin (amd64, arm64)

### 工作流程

1. **构建阶段 (Build Job)**
- 使用 Go 1.24 编译项目
- 为每个目标平台生成独立的二进制文件
- 二进制文件命名格式：`myxb_{version}_{os}_{arch}`（Windows 平台额外添加 `.exe` 后缀）
- 将构建产物上传为 GitHub Actions artifacts

2. **发布阶段 (Release Job)**
- 等待所有平台的构建完成
- 下载所有构建产物
- 生成 SHA256 校验和文件 (`SHA256SUMS`)
- 创建 GitHub Draft Release（草稿状态）
- 自动生成 Release Notes（包含自上个 tag 以来的所有 commit）
- 上传所有二进制文件和校验和文件作为 release assets

### 发布流程

1. 在本地创建并推送 tag：
```bash
git tag v1.0.0
git push origin v1.0.0
```

2. GitHub Actions 自动运行构建和发布流程

3. 在 GitHub Releases 页面会生成一个草稿 release

4. 手动审核 release 内容（包括自动生成的 changelog）

5. 确认无误后点击 "Publish release" 正式发布

### 安全性

- Release 默认为草稿状态，必须手动审核后才能发布
- 提供 SHA256 校验和文件供用户验证下载文件的完整性
- 仅在推送 tag 时触发，防止意外发布