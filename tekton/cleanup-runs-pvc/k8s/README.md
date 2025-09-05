# Tekton PVC 清理工具 - Kubernetes 部署指南

本文档说明如何将 Tekton PVC 清理工具部署到 Kubernetes 集群中作为定期任务运行。

## 🎯 功能特性

- ✅ **自动清理**：定期清理已完成的 Tekton PipelineRun/TaskRun 产生的孤儿 PVC
- ✅ **时间排序**：优先清理最早创建的 PVC，避免长期占用存储
- ✅ **安全防护**：仅清理 Tekton 创建的 PVC，不影响用户自建资源
- ✅ **Pod 清理**：可选清理占用 PVC 的已完成 Pod
- ✅ **审计日志**：完整的操作日志和统计信息
- ✅ **预览模式**：支持 dry-run 模式，安全测试清理逻辑

## 📁 文件结构

```
k8s/
├── README.md          # 本文档
├── cleanup-job.yaml   # CronJob 配置文件
├── rbac.yaml          # ServiceAccount 和权限配置
└── debug-pod.yaml     # 调试 Pod 配置（可选）
```

## 🚀 快速部署

### 前置条件

- Kubernetes 集群 (版本 1.30+)
- 已安装 Tekton Pipelines
- kubectl 命令行工具
- 集群管理员权限（创建 ClusterRole 和 ClusterRoleBinding）

### 1. 准备镜像（可选）

如果需要自定义镜像：

```bash
# 在项目根目录执行
$ docker build -t tekton-pvc-cleanup:latest .

# 如果使用私有镜像仓库
$ docker tag tekton-pvc-cleanup:latest your-registry.com/tekton-pvc-cleanup:latest
$ docker push your-registry.com/tekton-pvc-cleanup:latest
```

### 2. 创建审计日志 PVC

> 参考 [pvc.yaml](./pvc.yaml) 示例

```bash
# 创建用于存储审计日志的 PVC
$ kubectl apply -f - <<EOF
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: tekton-pvc-cleanup
  namespace: tekton-pipelines
  labels:
    app: tekton-pvc-cleanup
    component: audit-storage
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 2Gi
EOF
```

### 3. 部署到 Kubernetes

```bash
# 创建 ServiceAccount 和 RBAC 权限
$ kubectl apply -f k8s/rbac.yaml

serviceaccount/tekton-pvc-cleaner created
clusterrole.rbac.authorization.k8s.io/tekton-pvc-cleaner created
clusterrolebinding.rbac.authorization.k8s.io/tekton-pvc-cleaner created

# 部署 CronJob
$ kubectl apply -f k8s/cleanup-job.yaml

cronjob.batch/tekton-pvc-cleanup-cron created
```

### 4. 验证部署

```bash
# 检查 ServiceAccount
$ kubectl get sa tekton-pvc-cleaner -n tekton-pipelines

NAME                 SECRETS   AGE
tekton-pvc-cleaner   0         16s

# 检查权限绑定
$ kubectl get clusterrolebinding tekton-pvc-cleaner

NAME                 ROLE                             AGE
tekton-pvc-cleaner   ClusterRole/tekton-pvc-cleaner   3m26s

# 检查 CronJob
$ kubectl get cronjob tekton-pvc-cleanup-cron -n tekton-pipelines

NAME                      SCHEDULE       TIMEZONE   SUSPEND   ACTIVE   LAST SCHEDULE   AGE
tekton-pvc-cleanup-cron   */30 * * * *   <none>     False     0        <none>          4m25s

# 检查审计日志 PVC
$ kubectl get pvc tekton-pvc-cleanup -n tekton-pipelines

tekton-pvc-cleanup   Bound    pvc-9c9a75cb-eb72-462e-a11c-6f6bc0dddb4f   2Gi 
```

### 5. 测试运行

```bash
# 手动触发一次性 Job 进行测试（dry-run 模式）
$ kubectl create job test-cleanup --from=cronjob/tekton-pvc-cleanup-cron -n tekton-pipelines

job.batch/test-cleanup created

# 查看测试执行日志
$ kubectl logs -f job/test-cleanup -n tekton-pipelines

[2025-09-05 03:04:27] 审计目录已创建: /app/audit/20250905_030427
[2025-09-05 03:04:27] 缓存目录: /app/audit/20250905_030427/cache
[2025-09-05 03:04:27] 日志文件: /app/audit/20250905_030427/cleanup.log

# 清理测试 Job
$ kubectl delete job test-cleanup -n tekton-pipelines

job.batch "test-cleanup" deleted from tekton-pipelines namespace
```

## ⚙️ 配置说明

### 主要配置参数

在 `cleanup-job.yaml` 中可以通过环境变量和参数配置以下选项：

| 参数/环境变量 | 默认值 | 说明 |
|-------------|--------|------|
| `THRESHOLD_MINUTES` | `10` | PVC 完成时间阈值（分钟），只有超过此时间的已完成 PVC 才会被清理 |
| `--all-namespaces` | 启用 | 清理所有命名空间的 Tekton PVC |
| `--namespaces` | - | 指定命名空间清理（与 --all-namespaces 互斥），如 "ns1,ns2,ns3" |
| `--cleanup-pods` | 启用 | 删除占用 PVC 且处于 Completed/Succeeded/Failed 状态的 Pod |
| `--verbose` | 启用 | 输出详细的执行日志 |
| `--dry-run` | 注释 | 仅模拟执行，不实际删除资源（测试用） |
| `--keep-cache` | 注释 | 保留缓存文件，用于调试和审计 |
| `--serial-fetch` | 注释 | 串行获取 kubectl 数据（避免内存峰值，适用于大集群） |

### CronJob 调度配置

当前配置为每 30 分钟执行一次，可以根据需要修改 `schedule` 字段：

```yaml
spec:
  schedule: "*/30 * * * *"  # 每30分钟执行一次
  # schedule: "0 */2 * * *"   # 每2小时执行一次
  # schedule: "0 2 * * *"     # 每天凌晨2点执行
  # schedule: "0 2 * * 0"     # 每周日凌晨2点执行
```

### 资源配置

当前资源配置适用于中等规模的集群：

```yaml
resources:
  requests:
    cpu: "500m"      # 0.5 CPU 核心
    memory: "256Mi"  # 256MB 内存
  limits:
    cpu: "1"         # 1 CPU 核心
    memory: "512Mi"  # 512MB 内存
```

根据集群规模调整：
- **小型集群** (< 100 PVCs)：requests: cpu=200m, memory=128Mi
- **大型集群** (> 1000 PVCs)：requests: cpu=1, memory=512Mi，建议启用 `--serial-fetch` 参数

## 🔐 权限说明

清理工具需要以下 Kubernetes 权限：

### ClusterRole 权限（跨命名空间模式）

- **PVC**: `get`, `list`, `delete`, `watch` - 查询、列表、删除和监控 PVC
- **Pod**: `get`, `list`, `delete`, `watch` - 管理占用 PVC 的 Pod
- **PipelineRun**: `get`, `list`, `watch` - 查询 PipelineRun 完成状态
- **TaskRun**: `get`, `list`, `watch` - 查询 TaskRun 完成状态
- **Namespace**: `get`, `list` - 跨命名空间操作时需要

### Role 权限（单命名空间模式）

如果只在特定命名空间操作，可以使用 `Role` 替代 `ClusterRole`，权限范围更小更安全：

```bash
# 启用单命名空间模式
# 在 cleanup-job.yaml 中修改参数：
# - --namespaces
# - "your-namespace"
```

### 权限验证

```bash
# 验证 ServiceAccount 权限
$ kubectl auth can-i get pvc --as=system:serviceaccount:tekton-pipelines:tekton-pvc-cleaner --all-namespaces

yes

$ kubectl auth can-i delete pvc --as=system:serviceaccount:tekton-pipelines:tekton-pvc-cleaner --all-namespaces

yes

$ kubectl auth can-i list pipelineruns.tekton.dev --as=system:serviceaccount:tekton-pipelines:tekton-pvc-cleaner --all-namespaces

yes
```

## 📊 监控和日志

### 查看执行状态

```bash
# 查看 CronJob 状态和下次执行时间
$ kubectl get cronjob tekton-pvc-cleanup-cron -n tekton-pipelines

NAME                      SCHEDULE       TIMEZONE   SUSPEND   ACTIVE   LAST SCHEDULE   AGE
tekton-pvc-cleanup-cron   */60 * * * *   <none>     False     0        19m             2d14h

# 查看 CronJob 详细信息
$ kubectl describe cronjob tekton-pvc-cleanup-cron -n tekton-pipelines

  Normal   SawCompletedJob   18m (x2 over 18m)     cronjob-controller  Saw completed job: tekton-pvc-cleanup-cron-29284020, condition: Complete
  Warning  UnexpectedJob     7m28s (x13 over 20h)  cronjob-controller  Saw a job that the controller did not create or forgot: test-cleanup
  Normal   SuccessfulDelete  7m28s                 cronjob-controller  Deleted job tekton-pvc-cleanup-cron-29283900

# 查看最近的 Job 执行记录
$ kubectl get jobs -l app=tekton-pvc-cleanup -n tekton-pipelines --sort-by=.metadata.creationTimestamp

NAME                               STATUS     COMPLETIONS   DURATION   AGE
tekton-pvc-cleanup-cron-29283960   Complete   1/1           49s        80m
tekton-pvc-cleanup-cron-29284020   Complete   1/1           102s       20m
test-cleanup                       Complete   1/1           39s        8m21s

# 查看当前运行的 Job 日志
$ kubectl logs -f -l app=tekton-pvc-cleanup -n tekton-pipelines

[2025-09-05 03:01:39] 实际删除了 24 个 PVC，释放存储 111 GB
[2025-09-05 03:01:39] 脚本结束执行时间: 2025-09-05 03:01:39
[2025-09-05 03:01:39] ⏱️ 总耗时: 1分17秒
[2025-09-05 03:01:39] 执行摘要已保存到: /app/audit/20250905_030023/summary.txt
[2025-09-05 03:01:39] 清理审计目录: /app/audit/20250905_030023

# 查看特定 Job 的详细日志
$ kubectl logs job/tekton-pvc-cleanup-cron-1234567890 -n tekton-pipelines

[2025-09-05 03:01:39] 实际删除了 24 个 PVC，释放存储 111 GB
[2025-09-05 03:01:39] 脚本结束执行时间: 2025-09-05 03:01:39
[2025-09-05 03:01:39] ⏱️ 总耗时: 1分17秒
[2025-09-05 03:01:39] 执行摘要已保存到: /app/audit/20250905_030023/summary.txt
[2025-09-05 03:01:39] 清理审计目录: /app/audit/20250905_030023
```

### 审计日志查看

审计日志存储在持久化卷中，包含完整的执行记录和统计信息：

```bash
# 查看审计日志 PVC 使用情况
$ kubectl get pvc tekton-pvc-cleanup -n tekton-pipelines

NAME                 STATUS   VOLUME                                     CAPACITY   ACCESS MODES   STORAGECLASS   VOLUMEATTRIBUTESCLASS   AGE
tekton-pvc-cleanup   Bound    pvc-9c9a75cb-eb72-462e-a11c-6f6bc0dddb4f   2Gi        RWO            nfs            <unset>                 1h

# 创建临时 Pod 查看审计日志
$ kubectl run audit-viewer --rm -i --tty --image=busybox --restart=Never \
  --overrides='{"spec":{"containers":[{"name":"audit-viewer","image":"busybox","command":["sh"],"volumeMounts":[{"name":"audit","mountPath":"/audit"}]}],"volumes":[{"name":"audit","persistentVolumeClaim":{"claimName":"tekton-pvc-cleanup"}}]}}' \
  -n tekton-pipelines

# 在临时 Pod 中查看审计日志
$ ls -la /audit/
 
total 4
drwxrwsr-x 7 root    nonroot  121 Sep  5 03:12 .
drwxr-xr-x 1 root    root    4096 Sep  5 03:13 ..
drwxr-sr-x 3 nonroot nonroot   57 Sep  1 08:18 20250901_081622
drwxr-sr-x 3 nonroot nonroot   57 Sep  1 08:20 20250901_081813

$ cat /audit/*/summary.txt

执行参数:
  命名空间: 所有命名空间
  时间阈值: 30 分钟
  预览模式: false
  清理Pod: true
  保留缓存: true
 
执行结果:
  总 PVC 数量: 82
  Tekton 相关 PVC: 58
  已完成且可清理: 45 (大小: 104 GB)
  已完成但未到时间: 10 (大小: 20 GB)
  相关 Run 资源仍在运行中: 3 (大小: 4 GB)
  Pod 占用跳过: 0 (大小: 0 B)
  删除失败: 0
 
执行时间: 79 秒
```

## 🛠️ 故障排除

### 常见问题及解决方案

#### 1. 权限不足错误

**症状**：Job 日志显示权限拒绝错误

```bash
# 诊断步骤
$ kubectl describe clusterrolebinding tekton-pvc-cleaner

Name:         tekton-pvc-cleaner
Labels:       app=tekton-pvc-cleanup

$ kubectl auth can-i delete pvc --as=system:serviceaccount:tekton-pipelines:tekton-pvc-cleaner --all-namespaces

yes

# 解决方案：重新应用 RBAC 配置
$ kubectl apply -f k8s/rbac.yaml

serviceaccount/tekton-pvc-cleaner configured
clusterrole.rbac.authorization.k8s.io/tekton-pvc-cleaner configured
clusterrolebinding.rbac.authorization.k8s.io/tekton-pvc-cleaner configured
```

#### 2. 镜像拉取失败

**症状**：Pod 状态为 ImagePullBackOff

```bash
# 诊断步骤
$ kubectl describe pod <pod-name> -n tekton-pipelines
$ kubectl get events --field-selector involvedObject.name=<pod-name> -n tekton-pipelines

# 解决方案：检查镜像地址和拉取策略
# 在 cleanup-job.yaml 中修改镜像地址或添加 imagePullSecrets
```

#### 3. 脚本执行失败

**症状**：Job 完成但退出码非 0

```bash
# 查看详细错误日志
$ kubectl logs <pod-name> -n tekton-pipelines

# 常见问题：
# - 集群连接失败：检查 ServiceAccount 和网络
# - 权限不足：检查 RBAC 配置
# - 时间解析错误：检查系统时区设置
```

#### 4. CronJob 不执行

**症状**：CronJob 创建但不运行

```bash
# 检查 CronJob 状态
$ kubectl describe cronjob tekton-pvc-cleanup-cron -n tekton-pipelines

# 检查 CronJob 是否被暂停
$ kubectl get cronjob tekton-pvc-cleanup-cron -n tekton-pipelines -o yaml | grep suspend

# 恢复 CronJob
$ kubectl patch cronjob tekton-pvc-cleanup-cron -n tekton-pipelines -p '{"spec":{"suspend":false}}'
```

### 调试模式

#### 创建调试 Job

```bash
# 创建带 dry-run 的调试 Job
kubectl apply -f - <<EOF
apiVersion: batch/v1
kind: Job
metadata:
  name: tekton-pvc-cleanup-debug
  namespace: tekton-pipelines
  labels:
    app: tekton-pvc-cleanup-debug
spec:
  template:
    spec:
      serviceAccountName: tekton-pvc-cleaner
      restartPolicy: Never
      containers:
      - name: cleanup
        image: registry.alauda.cn:60070/devops/tektoncd/toolbox/cleanup-pvc:latest
        command: ["/app/cleanup-tekton-pvcs.sh"]
        args:
          - "--dry-run"
          - "--verbose"
          - "--all-namespaces"
          - "--threshold"
          - "1"
          - "--keep-cache"
          - "--serial-fetch"
          - "--audit-dir"
          - "/tmp/audit"
        env:
        - name: THRESHOLD_MINUTES
          value: "1"
        volumeMounts:
        - name: audit-storage
          mountPath: /tmp/audit
      volumes:
      - name: audit-storage
        emptyDir: {}
EOF

# 查看调试日志
$ kubectl logs -f job/tekton-pvc-cleanup-debug -n tekton-pipelines

# 清理调试 Job
$ kubectl delete job tekton-pvc-cleanup-debug -n tekton-pipelines
```

#### 交互式调试

```bash
# 创建交互式调试 Pod
$ kubectl apply -f k8s/debug-pod.yaml

# 进入 Pod 进行调试
$ kubectl exec -it tekton-pvc-cleanup-debug -n tekton-pipelines -- /bin/bash

# 在 Pod 内手动执行脚本（大集群建议使用串行模式）
$ /app/cleanup-tekton-pvcs.sh --dry-run --verbose --all-namespaces --threshold 1 --serial-fetch
```

## FAQ 常见问题

### Q1: 工具会删除我手动创建的 PVC 吗？

**A**: 不会。工具只会删除由 Tekton PipelineRun 或 TaskRun 创建的 PVC（通过 ownerReferences 字段识别）。用户手动创建的 PVC 不受影响。

### Q2: 如何调整清理频率？

**A**: 修改 `cleanup-job.yaml` 中的 `schedule` 字段：
- 每 15 分钟：`"*/15 * * * *"`
- 每小时：`"0 * * * *"`
- 每天凌晨 2 点：`"0 2 * * *"`

### Q3: 工具支持多租户环境吗？

**A**: 支持。可以通过以下方式限制清理范围：
- 使用 `--namespaces ns1,ns2` 指定特定命名空间
- 使用 Role 而不是 ClusterRole 限制权限范围
- 为不同租户部署独立的清理实例

### Q4: 如何确认工具正常工作？

**A**: 通过以下方式监控：
1. 查看 CronJob 执行历史：`kubectl get jobs -l app=tekton-pvc-cleanup -n tekton-pipelines`
2. 检查审计日志：查看执行摘要和统计信息
3. 监控存储使用情况的变化

### Q5: 工具执行失败怎么办？

**A**: 按以下步骤排查：
1. 查看 Job 日志：`kubectl logs -l app=tekton-pvc-cleanup -n tekton-pipelines`
2. 检查权限：`kubectl auth can-i delete pvc --as=system:serviceaccount:tekton-pipelines:tekton-pvc-cleaner`
3. 使用调试模式：创建带 `--dry-run` 的调试 Job
4. 查看事件：`kubectl get events -n tekton-pipelines`

### Q6: 如何备份重要数据？

**A**: 建议在启用工具前：
1. 备份重要的 PVC 数据
2. 先在测试环境验证
3. 使用 `--dry-run` 模式预览清理结果
4. 监控审计日志确保清理符合预期

### Q7: 工具对集群性能有影响吗？

**A**: 影响很小：
- 使用缓存机制减少 API 调用
- 可配置资源限制控制资源使用
- 支持调整执行频率避免高峰期运行
- 建议根据集群规模调整资源配置

## 🔄 更新和维护

### 更新镜像

```bash
# 构建新版本镜像
$ docker build -t registry.alauda.cn:60070/devops/tektoncd/toolbox/cleanup-pvc:v1.1.0 .
$ docker push registry.alauda.cn:60070/devops/tektoncd/toolbox/cleanup-pvc:v1.1.0

# 更新 CronJob 中的镜像版本
$ kubectl patch cronjob tekton-pvc-cleanup-cron -n tekton-pipelines \
  -p '{"spec":{"jobTemplate":{"spec":{"template":{"spec":{"containers":[{"name":"cleanup","image":"registry.alauda.cn:60070/devops/tektoncd/toolbox/cleanup-pvc:v1.1.0"}]}}}}}}'

# 验证更新
$ kubectl describe cronjob tekton-pvc-cleanup-cron -n tekton-pipelines | grep Image
```

### 暂停和恢复 CronJob

```bash
# 暂停 CronJob（维护期间）
$ kubectl patch cronjob tekton-pvc-cleanup-cron -n tekton-pipelines \
  -p '{"spec":{"suspend":true}}'

# 验证暂停状态
$ kubectl get cronjob tekton-pvc-cleanup-cron -n tekton-pipelines

# 恢复 CronJob
$ kubectl patch cronjob tekton-pvc-cleanup-cron -n tekton-pipelines \
  -p '{"spec":{"suspend":false}}'
```

### 配置更新

```bash
# 修改清理阈值（环境变量）
$ kubectl patch cronjob tekton-pvc-cleanup-cron -n tekton-pipelines \
  -p '{"spec":{"jobTemplate":{"spec":{"template":{"spec":{"containers":[{"name":"cleanup","env":[{"name":"THRESHOLD_MINUTES","value":"20"}]}]}}}}}}'

# 修改执行频率
$ kubectl patch cronjob tekton-pvc-cleanup-cron -n tekton-pipelines \
  -p '{"spec":{"schedule":"0 */2 * * *"}}'
```

## 📋 最佳实践

### 部署前

1. **环境验证**: 确保 Tekton Pipelines 已正确安装
2. **权限测试**: 验证 ServiceAccount 权限配置
3. **预览测试**: 先使用 `--dry-run` 模式测试清理效果
4. **备份策略**: 制定重要数据备份计划

### 运行时

1. **监控执行**: 定期检查 CronJob 执行状态和日志
2. **审计跟踪**: 保留审计日志用于问题排查
3. **资源监控**: 监控工具对集群资源的影响
4. **告警设置**: 配置执行失败的告警通知

### 维护时

1. **定期更新**: 保持工具版本和安全补丁更新
2. **配置优化**: 根据使用情况调整清理频率和阈值
3. **权限审查**: 定期审查和精简权限配置
4. **性能调优**: 根据集群规模调整资源配置

### 安全考虑

1. **权限最小化**: 使用 Role 而非 ClusterRole（如果可能）
2. **网络隔离**: 限制 Pod 的网络访问范围
3. **镜像安全**: 使用安全扫描过的镜像
4. **审计合规**: 保持完整的操作审计记录

## 🗑️ 卸载指南

### 完整卸载

```bash
# 1. 暂停 CronJob 防止新任务启动
$ kubectl patch cronjob tekton-pvc-cleanup-cron -n tekton-pipelines \
  -p '{"spec":{"suspend":true}}'

# 2. 等待正在运行的 Job 完成
$ kubectl get jobs -l app=tekton-pvc-cleanup -n tekton-pipelines

# 3. 删除 CronJob 和相关资源
$ kubectl delete -f k8s/cleanup-job.yaml

# 4. 删除 RBAC 配置
$ kubectl delete -f k8s/rbac.yaml

# 5. 删除审计日志 PVC（可选，注意数据丢失）
$ kubectl delete pvc tekton-pvc-cleanup -n tekton-pipelines

# 6. 清理镜像（可选）
$ docker rmi registry.alauda.cn:60070/devops/tektoncd/toolbox/cleanup-pvc:latest
```

### 验证卸载

```bash
# 确认所有资源已删除
$ kubectl get cronjob,job,sa,clusterrole,clusterrolebinding -A | grep tekton-pvc-clean
$ kubectl get pvc tekton-pvc-cleanup -n tekton-pipelines
```

## 📞 技术支持

### 日志收集

如遇问题，请收集以下信息：

```bash
# 1. 基本信息
$ kubectl version
$ kubectl get nodes

# 2. 组件状态
$ kubectl get cronjob,job,pod -l app=tekton-pvc-cleanup -n tekton-pipelines
$ kubectl describe cronjob tekton-pvc-cleanup-cron -n tekton-pipelines

# 3. 权限信息
$ kubectl describe clusterrolebinding tekton-pvc-cleaner
$ kubectl auth can-i delete pvc --as=system:serviceaccount:tekton-pipelines:tekton-pvc-cleaner --all-namespaces

# 4. 执行日志
$ kubectl logs -l app=tekton-pvc-cleanup -n tekton-pipelines --tail=100

# 5. 事件信息
$ kubectl get events -n tekton-pipelines --sort-by='.lastTimestamp' | grep cleanup
```

### 问题报告

报告问题时请包含：
1. Kubernetes 集群版本和规模
2. Tekton Pipelines 版本
3. 工具版本和配置文件
4. 错误日志和症状描述
5. 复现步骤

