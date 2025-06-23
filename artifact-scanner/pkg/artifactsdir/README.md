# Artifacts Directory Structure

The directory should contain plugins with the following structure as defined in alauda/artifacts:

```bash
artifacts
├── demo-operator
│   ├── artifacts.yaml
│   ├── metadata.yaml
│   └── versions.yaml
├── demo-operator-test
│   └── versions.yaml
├── tektoncd-operator
│   ├── artifacts.yaml
│   ├── metadata.yaml
│   └── versions.yaml
```

The directory name is treated as the plugin name.

## metadata.yaml

Contains plugin metadata:

```yaml
packageType: ModulePlugin
owners:
- email: user1@example.com
channels:
- channel: default
  defaultChannel: true
  repository: devops/chart-harbor-robot-gen
  stage: alpha
```

```yaml
packageType: OperatorBundle
owners:
  - email: user1@example.com
channels:
  - channel: alpha
    defaultChannel: true
    repository: devops/demo-operator-bundle
    stage: alpha
```

The `packageType` can be `ModulePlugin` or `OperatorBundle`. The metadata.yaml defines repositories for different channels.

## versions.yaml

Contains version information for each channel:

```yaml
alpha: v1.1.0-beta.126.gf70d7e4
```

## artifacts.yaml

Contains artifact definitions for different channels:

```yaml
channels:
  - channel: stable
    version: v2.13.0-beta.56.g8b08d33
    artifacts:
      - repository: devops/demo-operator-bundle
        tag: v2.13.0-beta.56.g8b08d33
        digest: sha256:1fe8ea7e226395bf16525bba630571fb847441fe65add8e6e8c5c7470200dd53
        type: Bundle
```

Artifact types can be `Bundle`, `Image`, or `Chart`.