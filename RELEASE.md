# 发布流程说明

## GitHub Releases 自动化流程

本项目已配置 GitHub Actions 自动化发布流程，当推送版本标签时会自动构建并发布到 GitHub Releases。

### 发布步骤

1. **准备发布**

   ```bash
   # 确保代码已提交
   git add .
   git commit -m "准备发布 v1.0.0"
   ```

2. **创建版本标签**

   ```bash
   # 创建并推送版本标签
   git tag -a v1.0.0 -m "Release v1.0.0"
   git push origin v1.0.0
   ```

3. **自动发布**
   - GitHub Actions 会自动检测到标签推送
   - 自动构建优化的二进制文件
   - 自动创建 GitHub Release
   - 二进制文件会附加到 Release 中

### 本地构建

如果需要本地构建发布版本：

```bash
# 构建优化版本
make build-release

# 查看构建结果
ls -la dist/
```

### 发布文件说明

- **单一二进制文件**: 由于 Go 的静态链接特性，单个二进制文件可以在不同架构上运行
- **优化构建**: 使用 `-ldflags="-s -w"` 去除调试信息，减小文件大小
- **无依赖**: 使用 `CGO_ENABLED=0` 确保纯 Go 构建，无外部依赖

### 版本命名规范

- 使用语义化版本号：`v1.0.0`, `v1.1.0`, `v2.0.0`
- 预发布版本：`v1.0.0-beta.1`, `v1.0.0-rc.1`
- 开发版本：`v1.0.0-dev.1`

### 手动发布

如果需要手动发布（不推荐）：

1. 运行 `make build-release`
2. 在 GitHub 上创建新的 Release
3. 上传 `dist/spot-instance-advisor` 文件
4. 填写 Release 说明

### 注意事项

- 确保在推送标签前所有代码已测试通过
- 版本标签一旦推送就会触发自动发布
- 如需撤销，需要删除 GitHub Release 和标签
