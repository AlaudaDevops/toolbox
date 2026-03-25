# Alauda Capability Ownership

Mapping of capability ownership to product teams, aligned with the OpenShift capability taxonomy (`knowledge/openshift-capability-taxonomy.md`).

## Teams

| Team | Scope | # Capabilities |
|------|-------|---------------|
| **Infrastructure** | Horizontal platform substrate and platform lifecycle: K8s runtime, compute, node OS, bare metal management, install/upgrade/config, HCP, backup/DR, disconnected ops, extension framework, cluster lifecycle, multi-cluster management (policy, placement, fleet observability, cost), security (IAM, runtime security, compliance, secrets, policy engine), observability (metrics, logging, alerting) | 24 |
| **Container Platform** | Networking, storage, registry, distributed tracing, network observability, GitOps, builds, application modernization, application workload orchestration, service mesh, serverless, virtualization | 15 |
| **DevOps** | Developer experience and delivery: CI/CD pipelines, developer portal, developer tooling, supply chain security | 4 |
| **AI** | AI/ML capabilities: GPU infrastructure, model training/serving, MLOps/LLMOps, AI-powered operations (HyperFlux) | 5 |
| **Ecosystem** | Third-party and operator-managed services: API management, messaging, databases, caches, application runtimes, integration | 5 |
| **Platform Team** | Internal enabling team — shared engineering infrastructure, build systems, internal tooling. Does not own customer-facing capabilities. | 0 |

## Capability Ownership Matrix

Each capability (L2) from the taxonomy has exactly one owning team.

### Domain 1: Infrastructure

| # | Capability | Owning Team |
|---|-----------|-------------|
| 1.1 | Kubernetes Runtime | Infrastructure |
| 1.2 | Compute & Node Management | Infrastructure |
| 1.3 | Networking | Container Platform |
| 1.4 | Storage | Container Platform |
| 1.5 | Container Registry | Container Platform |
| 1.6 | Node Operating System | Infrastructure |
| 1.7 | Bare Metal Management | Infrastructure |

### Domain 2: Lifecycle & Operations

| # | Capability | Owning Team |
|---|-----------|-------------|
| 2.1 | Installation & Provisioning | Infrastructure |
| 2.2 | Platform Updates | Infrastructure |
| 2.3 | Platform Configuration | Infrastructure |
| 2.4 | Hosted Control Planes | Infrastructure |
| 2.5 | Backup & Disaster Recovery | Infrastructure |
| 2.6 | Disconnected Operations | Infrastructure |
| 2.7 | Extension & Operator Ecosystem | Infrastructure |

### Domain 3: Multi-Cluster Management

| # | Capability | Owning Team |
|---|-----------|-------------|
| 3.1 | Cluster Lifecycle | Infrastructure |
| 3.2 | Policy & Governance | Infrastructure |
| 3.3 | Workload Placement | Infrastructure |
| 3.4 | Fleet Observability | Infrastructure |
| 3.5 | Cost Management | Infrastructure |

### Domain 4: Security & Compliance

| # | Capability | Owning Team |
|---|-----------|-------------|
| 4.1 | Identity & Access Management | Infrastructure |
| 4.2 | Runtime Security | Infrastructure |
| 4.3 | Supply Chain Security | DevOps |
| 4.4 | Compliance & Audit | Infrastructure |
| 4.5 | Secrets, Certificates & Workload Identity | Infrastructure |
| 4.6 | Policy Engine | Infrastructure |

### Domain 5: Observability

| # | Capability | Owning Team |
|---|-----------|-------------|
| 5.1 | Metrics & Monitoring | Infrastructure |
| 5.2 | Logging | Infrastructure |
| 5.3 | Distributed Tracing | Container Platform |
| 5.4 | Network Observability | Container Platform |
| 5.5 | Alerting & Incident Management | Infrastructure |

### Domain 6: Developer Experience & Delivery

| # | Capability | Owning Team |
|---|-----------|-------------|
| 6.1 | CI/CD Pipelines | DevOps |
| 6.2 | GitOps & Continuous Delivery | Container Platform |
| 6.3 | Builds | Container Platform |
| 6.4 | Developer Portal & Self-Service | DevOps |
| 6.5 | Developer Tooling | DevOps |
| 6.6 | Application Modernization | Container Platform |
| 6.7 | Application Workload Orchestration | Container Platform |

### Domain 7: Application Services

| # | Capability | Owning Team |
|---|-----------|-------------|
| 7.1 | Service Mesh | Container Platform |
| 7.2 | Serverless | Container Platform |
| 7.3 | API Management | Ecosystem |
| 7.4 | Messaging & Streaming | Ecosystem |
| 7.5 | Data Services | Ecosystem |
| 7.6 | Application Runtimes | Ecosystem |
| 7.7 | Integration | Ecosystem |

### Domain 8: Virtualization

| # | Capability | Owning Team |
|---|-----------|-------------|
| 8.1 | VM Lifecycle | Container Platform |
| 8.2 | VM Live Migration | Container Platform |
| 8.3 | VM Networking & Storage | Container Platform |
| 8.4 | VM Migration from External Platforms | Container Platform |

### Domain 9: AI & Intelligent Platform

| # | Capability | Owning Team |
|---|-----------|-------------|
| 9.1 | AI Infrastructure | AI |
| 9.2 | Model Training & Experimentation | AI |
| 9.3 | Model Serving & Inference | AI |
| 9.4 | MLOps & LLMOps | AI |
| 9.5 | AI-Powered Operations | AI |

## Notes

- **Infrastructure is the largest team by capability count (24/52).** It owns the entire platform lifecycle (Domain 2), all multi-cluster management (Domain 3), most of security (Domain 4 except supply chain), and most of observability (Domain 5 except tracing and network observability).
- **Container Platform owns capabilities across Domains 1, 5, 6, 7, 8.** It is the home of the networking team (Kube-OVN), storage, registry, and first-party integrated services (service mesh, serverless, virtualization, GitOps, builds).
- **DevOps is focused on the developer-facing delivery pipeline** — CI/CD, developer portal, tooling, and supply chain security (scanning/signing in the build pipeline).
- **Capability domains ≠ team boundaries.** The taxonomy is organized by *what the platform does*; the teams are organized by *how Alauda builds it*. Infrastructure spans Domains 1-5. Container Platform spans Domains 1, 5, 6, 7, 8.
- **Platform Team** is an internal enabling team with no customer-facing capability ownership.
