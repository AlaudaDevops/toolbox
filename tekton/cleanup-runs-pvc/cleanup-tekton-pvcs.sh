#!/bin/bash

set -e

# 常量定义
readonly BYTES_PER_GB=1073741824
readonly BYTES_PER_MB=1048576
readonly BYTES_PER_KB=1024
readonly PROGRESS_INTERVAL=5
readonly DEFAULT_THRESHOLD_MINUTES=10
readonly DEFAULT_AUDIT_DIR="./audit"

# 变量定义
NAMESPACES=()
ALL_NAMESPACES=false
DRY_RUN=false
THRESHOLD_MINUTES=${DEFAULT_THRESHOLD_MINUTES}
VERBOSE=false
LOG_LEVEL="INFO"  # DEBUG, INFO, WARN, ERROR
CLEANUP_COMPLETED_PODS=false
AUDIT_DIR="${DEFAULT_AUDIT_DIR}"
KEEP_CACHE=false
PARALLEL_FETCH=true

usage() {
    cat << EOF
Usage: $0 [OPTIONS]

清理已完成的 Tekton PipelineRun / TaskRun 创建的 PVC

OPTIONS:
    -n, --namespaces NAMESPACE  指定命名空间（支持多个，用逗号分隔，如：ns1,ns2,ns3）
    -A, --all-namespaces       处理所有命名空间
    -d, --dry-run              预览模式，不实际删除
    -t, --threshold MINUTES    完成时间阈值（分钟，默认 ${DEFAULT_THRESHOLD_MINUTES} 分钟）
    -c, --cleanup-pods         删除占用 PVC 且处于 Completed、Succeeded、Failed 状态的 Pod
    -v, --verbose              详细输出
    --audit-dir DIR            指定审计目录（默认 ${DEFAULT_AUDIT_DIR}）
    --keep-cache               保留缓存和日志文件
    --serial-fetch             串行获取 kubectl 数据（避免内存峰值）
    -h, --help                 显示帮助信息

示例:
    $0 --namespaces tekton-pipelines --dry-run
    $0 --namespaces ns1,ns2,ns3 --threshold 20 --cleanup-pods
    $0 --all-namespaces --threshold 20 --cleanup-pods
    $0 -n my-namespace -v -c --audit-dir /tmp/cleanup-audit
    $0 --all-namespaces --serial-fetch --threshold 30 --dry-run
EOF
}

log() {
    local message="[$(date '+%Y-%m-%d %H:%M:%S')] $*"
    # 输出到控制台
    stdbuf --output=0 echo "$message" 1>&2
    # 同时写入日志文件（如果已定义）
    if [[ -n "$LOG_FILE" ]]; then
        echo "$message" >> "$LOG_FILE"
    fi
    return
}

debug_log() {
    [[ "$LOG_LEVEL" == "DEBUG" ]] && log "DEBUG: $1"
}

verbose_log() {
    if [[ "$VERBOSE" == "true" ]]; then
        log "${1}"
    fi
}

warn_log() {
    local message="[$(date '+%Y-%m-%d %H:%M:%S')] WARN: ${1}"
    echo "$message" >&2
    if [[ -n "$LOG_FILE" ]]; then
        echo "$message" >> "$LOG_FILE"
    fi
}

error_log() {
    local message="[$(date '+%Y-%m-%d %H:%M:%S')] ERROR: ${1}"
    echo "$message" >&2
    if [[ -n "$LOG_FILE" ]]; then
        echo "$message" >> "$LOG_FILE"
    fi
}

convert_storage_size() {
    local size_str="${1}"
    # 移除单位，转换为字节
    if [[ "${size_str}" =~ ^([0-9]+)Gi$ ]]; then
        echo $(( BASH_REMATCH[1] * BYTES_PER_GB ))
    elif [[ "${size_str}" =~ ^([0-9]+)G$ ]]; then
        echo $(( BASH_REMATCH[1] * 1000000000 ))
    elif [[ "${size_str}" =~ ^([0-9]+)Mi$ ]]; then
        echo $(( BASH_REMATCH[1] * BYTES_PER_MB ))
    elif [[ "${size_str}" =~ ^([0-9]+)M$ ]]; then
        echo $(( BASH_REMATCH[1] * 1000000 ))
    elif [[ "${size_str}" =~ ^([0-9]+)Ki$ ]]; then
        echo $(( BASH_REMATCH[1] * BYTES_PER_KB ))
    elif [[ "${size_str}" =~ ^([0-9]+)K$ ]]; then
        echo $(( BASH_REMATCH[1] * 1000 ))
    elif [[ "${size_str}" =~ ^([0-9]+)$ ]]; then
        echo "${BASH_REMATCH[1]}"
    else
        echo "0"
    fi
}

format_storage_size() {
    local bytes="${1}"
    if [ "${bytes}" = "null" ]; then
        bytes=0
    fi
    if [[ ${bytes} -ge ${BYTES_PER_GB} ]]; then
        echo "$(( bytes / BYTES_PER_GB )) GB"
    elif [[ ${bytes} -ge ${BYTES_PER_MB} ]]; then
        echo "$(( bytes / BYTES_PER_MB )) MB"
    elif [[ ${bytes} -ge ${BYTES_PER_KB} ]]; then
        echo "$(( bytes / BYTES_PER_KB )) KB"
    else
        echo "${bytes} B"
    fi
}

parse_timestamp() {
    local time_str="$1"
    local timestamp

    # 支持多种时间格式
    timestamp=$(date -d "$time_str" +%s 2>/dev/null) || \
    timestamp=$(date -j -f "%Y-%m-%dT%H:%M:%SZ" "$time_str" +%s 2>/dev/null) || \
    timestamp=$(date -j -f "%Y-%m-%dT%H:%M:%S%z" "$time_str" +%s 2>/dev/null) || \
    timestamp=$(gdate -d "$time_str" +%s 2>/dev/null)

    if [[ -z "$timestamp" ]]; then
        error_log "无法解析时间格式: $time_str"
        return 1
    fi
    echo "$timestamp"
}

check_prerequisites() {
    if ! command -v kubectl &> /dev/null; then
        error_log "kubectl 命令未找到，请先安装 kubectl"
        exit 1
    fi

    if ! command -v jq &> /dev/null; then
        error_log "jq 命令未找到，请先安装 jq"
        exit 1
    fi

    # 检查集群连接 - 使用简单的权限检查替代 cluster-info
    local auth_check_output
    if ! auth_check_output=$(kubectl auth can-i get pvc --all-namespaces 2>&1); then
        error_log "无法连接到 Kubernetes 集群或权限不足"
        error_log "详细错误信息: $auth_check_output"
        exit 1
    fi

    # 检查删除权限
    local delete_check_output
    if ! delete_check_output=$(kubectl auth can-i delete pvc --all-namespaces 2>&1); then
        error_log "ServiceAccount 缺少删除 PVC 权限"
        error_log "详细错误信息: $delete_check_output"
        exit 1
    fi
}

parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            -n|--namespaces)
                # 支持多个命名空间，用逗号分隔
                IFS=',' read -ra NAMESPACES <<< "$2"
                shift 2
                ;;
            -A|--all-namespaces)
                ALL_NAMESPACES=true
                shift
                ;;
            -d|--dry-run)
                DRY_RUN=true
                shift
                ;;
            -t|--threshold)
                THRESHOLD_MINUTES="$2"
                shift 2
                ;;
            -c|--cleanup-pods)
                CLEANUP_COMPLETED_PODS=true
                shift
                ;;
            -v|--verbose)
                VERBOSE=true
                shift
                ;;
            --audit-dir)
                AUDIT_DIR="$2"
                shift 2
                ;;
            --keep-cache)
                KEEP_CACHE=true
                shift
                ;;
            --serial-fetch)
                PARALLEL_FETCH=false
                shift
                ;;
            -h|--help)
                usage
                exit 0
                ;;
            *)
                error_log "未知参数: ${1}"
                usage
                exit 1
                ;;
        esac
    done

    if [[ "${ALL_NAMESPACES}" == "false" && ${#NAMESPACES[@]} -eq 0 ]]; then
        error_log "必须指定命名空间或使用 --all-namespaces"
        usage
        exit 1
    fi

    if [[ "${ALL_NAMESPACES}" == "true" && ${#NAMESPACES[@]} -gt 0 ]]; then
        error_log "不能同时使用 --namespaces 和 --all-namespaces"
        usage
        exit 1
    fi

    if ! [[ "${THRESHOLD_MINUTES}" =~ ^[0-9]+$ ]] || [[ "${THRESHOLD_MINUTES}" -lt 1 ]]; then
        error_log "阈值必须是大于 0 的整数"
        exit 1
    fi
}

extract_tekton_pvcs_from_cache() {
    jq -r '
        def convert_storage(size):
            if (size | test("^[0-9]+Gi$")) then
                (size | gsub("Gi";"") | tonumber) * 1073741824
            elif (size | test("^[0-9]+G$")) then
                (size | gsub("G";"") | tonumber) * 1000000000
            elif (size | test("^[0-9]+Mi$")) then
                (size | gsub("Mi";"") | tonumber) * 1048576
            elif (size | test("^[0-9]+M$")) then
                (size | gsub("M";"") | tonumber) * 1000000
            elif (size | test("^[0-9]+Ki$")) then
                (size | gsub("Ki";"") | tonumber) * 1024
            elif (size | test("^[0-9]+K$")) then
                (size | gsub("K";"") | tonumber) * 1000
            else
                (size | tonumber // 0)
            end;
        [.items[] |
        select(.metadata.ownerReferences[]? | .kind == "PipelineRun" or .kind == "TaskRun") |
        {
            name: .metadata.name,
            namespace: .metadata.namespace,
            ownerKind: (.metadata.ownerReferences[] | select(.kind == "PipelineRun" or .kind == "TaskRun") | .kind),
            ownerName: (.metadata.ownerReferences[] | select(.kind == "PipelineRun" or .kind == "TaskRun") | .name),
            creationTime: .metadata.creationTimestamp,
            storageSize: (.spec.resources.requests.storage // "0"),
            storageBytes: (.spec.resources.requests.storage // "0" | convert_storage(.)),
            phase: (.status.phase // "Unknown")
        }] |
        sort_by(.creationTime) |
        .[] |
        [.name, .namespace, .ownerKind, .ownerName, .creationTime, .storageSize, .storageBytes, .phase] |
        @tsv
    ' "${TEMP_DIR}/pvcs.json"
}

get_total_pvc_count_from_cache() {
    jq '.items | length' "$TEMP_DIR/pvcs.json"
}

get_tekton_pvc_count_from_cache() {
    jq '[.items[] | select(.metadata.ownerReferences[]? | .kind == "PipelineRun" or .kind == "TaskRun")] | length' "$TEMP_DIR/pvcs.json"
}

get_timestamp_and_age_info() {
    local completion_time="$1"
    local completion_timestamp

    if ! completion_timestamp=$(parse_timestamp "$completion_time"); then
        return 1
    fi

    local current_timestamp
    current_timestamp=$(date +%s)
    local age_seconds=$((current_timestamp - completion_timestamp))
    local threshold_seconds=$((THRESHOLD_MINUTES * 60))

    # 计算时间差
    local age_minutes=$((age_seconds / 60))
    local age_hours=$((age_minutes / 60))
    local age_days=$((age_hours / 24))

    # 格式化时间差显示
    local age_info
    if [[ $age_days -gt 0 ]]; then
        age_info="${age_days} 天 $((age_hours % 24)) 小时$((age_minutes % 60)) 分钟"
    elif [[ $age_hours -gt 0 ]]; then
        age_info="${age_hours} 小时$((age_minutes % 60)) 分钟"
    else
        age_info="${age_minutes} 分钟"
    fi

    # 返回: timestamp|age_info|is_older_than_threshold
    if [[ $age_seconds -gt $threshold_seconds ]]; then
        echo "${completion_timestamp}|${age_info}|true"
    else
        echo "${completion_timestamp}|${age_info}|false"
    fi
}

setup_audit_dir() {
    # 创建审计目录结构
    local timestamp=$(date +%Y%m%d_%H%M%S)
    AUDIT_SESSION_DIR="${AUDIT_DIR}/${timestamp}"
    TEMP_DIR="${AUDIT_SESSION_DIR}/cache"
    LOG_FILE="${AUDIT_SESSION_DIR}/cleanup.log"

    mkdir -p "$TEMP_DIR"
    log "审计目录已创建: $AUDIT_SESSION_DIR"
    log "缓存目录: $TEMP_DIR"
    log "日志文件: $LOG_FILE"
}

cache_all_data() {
    verbose_log "缓存 Kubernetes 资源数据到目录: $TEMP_DIR"

    if [[ "$ALL_NAMESPACES" == "true" ]]; then
        if [[ "$PARALLEL_FETCH" == "true" ]]; then
            # 处理所有命名空间 - 并行获取
            kubectl get pvc --all-namespaces -o json > "$TEMP_DIR/pvcs.json" &
            local pvc_pid=$!

            kubectl get pipelinerun --all-namespaces -o json > "$TEMP_DIR/pipelineruns.json" 2>/dev/null &
            local pr_pid=$!

            kubectl get taskrun --all-namespaces -o json > "$TEMP_DIR/taskruns.json" 2>/dev/null &
            local tr_pid=$!

            kubectl get pods --all-namespaces -o json > "$TEMP_DIR/pods.json" &
            local pods_pid=$!
        else
            # 处理所有命名空间 - 串行获取（避免内存峰值）
            verbose_log "获取 PVC 数据..."
            kubectl get pvc --all-namespaces -o json > "$TEMP_DIR/pvcs.json"

            verbose_log "获取 PipelineRun 数据..."
            kubectl get pipelinerun --all-namespaces -o json > "$TEMP_DIR/pipelineruns.json" 2>/dev/null || echo '{"items":[]}' > "$TEMP_DIR/pipelineruns.json"

            verbose_log "获取 TaskRun 数据..."
            kubectl get taskrun --all-namespaces -o json > "$TEMP_DIR/taskruns.json" 2>/dev/null || echo '{"items":[]}' > "$TEMP_DIR/taskruns.json"

            verbose_log "获取 Pod 数据..."
            kubectl get pods --all-namespaces -o json > "$TEMP_DIR/pods.json"
        fi
    else
        # 处理多个指定的命名空间
        local all_data='{"items":[]}'

        # 临时文件前缀
        local temp_prefix="$TEMP_DIR/temp_"

        # 串行处理每个命名空间，但每个命名空间内的资源类型可选择并行获取
        for namespace in "${NAMESPACES[@]}"; do
            verbose_log "获取命名空间 $namespace 的数据..."
            if [[ "$PARALLEL_FETCH" == "true" ]]; then
                # 在单个命名空间内并行获取4种资源类型
                kubectl get pvc -n "$namespace" -o json > "${temp_prefix}pvcs_${namespace}.json" 2>/dev/null &
                kubectl get pipelinerun -n "$namespace" -o json > "${temp_prefix}pipelineruns_${namespace}.json" 2>/dev/null &
                kubectl get taskrun -n "$namespace" -o json > "${temp_prefix}taskruns_${namespace}.json" 2>/dev/null &
                kubectl get pods -n "$namespace" -o json > "${temp_prefix}pods_${namespace}.json" 2>/dev/null &
                # 等待当前命名空间的所有命令完成
                wait
            else
                # 串行获取当前命名空间的数据
                kubectl get pvc -n "$namespace" -o json > "${temp_prefix}pvcs_${namespace}.json" 2>/dev/null || echo '{"items":[]}' > "${temp_prefix}pvcs_${namespace}.json"
                kubectl get pipelinerun -n "$namespace" -o json > "${temp_prefix}pipelineruns_${namespace}.json" 2>/dev/null || echo '{"items":[]}' > "${temp_prefix}pipelineruns_${namespace}.json"
                kubectl get taskrun -n "$namespace" -o json > "${temp_prefix}taskruns_${namespace}.json" 2>/dev/null || echo '{"items":[]}' > "${temp_prefix}taskruns_${namespace}.json"
                kubectl get pods -n "$namespace" -o json > "${temp_prefix}pods_${namespace}.json" 2>/dev/null || echo '{"items":[]}' > "${temp_prefix}pods_${namespace}.json"
            fi
        done

        # 合并各个命名空间的数据
        for resource_type in pvcs pipelineruns taskruns pods; do
            local temp_files=()
            for namespace in "${NAMESPACES[@]}"; do
                local temp_file="${temp_prefix}${resource_type}_${namespace}.json"
                if [[ -f "$temp_file" ]]; then
                    temp_files+=("$temp_file")
                fi
            done

            if [[ ${#temp_files[@]} -gt 0 ]]; then
                # 使用jq合并所有文件的items数组
                jq -s 'map(.items) | flatten | {items: .}' "${temp_files[@]}" > "$TEMP_DIR/${resource_type}.json"
                # 清理临时文件
                rm -f "${temp_files[@]}"
            else
                echo '{"items":[]}' > "$TEMP_DIR/${resource_type}.json"
            fi
        done

    fi

    # 等待所有命令完成（仅当使用--all-namespaces且并行模式时）
    if [[ "$ALL_NAMESPACES" == "true" && "$PARALLEL_FETCH" == "true" ]]; then
        wait $pvc_pid
        wait $pr_pid || echo '{"items":[]}' > "$TEMP_DIR/pipelineruns.json"
        wait $tr_pid || echo '{"items":[]}' > "$TEMP_DIR/taskruns.json"
        wait $pods_pid
    fi

    verbose_log "数据缓存完成"
}

cleanup_cache_data() {
    if [[ "$KEEP_CACHE" == "true" ]]; then
        log "保留审计数据和缓存文件: $AUDIT_SESSION_DIR"
    else
        if [[ -n "$TEMP_DIR" && -d "$TEMP_DIR" ]]; then
            verbose_log "清理临时缓存目录: $TEMP_DIR"
            rm -rf "$TEMP_DIR" || true
        fi
    fi
}

get_run_completion_time_from_cache() {
    local run_name="$1"
    local namespace="$2"
    local run_type="$3"

    if [[ "$run_type" == "pipelinerun" ]]; then
        jq -r --arg name "$run_name" --arg ns "$namespace" '
            .items[] |
            select(.metadata.name == $name and .metadata.namespace == $ns) |
            select(.status.conditions[]? | select(.type == "Succeeded" and (.status == "True" or .status == "False"))) |
            select(.status.completionTime != null) |
            .status.completionTime
        ' "$TEMP_DIR/pipelineruns.json"
    elif [[ "$run_type" == "taskrun" ]]; then
        # 一次性查询TaskRun信息和父级PipelineRun状态
        jq -r --arg name "$run_name" --arg ns "$namespace" '
            (.items[] | select(.metadata.name == $name and .metadata.namespace == $ns)) as $taskrun |
            if ($taskrun.metadata.labels["tekton.dev/pipelineRun"] // empty) != "" then
                # TaskRun 属于 PipelineRun，检查父级 PipelineRun 状态
                ($taskrun.metadata.labels["tekton.dev/pipelineRun"]) as $parent_pr |
                "PARENT|" + $parent_pr
            else
                # 独立的 TaskRun，检查自身状态
                if ($taskrun.status.conditions[]? | select(.type == "Succeeded" and (.status == "True" or .status == "False"))) and ($taskrun.status.completionTime != null) then
                    $taskrun.status.completionTime
                else
                    empty
                end
            end
        ' "$TEMP_DIR/taskruns.json" | while IFS= read -r result; do
            if [[ "$result" =~ ^PARENT\| ]]; then
                local parent_pipelinerun="${result#PARENT|}"
                verbose_log "TaskRun $namespace/$run_name 属于 PipelineRun $namespace/$parent_pipelinerun，检查 PipelineRun 状态"
                # 检查父级PipelineRun的完成状态
                jq -r --arg name "$parent_pipelinerun" --arg ns "$namespace" '
                    .items[] |
                    select(.metadata.name == $name and .metadata.namespace == $ns) |
                    select(.status.conditions[]? | select(.type == "Succeeded" and (.status == "True" or .status == "False"))) |
                    select(.status.completionTime != null) |
                    .status.completionTime
                ' "$TEMP_DIR/pipelineruns.json"
            else
                verbose_log "TaskRun $namespace/$run_name 是独立的 TaskRun，检查 TaskRun 状态"
                echo "$result"
            fi
            break
        done
    fi
}

check_and_cleanup_pods_using_pvc() {
    local pvc_name="$1"
    local namespace="$2"

    # 查找引用该 PVC 的所有 Pod
    local using_pods
    using_pods=$(jq -r --arg pvc_name "$pvc_name" --arg ns "$namespace" '
        .items[] |
        select(.metadata.namespace == $ns) |
        select(.spec.volumes[]?.persistentVolumeClaim?.claimName == $pvc_name) |
        {name: .metadata.name, phase: (.status.phase // "Unknown")} |
        [.name, .phase] |
        @tsv
    ' "$TEMP_DIR/pods.json")

    if [[ -z "$using_pods" ]]; then
        # 没有 Pod 使用该 PVC
        return 0
    fi

    # 处理使用该 PVC 的每个 Pod
    local deleted_pods=0
    while IFS=$'\t' read -r pod_name pod_phase; do
        if [[ -z "$pod_name" ]]; then
            continue
        fi

        verbose_log "发现 Pod $namespace/$pod_name (状态: $pod_phase) 正在使用 PVC $namespace/$pvc_name"

        # 如果开启了清理选项且 Pod 是 Completed 状态，则删除
        if [[ "$CLEANUP_COMPLETED_PODS" == "true" && ("$pod_phase" == "Succeeded" || "$pod_phase" == "Failed" || "$pod_phase" == "Completed") ]]; then
            # 安全检查：确保参数不为空
            if [[ -z "$pod_name" || -z "$namespace" ]]; then
                error_log "删除 Pod 参数错误：Pod 名称或命名空间为空"
                continue
            fi

            if [[ "$DRY_RUN" == "true" ]]; then
                log "[DRY-RUN] 将删除占用 PVC 的 Pod: $namespace/$pod_name (状态: $pod_phase)"
                deleted_pods=$((deleted_pods + 1))
            else
                verbose_log "删除占用 PVC 的 Pod: $namespace/$pod_name (状态: $pod_phase)"
                # 安全检查：使用精确的资源类型和名称
                if kubectl delete pod "$pod_name" -n "$namespace" --ignore-not-found=true; then
                    log "已删除占用 PVC 的 Pod: $namespace/$pod_name"
                    deleted_pods=$((deleted_pods + 1))
                else
                    error_log "删除 Pod 失败: $namespace/$pod_name"
                fi
            fi
        else
            if [[ "$pod_phase" == "Running" || "$pod_phase" == "Pending" ]]; then
                verbose_log "Pod $namespace/$pod_name 仍在运行中，跳过 PVC 删除"
            elif [[ "$CLEANUP_COMPLETED_PODS" == "false" ]]; then
                verbose_log "Pod $namespace/$pod_name 已完成但未启用 Pod 清理选项，跳过 PVC 删除"
            fi
        fi
    done <<< "$using_pods"

    # 如果有 Pod 被删除，等待一下让 Kubernetes 更新状态
    if [[ $deleted_pods -gt 0 && "$DRY_RUN" == "false" ]]; then
        verbose_log "等待 Pod 删除完成..."
        sleep 2
    fi

    # 重新检查是否还有 Pod 在使用该 PVC
    local remaining_pods
    remaining_pods=$(jq -r --arg pvc_name "$pvc_name" --arg ns "$namespace" '
        .items[] |
        select(.metadata.namespace == $ns) |
        select(.spec.volumes[]?.persistentVolumeClaim?.claimName == $pvc_name) |
        .metadata.name
    ' "$TEMP_DIR/pods.json")

    if [[ -n "$remaining_pods" && "$deleted_pods" -eq 0 ]]; then
        # 仍有 Pod 占用且没有删除任何 Pod
        return 1
    else
        # 没有 Pod 占用或已清理完成
        return 0
    fi
}

delete_pvc() {
    local pvc_name="$1"
    local namespace="$2"

    # 安全检查：确保参数不为空
    if [[ -z "$pvc_name" || -z "$namespace" ]]; then
        error_log "删除 PVC 参数错误：PVC 名称或命名空间为空"
        return 1
    fi

    # 安全检查：再次验证PVC确实属于Tekton
    local owner_check
    owner_check=$(jq -r --arg name "$pvc_name" --arg ns "$namespace" '
        .items[] |
        select(.metadata.name == $name and .metadata.namespace == $ns) |
        .metadata.ownerReferences[]? |
        select(.kind == "PipelineRun" or .kind == "TaskRun") |
        .kind
    ' "$TEMP_DIR/pvcs.json" | head -1)

    if [[ -z "$owner_check" ]]; then
        error_log "安全检查失败：PVC $namespace/$pvc_name 不属于 Tekton 资源，跳过删除"
        return 1
    fi

    if [[ "$DRY_RUN" == "true" ]]; then
        log "[DRY-RUN] 将删除 PVC: $namespace/$pvc_name (所有者类型: $owner_check)"
        return 0
    fi

    verbose_log "删除 PVC: $namespace/$pvc_name (所有者类型: $owner_check)"
    if kubectl delete pvc "$pvc_name" -n "$namespace" --ignore-not-found=true; then
        log "已删除 PVC: $namespace/$pvc_name"
        return 0
    else
        error_log "删除 PVC 失败: $namespace/$pvc_name"
        return 1
    fi
}

main() {
    # 记录开始时间
    local start_time=$(date +%s)

    parse_args "$@"
    check_prerequisites

    # 设置审计目录
    setup_audit_dir

    log "脚本开始执行时间: $(date '+%Y-%m-%d %H:%M:%S')"

    # 设置退出时清理缓存文件
    trap cleanup_cache_data EXIT INT TERM

    if [[ "$DRY_RUN" == "true" ]]; then
        log "运行在预览模式，不会实际删除任何资源"
    else
        log "⚠️  注意：脚本将实际删除资源"
        if [[ "$CLEANUP_COMPLETED_PODS" == "true" ]]; then
            log "⚠️  将删除已完成状态的Pod (Succeeded/Failed/Completed)"
        fi
        log "⚠️  将删除符合条件的Tekton PVC"
    fi

    log "开始清理 Tekton PVC..."
    log "时间阈值: ${THRESHOLD_MINUTES} 分钟"

    # 一次性缓存所有数据
    local cache_start_time=$(date +%s)
    verbose_log "开始缓存数据: $(date '+%Y-%m-%d %H:%M:%S')"
    cache_all_data
    local cache_end_time=$(date +%s)
    local cache_duration=$((cache_end_time - cache_start_time))
    log "数据缓存完成，耗时: ${cache_duration}秒"

    local total_pvc_count
    total_pvc_count=$(get_total_pvc_count_from_cache)
    local total_tekton_pvc_count
    total_tekton_pvc_count=$(get_tekton_pvc_count_from_cache)
    log "总 PVC 数量: ${total_pvc_count}"
    log "Tekton PVC 数量: ${total_tekton_pvc_count}"
    local tekton_pvc_count=0
    local deleted_count=0
    local error_count=0
    local completed_not_ready_count=0
    local still_running_count=0
    local bound_pvc_count=0

    # 存储大小统计
    local deleted_storage_bytes=0
    local completed_not_ready_storage_bytes=0
    local still_running_storage_bytes=0
    local bound_storage_bytes=0
    local non_tekton_storage_bytes=0

    # 计算非 Tekton PVC 的存储大小
    non_tekton_storage_bytes=$(jq -r '
        def convert_storage(size):
            if (size | test("^[0-9]+Gi$")) then
                (size | gsub("Gi";"") | tonumber) * 1073741824
            elif (size | test("^[0-9]+G$")) then
                (size | gsub("G";"") | tonumber) * 1000000000
            elif (size | test("^[0-9]+Mi$")) then
                (size | gsub("Mi";"") | tonumber) * 1048576
            elif (size | test("^[0-9]+M$")) then
                (size | gsub("M";"") | tonumber) * 1000000
            elif (size | test("^[0-9]+Ki$")) then
                (size | gsub("Ki";"") | tonumber) * 1024
            elif (size | test("^[0-9]+K$")) then
                (size | gsub("K";"") | tonumber) * 1000
            else
                (size | tonumber // 0)
            end;
        [.items[] | select(.metadata.ownerReferences | (. == null or (. | map(.kind) | index("PipelineRun") == null and index("TaskRun") == null))) | .spec.resources.requests.storage // "0" | convert_storage(.)] | add
    ' "${TEMP_DIR}/pvcs.json")

    while IFS=$'\t' read -r pvc_name pvc_namespace owner_kind owner_name creation_time storage_size storage_bytes pvc_phase; do
        if [[ -z "${pvc_name}" ]]; then
            continue
        fi
        verbose_log ""

        tekton_pvc_count=$((tekton_pvc_count + 1))
        if [[ $((tekton_pvc_count % PROGRESS_INTERVAL)) == 0 ]]; then
            log "🔄 已处理 ${tekton_pvc_count}/${total_tekton_pvc_count} 个 Tekton PVC..."
        fi
        verbose_log "检查 PVC: ${pvc_namespace}/${pvc_name} (创建时间: ${creation_time}, 所有者: ${owner_kind}/${owner_name}, 大小: ${storage_size}, 状态: ${pvc_phase})"

        local run_name="${owner_name}"
        local run_type=""
        local completion_time=""

        if [[ "${owner_kind}" == "PipelineRun" ]]; then
            run_type="pipelinerun"
            completion_time=$(get_run_completion_time_from_cache "${run_name}" "${pvc_namespace}" "${run_type}")
        elif [[ "${owner_kind}" == "TaskRun" ]]; then
            run_type="taskrun"
            completion_time=$(get_run_completion_time_from_cache "${run_name}" "${pvc_namespace}" "${run_type}")
        else
            verbose_log "PVC ${pvc_namespace}/${pvc_name} 没有关联的 PipelineRun 或 TaskRun，跳过"
            continue
        fi

        if [[ -z "${completion_time}" ]]; then
            verbose_log "${run_type} ${pvc_namespace}/${run_name} 未完成或不存在，跳过删除 PVC"
            still_running_count=$((still_running_count + 1))
            still_running_storage_bytes=$((still_running_storage_bytes + storage_bytes))
            continue
        fi

        verbose_log "${run_type} ${pvc_namespace}/${run_name} 已完成，完成时间: ${completion_time}"

        # 使用新的统一时间处理函数
        local time_result
        if ! time_result=$(get_timestamp_and_age_info "${completion_time}"); then
            warn_log "PVC ${pvc_namespace}/${pvc_name} 时间解析失败，跳过"
            continue
        fi

        local timestamp age_info is_older
        IFS='|' read -r timestamp age_info is_older <<< "$time_result"

        if [[ "$is_older" == "true" ]]; then
            verbose_log "PVC ${pvc_namespace}/${pvc_name} 符合删除条件（关联的 ${run_type} 已完成 ${age_info}）"

            # 检查是否有 Pod 占用该 PVC
            if check_and_cleanup_pods_using_pvc "${pvc_name}" "${pvc_namespace}"; then
                # 没有 Pod 占用或已清理完成，可以删除 PVC
                if delete_pvc "${pvc_name}" "${pvc_namespace}"; then
                    deleted_count=$((deleted_count + 1))
                    deleted_storage_bytes=$((deleted_storage_bytes + storage_bytes))
                else
                    error_count=$((error_count + 1))
                fi
            else
                # 仍有 Pod 占用，跳过删除
                verbose_log "PVC ${pvc_namespace}/${pvc_name} 仍被 Pod 占用，跳过删除"
                bound_pvc_count=$((bound_pvc_count + 1))
                bound_storage_bytes=$((bound_storage_bytes + storage_bytes))
            fi
        else
            verbose_log "PVC ${pvc_namespace}/${pvc_name} 未超过时间阈值，跳过（关联的 ${run_type} 已完成 ${age_info}）"
            completed_not_ready_count=$((completed_not_ready_count + 1))
            completed_not_ready_storage_bytes=$((completed_not_ready_storage_bytes + storage_bytes))
            # 由于PVC按时间排序，后续PVC都不会超过阈值，可以提前退出
            log "由于 PVC 按创建时间排序，后续 PVC 都不会超过时间阈值，提前结束处理"
            # 统计剩余未处理的 PVC 数量
            local remaining_count=$((total_tekton_pvc_count - tekton_pvc_count))
            if [[ $remaining_count -gt 0 ]]; then
                log "跳过剩余 ${remaining_count} 个 Tekton PVC 的处理"
            fi
            break
        fi

    done < <(extract_tekton_pvcs_from_cache)

    # 记录处理完成时间
    local processing_end_time=$(date +%s)
    local processing_duration=$((processing_end_time - cache_end_time))
    log "PVC 处理完成，耗时: ${processing_duration}秒"
    log "清理完成！"
    log "==============================="
    log "统计结果:"
    log "  总 PVC 数量: ${total_pvc_count}"
    log "  Tekton 相关 PVC: ${tekton_pvc_count}"
    log "  非 Tekton PVC: $((total_pvc_count - tekton_pvc_count))"
    log "  已完成且可清理: ${deleted_count} (大小: $(format_storage_size ${deleted_storage_bytes}))"
    log "  已完成但未到时间: ${completed_not_ready_count} (大小: $(format_storage_size ${completed_not_ready_storage_bytes}))"
    log "  相关 Run 资源仍在运行中: ${still_running_count} (大小: $(format_storage_size ${still_running_storage_bytes}))"
    log "  Pod 占用跳过: ${bound_pvc_count} (大小: $(format_storage_size ${bound_storage_bytes}))"
    log "  非 Tekton PVC 总大小: $(format_storage_size "${non_tekton_storage_bytes}")"
    if [[ ${error_count} -gt 0 ]]; then
        log "  删除失败: ${error_count}"
    fi
    log "==============================="
    if [[ "${DRY_RUN}" == "true" ]]; then
        log "预览结果: 找到 ${deleted_count} 个 PVC 可以删除，总大小 $(format_storage_size ${deleted_storage_bytes})"
    else
        log "实际删除了 ${deleted_count} 个 PVC，释放存储 $(format_storage_size ${deleted_storage_bytes})"
    fi

    # 记录脚本结束时间和总耗时
    local end_time=$(date +%s)
    local total_duration=$((end_time - start_time))
    local minutes=$((total_duration / 60))
    local seconds=$((total_duration % 60))
    log "脚本结束执行时间: $(date '+%Y-%m-%d %H:%M:%S')"
    # log "脚本总执行时间: ${total_duration} 秒"
    if [ $minutes -gt 0 ]; then
        log "⏱️ 总耗时: ${minutes}分${seconds}秒"
    else
        log "⏱️ 总耗时: ${seconds}秒"
    fi

    # 保存执行摘要到审计目录
    local summary_file="${AUDIT_SESSION_DIR}/summary.txt"
    local namespace_info
    if [[ "$ALL_NAMESPACES" == "true" ]]; then
        namespace_info="所有命名空间"
    else
        namespace_info=$(IFS=','; echo "${NAMESPACES[*]}")
    fi

    cat > "$summary_file" << EOF
脚本执行摘要 - $(date '+%Y-%m-%d %H:%M:%S')
=============================================
执行参数:
  命名空间: ${namespace_info}
  时间阈值: ${THRESHOLD_MINUTES} 分钟
  预览模式: ${DRY_RUN}
  清理Pod: ${CLEANUP_COMPLETED_PODS}
  保留缓存: ${KEEP_CACHE}

执行结果:
  总 PVC 数量: ${total_pvc_count}
  Tekton 相关 PVC: ${tekton_pvc_count}
  已完成且可清理: ${deleted_count} (大小: $(format_storage_size ${deleted_storage_bytes}))
  已完成但未到时间: ${completed_not_ready_count} (大小: $(format_storage_size ${completed_not_ready_storage_bytes}))
  相关 Run 资源仍在运行中: ${still_running_count} (大小: $(format_storage_size ${still_running_storage_bytes}))
  Pod 占用跳过: ${bound_pvc_count} (大小: $(format_storage_size ${bound_storage_bytes}))
  删除失败: ${error_count}

执行时间: ${total_duration} 秒
EOF

    log "执行摘要已保存到: $summary_file"

    if [[ $error_count -gt 0 ]]; then
        exit 1
    fi
}

main "$@"
