# DevOps Roadmap (2026Q2)

A detailed capability framework for the 2026Q2 quarter.

### Domains

| # | Domain | Epics Count |
|---|--------|-------------|
| 1 | Tool Integration | 12 |
| 2 | CI/CD | 16 |
| 3 | Tool Deployment | 7 |
| 4 | AI-Augmented SDLC | 1 |
| 5 | DevOps - RFEs & NFR | 6 |

---

## Domain 1: Tool Integration

### Milestone: Artifact Promotion and Cleanup throught connectors

| # | Capability | Scope | Components |
|---|------------|-------|------------|
| 1.1 | [DEVOPS-43352](https://jira.alauda.cn/browse/DEVOPS-43352) Image Promotion Requirements and Product Planning | Gather customer requirements for image promotion enhancements, including UX improvements, search, tag visibility, and integrating Harbor vulnerability scanning to prevent promoting vulnerable images. | N/A |
| 1.2 | [DEVOPS-42732](https://jira.alauda.cn/browse/DEVOPS-42732) Artifact Cleanup - Harbor Cleanup - Best Practice [KB] | Output a knowledge base guide on implementing best practices for artifact lifecycle management and automated cleanup within the SDLC using native Harbor functionalities. | N/A |
| 1.3 | [DEVOPS-42608](https://jira.alauda.cn/browse/DEVOPS-42608) Artifact Promotion - Pipeline Template + ApprovalRequest + Docs | Offer an out-of-the-box artifact promotion solution using Skopeo copy tasks, target connector approvals, pipeline templates, and Kyverno constraint policies for secure cross-registry syncs. | connectors-operator |

### Milestone: Connectors security enhancement

| # | Capability | Scope | Components |
|---|------------|-------|------------|
| 1.4 | [DEVOPS-43567](https://jira.alauda.cn/browse/DEVOPS-43567) Connector Operator Out-of-the-box deployment | Provide an out-of-the-box deployment setup for the Connector Operator. | connectors-operator |
| 1.5 | [DEVOPS-42720](https://jira.alauda.cn/browse/DEVOPS-42720) Connector end-to-end encryption | Improve security between connector components to prevent malicious attacks by implementing mTLS/end-to-end encryption across core services. | connectors-operator |
| 1.6 | [DEVOPS-41818](https://jira.alauda.cn/browse/DEVOPS-41818) oAuth2 App support for Authentication | Add OAuth2 application support for enhanced authentication in connectors. | connectors-operator |

### Milestone: Container Platform - Connectors Integration

| # | Capability | Scope | Components |
|---|------------|-------|------------|
| 1.7 | [DEVOPS-43573](https://jira.alauda.cn/browse/DEVOPS-43573) Container Platform - Code repository integration | Integrate external code repositories directly into the container platform. | N/A |
| 1.8 | [DEVOPS-43572](https://jira.alauda.cn/browse/DEVOPS-43572) Container Platform - Image repository integration | Integrate external image repositories directly into the container platform. | N/A |

### Milestone: v3 to v4 Integrations Migration

| # | Capability | Scope | Components |
|---|------------|-------|------------|
| 1.9 | [DEVOPS-43559](https://jira.alauda.cn/browse/DEVOPS-43559) Automatic Project, Connector and Secret creation (Nexus, SonarQube) | Automate the creation and configuration of projects, connectors, and secrets for Nexus and SonarQube integrations. | connectors-operator |
| 1.10 | [DEVOPS-43544](https://jira.alauda.cn/browse/DEVOPS-43544) v3 to v4 integrations migration | Provide migration CLIs, scripts, and documentation to help users import legacy v3 integrations into the new v4 connectors system. | connectors-operator |
| 1.11 | [DEVOPS-43562](https://jira.alauda.cn/browse/DEVOPS-43562) Artifact Promotion migration | Provide migration support specifically for transitioning artifact promotion workflows from v3 to v4. | N/A |
| 1.12 | [DEVOPS-43584](https://jira.alauda.cn/browse/DEVOPS-43584) Jira, S3 Connectors | Introduce and fully support native connectors for Atlassian Jira and AWS S3 storage. | connectors-operator |

---

## Domain 2: CI/CD

### Milestone: Tekton Pipelines - Restricted Security Policy Adaptation / Tekton Hub Deprecation

| # | Capability | Scope | Components |
|---|------------|-------|------------|
| 2.1 | [DEVOPS-43599](https://jira.alauda.cn/browse/DEVOPS-43599) Tekton Pipelines - Restricted Security Policy Adaptation for Buildah Task | Adapt the Buildah task to comply with restricted security policies in OpenShift. | tektoncd-operator |
| 2.2 | [DEVOPS-43600](https://jira.alauda.cn/browse/DEVOPS-43600) Tekton Pipelines - Replace Tekton Hub with Artifact Hub | Migrate pipeline resources from the deprecated Tekton Hub to Artifact Hub. | tektoncd-operator |
| 2.3 | [DEVOPS-43601](https://jira.alauda.cn/browse/DEVOPS-43601) Tekton Pipelines - New Web Console Migration | Migrate user experience to the new Tekton Web Console. | tektoncd-operator |
| 2.4 | [DEVOPS-43602](https://jira.alauda.cn/browse/DEVOPS-43602) Tekton Pipelines - Usability Improvements (UI Optimization) | Improve overall usability for pipeline operations and optimize the UI based on user feedback. | tektoncd-operator |

### Milestone: Tekton Pipelines - Enhance DevOps v3 feature parity for internal migration

| # | Capability | Scope | Components |
|---|------------|-------|------------|
| 2.5 | [DEVOPS-43595](https://jira.alauda.cn/browse/DEVOPS-43595) Tekton Pipelines - v3 to v4 Migration Research (Internal Edge Environment) | Research and plan the migration strategy from DevOps v3 to v4 for internal edge environments. | tektoncd-operator |
| 2.6 | [DEVOPS-37058](https://jira.alauda.cn/browse/DEVOPS-37058) Tekton Pipelines - Artifact Version Generation Strategy (git-version task) | Support component versioning strategies during the build process to align with legacy v3 capabilities. | tektoncd-operator |
| 2.7 | [DEVOPS-43596](https://jira.alauda.cn/browse/DEVOPS-43596) Tekton Pipelines - Enhance Dynamic Forms for Pipeline Orchestration | Improve the dynamic form experience for pipeline orchestration to make it more intuitive. | tektoncd-operator |
| 2.8 | [DEVOPS-41826](https://jira.alauda.cn/browse/DEVOPS-41826) Tekton Pipelines - Pipeline Constraint Strategies | Add support for constraint strategies and execution policies in Tekton pipelines. | N/A |
| 2.9 | [DEVOPS-42698](https://jira.alauda.cn/browse/DEVOPS-42698) Tekton Pipelines - Serial/Parallel Execution Strategy | Implement advanced pipeline execution strategies allowing serial and parallel task execution. | tektoncd-operator |
| 2.10 | [DEVOPS-43597](https://jira.alauda.cn/browse/DEVOPS-43597) Tekton Pipelines - Pipeline Policy Constraints (Docs, based on Kyverno) | Document how to implement pipeline policy constraints using Kyverno. | tektoncd-operator |

### Milestone: Tekton Pipelines - Enhance alignment with OCP capabilities

| # | Capability | Scope | Components |
|---|------------|-------|------------|
| 2.11 | [DEVOPS-43591](https://jira.alauda.cn/browse/DEVOPS-43591) Tekton Pipelines - PAC Repository Management | Introduce management capabilities for Pipeline as Code (PAC) repositories. | tektoncd-operator |
| 2.12 | [DEVOPS-43592](https://jira.alauda.cn/browse/DEVOPS-43592) Tekton Pipelines - Upgrade Tekton to v1.12 | Upgrade the underlying Tekton component to version 1.12. | tektoncd-operator |
| 2.13 | [DEVOPS-42731](https://jira.alauda.cn/browse/DEVOPS-42731) Tekton Pipelines - Image Synchronization Support (Skopeo) | Add out-of-the-box support for image synchronization via Skopeo tasks to align with OCP GitOps release templates. | tektoncd-operator |
| 2.14 | [DEVOPS-43593](https://jira.alauda.cn/browse/DEVOPS-43593) Tekton Pipelines - PAC Integration with GitHub (Docs) | Document how to integrate Pipeline as Code (PAC) with GitHub. | tektoncd-operator |
| 2.15 | [DEVOPS-43594](https://jira.alauda.cn/browse/DEVOPS-43594) Tekton Pipelines - tkn CLI Usage Documentation | Provide comprehensive documentation for using the `tkn` CLI. | tektoncd-operator |
| 2.16 | [DEVOPS-42717](https://jira.alauda.cn/browse/DEVOPS-42717) Tekton Pipelines - Pipeline Caching | Implement robust pipeline caching mechanisms for Tekton to accelerate build times, aligning with Red Hat OpenShift Pipelines standards. | N/A |

---

## Domain 3: Tool Deployment

### Milestone: AI-Augmented Toolchain Incident Resolution

| # | Capability | Scope | Components |
|---|------------|-------|------------|
| 3.1 | [DEVOPS-43578](https://jira.alauda.cn/browse/DEVOPS-43578) Gitlab Skills | Develop AI agent skills tailored for Gitlab incident resolution and operational guidance. | gitlab-ce-operator |
| 3.2 | [DEVOPS-43579](https://jira.alauda.cn/browse/DEVOPS-43579) Harbor Skills | Develop AI agent skills tailored for Harbor incident resolution and operational guidance. | harbor-ce-operator |
| 3.3 | [DEVOPS-43580](https://jira.alauda.cn/browse/DEVOPS-43580) SonarQube Skills | Develop AI agent skills tailored for SonarQube incident resolution and operational guidance. | sonarqube-ce-operator |

### Milestone: OCP Toolchain

| # | Capability | Scope | Components |
|---|------------|-------|------------|
| 3.4 | [DEVOPS-43570](https://jira.alauda.cn/browse/DEVOPS-43570) Harbor vs Quay - Capabilities comparison | Conduct a comprehensive technical comparison of capabilities between Harbor and Red Hat Quay registries. | harbor-ce-operator |

### Milestone: Production Ready - SonarQube baseline storage and network

| # | Capability | Scope | Components |
|---|------------|-------|------------|
| 3.5 | [DEVOPS-41443](https://jira.alauda.cn/browse/DEVOPS-41443) Define SonarQube production baselines (Storage, Network, CPU) | Define and document the performance baselines and minimum requirements for Storage, Network, and CPU for running SonarQube in production across varying user scales. | sonarqube-ce-operator |

### Milestone: Toolchain Upgrades

| # | Capability | Scope | Components |
|---|------------|-------|------------|
| 3.6 | [DEVOPS-43575](https://jira.alauda.cn/browse/DEVOPS-43575) Gitlab Upgrade to 18.8 | Upgrade the platform's Gitlab component integration to version 18.8. | gitlab-ce-operator |
| 3.7 | [DEVOPS-43574](https://jira.alauda.cn/browse/DEVOPS-43574) Harbor Update to 2.16 | Upgrade the platform's Harbor component integration to version 2.16. | harbor-ce-operator |

---

## Domain 4: AI-Augmented SDLC

### Milestone: SRE Agent 1.0 - DBS Design Partner

| # | Capability | Scope | Components |
|---|------------|-------|------------|
| 4.1 | [DEVOPS-43535](https://jira.alauda.cn/browse/DEVOPS-43535) SRE Agent pre-alpha - DBS Design Partner Onboarding | Officially land and implement the pre-alpha SRE Agent with DBS as a core design partner. | N/A |

---

## Domain 5: DevOps - RFEs & NFR

### Milestone: DevOps AI-Augmented processes

| # | Capability | Scope | Components |
|---|------------|-------|------------|
| 5.1 | [DEVOPS-43547](https://jira.alauda.cn/browse/DEVOPS-43547) DevOps AI-Augmented Processes | Enhance the DevOps team's process by integrating AI-native tooling and skills, enabling individuals to generate value end-to-end and incubate new AI-augmented workflows. | N/A |

### Milestone: DevOps L5 Plugin Release

| # | Capability | Scope | Components |
|---|------------|-------|------------|
| 5.2 | [DEVOPS-43515](https://jira.alauda.cn/browse/DEVOPS-43515) DevOps Plugin Release | Execute the standard release process for DevOps L5 plugins. | N/A |
| 5.3 | [DEVOPS-43566](https://jira.alauda.cn/browse/DEVOPS-43566) DORA + Release Efficiency Metrics Collection | Collect and analyze actual release metrics, including lead time from release candidate to final publication, phase duration breakdowns, and human effort costs. | N/A |

### Milestone: Engineering improvements

| # | Capability | Scope | Components |
|---|------------|-------|------------|
| 5.4 | [DEVOPS-43588](https://jira.alauda.cn/browse/DEVOPS-43588) Connectors-Extensions Repository Restructuring | Refactor the `connectors-extensions` repository structure to be lighter, grouping by component presence to reduce unnecessary pipeline executions and lower delivery costs. | connectors-operator |
| 5.5 | [DEVOPS-43521](https://jira.alauda.cn/browse/DEVOPS-43521) DevOps L5 Plugin Test Automation Coverage | Achieve 90% test automation coverage for the DevOps L5 plugins. | N/A |
| 5.6 | [DEVOPS-43581](https://jira.alauda.cn/browse/DEVOPS-43581) Tooling NFR Automation | Automate the Non-Functional Requirements (NFR) testing for internal tooling. | N/A |
