# OpenShift Capability Taxonomy

A vendor-neutral capability framework for analyzing the Red Hat OpenShift ecosystem. Designed for competitive comparison with ACP, roadmap analysis, and tracking OpenShift strategic priorities.

**Structure:** Domain (L1) → Capability (L2) → Feature (L3, not enumerated here)
**Cross-cutting dimensions:** Edge/Telco/SNO, Disconnected/Air-gapped compatibility, Multi-architecture, FIPS/Compliance baselines, Bare metal compatibility, SaaS dependency

### Domains

| # | Domain | L2 Count |
|---|--------|----------|
| 1 | Infrastructure | 7 |
| 2 | Lifecycle & Operations | 7 |
| 3 | Multi-Cluster Management | 5 |
| 4 | Security & Compliance | 6 |
| 5 | Observability | 5 |
| 6 | Developer Experience & Delivery | 7 |
| 7 | Application Services | 7 |
| 8 | Virtualization | 4 |
| 9 | AI & Intelligent Platform | 5 |
| | **Total** | **53** |

---

## Domain 1: Infrastructure

The compute, network, and storage substrate the platform runs on.

| # | Capability | Scope | OCP Mapping |
|---|-----------|-------|-------------|
| 1.1 | Kubernetes Runtime | K8s version currency, API conformance, core workload primitives (pods, deployments, statefulsets, jobs, CRDs), resource model, service discovery, etcd | CRI-O, etcd, kube-apiserver, Kubernetes conformance |
| 1.2 | Compute & Node Management | Node lifecycle, machine sets, cluster-level autoscaling (node add/remove), node scheduling configuration, multi-arch (ARM/Power/Z), Windows containers, node tuning, performance profiles, scalability testing | Machine API, Nodes, NTO, ClusterAutoscaler |
| 1.3 | Networking | CNI, network policy, ingress/routes, Gateway API, load balancing, DNS, multi-network (Multus), hardware networking (SR-IOV, DPDK), cross-cluster service connectivity | OVN-Kubernetes, MetalLB, NMState, Multus, Service Interconnect (Skupper) |
| 1.4 | Storage | CSI drivers, persistent volume lifecycle, snapshots, cloning, software-defined storage, local storage | ODF, CSI, LSO |
| 1.5 | Container Registry | Image registry, image management, scanning, geo-replication | Internal Registry, Quay |
| 1.6 | Node Operating System | Immutable/container-optimized OS, atomic updates, node image management | RHCOS, rpm-ostree |
| 1.7 | Bare Metal Management | Physical server provisioning, BMC/IPMI/Redfish control, hardware introspection, host lifecycle management | Bare Metal Operator, Metal3, Ironic |

## Domain 2: Lifecycle & Operations

How the platform itself is installed, updated, configured, and recovered — day 0 through day N.

| # | Capability | Scope | OCP Mapping |
|---|-----------|-------|-------------|
| 2.1 | Installation & Provisioning | Installer (IPI/UPI), agent-based, assisted, cloud provider integration, multi-platform install targets | Installer, Assisted Installer |
| 2.2 | Platform Updates | Over-the-air upgrades, version lifecycle (EUS), upgrade paths, rollback | Cincinnati, OTA, EUS |
| 2.3 | Platform Configuration | Post-install config, machine config, cluster settings, certificate rotation | MCO, post-install config |
| 2.4 | Hosted Control Planes | Control plane as workload, management cluster topology, cost efficiency | HyperShift |
| 2.5 | Backup & Disaster Recovery | Platform backup, etcd backup, cluster restore, DR strategies | OADP, etcd backup |
| 2.6 | Disconnected Operations | Air-gapped install, mirror registry, content mirroring, disconnected updates | oc-mirror, disconnected environments |
| 2.7 | Extension & Operator Ecosystem | Operator lifecycle management, marketplace/catalog, operator SDK, plugin framework, cluster extensions | OLM, OperatorHub, Operator SDK |

## Domain 3: Multi-Cluster Management

Fleet-level governance, lifecycle, and visibility across clusters.

| # | Capability | Scope | OCP Mapping |
|---|-----------|-------|-------------|
| 3.1 | Cluster Lifecycle | Provision, import, upgrade, decommission clusters across fleet, cross-cluster workload migration | ACM, MCE, MTC |
| 3.2 | Policy & Governance | Configuration policies, compliance enforcement, security baselines across fleet | ACM Policy Framework |
| 3.3 | Workload Placement | Scheduling workloads across clusters, cluster sets, placement rules | ACM placement |
| 3.4 | Fleet Observability | Aggregated metrics, fleet health, cross-cluster visibility | ACM Observability |
| 3.5 | Cost Management | Cost visibility, resource optimization, right-sizing recommendations | Cost Management, ACM right-sizing |

## Domain 4: Security & Compliance

Identity, threat protection, supply chain integrity, and regulatory compliance.

| # | Capability | Scope | OCP Mapping |
|---|-----------|-------|-------------|
| 4.1 | Identity & Access Management | Authentication (OAuth, OIDC, LDAP), RBAC, multi-tenancy isolation (projects, quotas, limit ranges), SCC/PSS | OAuth server, SCC, RBAC |
| 4.2 | Runtime Security | Threat detection, behavioral analysis, network segmentation, vulnerability management, workload isolation (sandboxed containers, confidential computing) | ACS (StackRox), Sandboxed Containers, CoCo |
| 4.3 | Supply Chain Security | Image signing, SBOM, build provenance, vulnerability scanning, compliance gates | Trusted Software Supply Chain, Quay/Clair |
| 4.4 | Compliance & Audit | Compliance scanning (CIS, NIST, PCI), audit logging, gap reporting, auto-remediation | Compliance Operator, audit logs |
| 4.5 | Secrets, Certificates & Workload Identity | Certificate lifecycle, trust distribution, workload identity, external secrets, encryption at rest, key management | cert-manager, trust-manager, Zero Trust Workload Identity Manager, External Secrets Operator |
| 4.6 | Policy Engine | Admission control, policy enforcement, governance rules | Gatekeeper, ACS policies |

## Domain 5: Observability

Visibility into platform and application health, performance, and cost.

| # | Capability | Scope | OCP Mapping |
|---|-----------|-------|-------------|
| 5.1 | Metrics & Monitoring | Metrics collection, dashboards, user workload monitoring, power monitoring/sustainability | Prometheus, COO, Kepler |
| 5.2 | Logging | Log collection, aggregation, storage, querying | Vector, Loki, Logging Operator |
| 5.3 | Distributed Tracing | Trace collection, analysis, sampling, correlation | OpenTelemetry, Tempo/Jaeger |
| 5.4 | Network Observability | eBPF flow collection, topology visualization, DNS tracking, packet capture | Network Observability Operator |
| 5.5 | Alerting & Incident Management | Alert rules, routing, notification, incident detection and correlation | Alertmanager, Lightspeed integration |

## Domain 6: Developer Experience & Delivery

How developers build, ship, and operate applications on the platform.

| # | Capability | Scope | OCP Mapping |
|---|-----------|-------|-------------|
| 6.1 | CI/CD Pipelines | Pipeline definition, execution, triggers, pipelines-as-code | OpenShift Pipelines (Tekton) |
| 6.2 | GitOps & Continuous Delivery | Declarative app delivery, sync, drift detection, progressive rollout | OpenShift GitOps (Argo CD) |
| 6.3 | Builds | Container image build strategies, source-to-image, cloud-native builds | Builds (Shipwright), S2I |
| 6.4 | Developer Portal & Self-Service | Software catalog, templates, scaffolding, onboarding, RBAC | Developer Hub (Backstage) |
| 6.5 | Developer Tooling | CLI, cloud IDE, local dev environment, SDK | oc, Dev Spaces, OpenShift Local |
| 6.6 | Application Modernization | Migration analysis, refactoring guidance, containerization tooling | MTA, MTR |
| 6.7 | Application Workload Orchestration | Application scaling (HPA, VPA, KEDA), deployment strategies, pod disruption budgets, workload priority & preemption, descheduler, in-place pod resource resize, service unidling | HPA, VPA, KEDA, PDB, kube-descheduler, Deployment/DeploymentConfig strategies |

## Domain 7: Application Services

Middleware and runtime services consumed by applications.

| # | Capability | Scope | OCP Mapping |
|---|-----------|-------|-------------|
| 7.1 | Service Mesh | Traffic management, mTLS, observability, policy, canary/blue-green | OpenShift Service Mesh (Istio) |
| 7.2 | Serverless | Scale-to-zero compute, event-driven architecture, functions | OpenShift Serverless (Knative) |
| 7.3 | API Management | API gateway, rate limiting, developer portal, analytics | 3scale, Connectivity Link |
| 7.4 | Messaging & Streaming | Message queues, event streaming, pub/sub | AMQ, AMQ Streams (Kafka) |
| 7.5 | Data Services | Managed databases, caches, stream processing | Limited in OCP; ACP strength |
| 7.6 | Application Runtimes | Supported language runtimes, application servers, frameworks, application identity (SSO) | Quarkus, JBoss EAP, Spring Boot, Keycloak |
| 7.7 | Integration | Application integration, connectors, ETL, event routing | Camel, Fuse, Service Registry |

## Domain 8: Virtualization

Running and migrating VM workloads alongside containers.

| # | Capability | Scope | OCP Mapping |
|---|-----------|-------|-------------|
| 8.1 | VM Lifecycle | Create, configure, run, snapshot, clone, template VMs | OpenShift Virtualization (KubeVirt) |
| 8.2 | VM Live Migration | Cross-node migration, CPU rebalancing, zero-downtime maintenance | KubeVirt live migration |
| 8.3 | VM Networking & Storage | VM-specific networking, storage classes, passthrough, SR-IOV for VMs | Multus, ODF for VMs |
| 8.4 | VM Migration from External Platforms | Migrate VMs from VMware/other hypervisors, bulk migration, storage offload | MTV |

## Domain 9: AI & Intelligent Platform

AI as workload (run AI on the platform) and AI as operator (AI runs the platform).

| # | Capability | Scope | OCP Mapping |
|---|-----------|-------|-------------|
| 9.1 | AI Infrastructure | GPU/accelerator scheduling, RDMA networking, DRA, heterogeneous compute | GPU Operator, Node Feature Discovery |
| 9.2 | Model Training & Experimentation | Distributed training, notebooks, experiment tracking, fine-tuning | OpenShift AI (Jupyter, KubeFlow) |
| 9.3 | Model Serving & Inference | Serving frameworks, autoscaling, model runtimes, inference optimization | KServe, vLLM, llm-d |
| 9.4 | MLOps & LLMOps | Model registry, pipelines, monitoring, evaluation, AIBOM | OpenShift AI model registry |
| 9.5 | AI-Powered Operations | AI assistant, natural language troubleshooting, incident detection, recommendations | Lightspeed, Insights |

---

## Cross-Cutting Dimensions

These are not domains — they are constraints or deployment contexts that apply across multiple domains. Each capability (L2) can be evaluated against these dimensions.

| Dimension | Examples |
|-----------|---------|
| Edge / Telco / SNO | Single-node OpenShift, MicroShift, telco RAN profiles, low-latency tuning |
| Disconnected / Air-gapped Compatibility | Features that require special handling without internet access (offline catalogs, registry mirroring, update graphs, etc.) |
| Multi-Architecture | ARM, IBM Power, IBM Z, heterogeneous clusters |
| FIPS / Compliance Baselines | FIPS 140-3, FedRAMP, common criteria |
| Bare Metal Compatibility | Features that require special handling on bare metal (no cloud API, no cloud LB, etc.) |
| SaaS Dependency | Features that require or benefit from cloud-hosted services (Hybrid Cloud Console, Insights, Assisted Installer, OCM, cost management SaaS) — relevant for fully self-contained on-premises deployments |

---

## Methodology

**Based on:** OCP 4.21 (latest GA as of March 2026)

### Sources

This taxonomy was built by cross-referencing multiple authoritative sources. No single source is sufficient — each has strengths and blind spots:

| Source | What it's good for | What it misses |
|--------|-------------------|----------------|
| **OCP 4.21 Documentation TOC** — 19 top-level categories, ~50 documentation guides (scraped live from docs.redhat.com) | Most stable view of capability structure; reflects how Red Hat expects users to navigate | Add-on products (ACM, ACS, Quay) have separate doc sets not visible here |
| **OCP 4.21 Release Notes** — 20 feature categories under "New features and enhancements" (cross-referenced with 4.18 for stability) | Most granular engineering view; reveals how Red Hat tracks features internally | Only shows what changed in a given release; quiet areas disappear |
| **Red Hat product portfolio** — ~25 distinct OpenShift-related products (web research, redhat.com/en/technologies/all-products) | Covers add-on products outside OCP core (ACM, ACS, Quay, ODF, OpenShift AI, RHDH, etc.) | Mixes platforms, products, suites, SKUs, tools — not a capability taxonomy |
| **OpenShift 5 Roadmap** — 7 strategic pillars from Red Hat Intent to Release | Forward-looking; shows where Red Hat is investing | Omits mature/stable areas that aren't getting new investment |
| **What's New in OpenShift 4.11-4.21** — stable section headings across 11 releases (analyzed by Zhou Chao) | Best signal for long-term stable capability groupings | Marketing-influenced; section names drift with narrative |
| **What's Next in OpenShift Q4CY2022-Q4CY2025** — strategic roadmap decks (analyzed by Zhou Chao) | Shows which areas Red Hat elevates to strategic priority | Structure changes every quarter; not suitable as taxonomy backbone |
| **Zhou Chao's v11 spreadsheet** — 192 features, 22 modules, 41 products, 207 source objects | Most complete bottom-up enumeration of OCP capabilities | Mixes feature/product/operator as objects; module granularity too fine for L1 |
| **ACP five-pillar structure** (`knowledge/acp-overview.md`) | Ensures the framework can map back to ACP for competitive comparison | ACP-centric; doesn't cover areas where ACP has no equivalent |

### How the L1 Domains Were Derived

Starting point was Zhou Chao's proposed 8 L1s, which synthesized What's New stable headings, What's Next strategic direction, and the v11 module structure. We adjusted based on three criteria:

1. **Each domain should map to a distinct buying decision or team responsibility.** This led to splitting Zhou Chao's "Platform Operations" (security + observability + cost) into separate Security and Observability domains — these are distinct products at Red Hat (ACS vs. monitoring stack) and distinct teams at customers.

2. **Each domain should contain capabilities with natural affinity.** This led to splitting "Infrastructure & Lifecycle" into Infrastructure (what you run on) and Lifecycle & Operations (how you manage the platform itself) — different personas, different evaluation criteria.

3. **Each domain must be stable across OCP versions.** Validated by checking that the domain structure holds across OCP 4.11-4.21 documentation and release notes without requiring restructuring.

Key decisions:
- **"Platform Foundation" was dissolved.** Zhou Chao bundled UI, IAM, OLM, networking, storage, and registry into one L1. These span multiple buying decisions, so they were distributed to their natural domains.
- **Console/CLI/API are not a domain.** They are presentation layers that surface capabilities from every domain. The console doesn't create capabilities, it exposes them.
- **Edge/Telco is cross-cutting, not a domain.** Edge is a deployment topology that touches infrastructure (SNO, MicroShift), lifecycle (remote provisioning), networking (bandwidth-constrained), and management (fleet at scale). Making it a domain would force duplicating features.
- **AI is one domain covering both directions.** "AI as workload" (OpenShift AI) and "AI as operator" (Lightspeed) share infrastructure and are converging strategically.

### How L2 Capabilities Were Scoped

A capability (L2) must satisfy all three criteria:
1. **Independently scoreable** — you can meaningfully rate "how well does platform X deliver this?" on a 1-4 scale
2. **Bounded** — it has a clear scope boundary that doesn't heavily overlap with other L2s in the same domain
3. **Substantial** — it contains enough features (L3s) to warrant separate evaluation, but not so many that it becomes an L1

When a potential L2 failed criterion 3 (too small), it was folded into an adjacent L2's scope description. When it failed criterion 2 (too overlapping), the boundaries were redrawn.

### Completeness Check Procedure

The taxonomy was validated using a systematic cross-reference: every item in every source must map to an L2 with no orphans. This same procedure should be repeated when updating the taxonomy.

**Step 1: Release notes check.** Take every feature category heading from the latest OCP release notes "New features and enhancements" section. Each must map to an L2. If a new category appears that doesn't fit, evaluate whether an existing L2's scope should expand or a new L2 is needed.

**Step 2: Documentation TOC check.** Take every top-level category from the OCP docs landing page (docs.redhat.com). Each must map to a domain. If a new top-level category appears (as "AI" and "Virtualization" did in 4.18), evaluate whether a new domain is warranted.

**Step 3: Product portfolio check.** Review the Red Hat all-products page and the OpenShift Platform Plus datasheet. Every OpenShift-related product must map to at least one L2. Pay special attention to newly announced products (e.g., Connectivity Link, Edge Manager).

**Step 4: Roadmap check.** Review the latest What's Next deck or OCP 5 roadmap. Any strategic pillar that doesn't map to a domain signals a potential structural gap.

**Step 5: Cross-cutting dimension check.** For each cross-cutting dimension, verify it's still cross-cutting (touches 3+ domains) and hasn't grown large enough to warrant its own domain.

### How to Update This Taxonomy

When a new OCP version ships:

1. Run the completeness check procedure above against the new version's release notes and documentation
2. For **new features in existing areas**: expand the relevant L2 scope description; no structural change needed
3. For **new feature categories in release notes**: evaluate whether to expand an existing L2 or add a new one (apply the three L2 criteria)
4. For **new top-level documentation categories**: evaluate whether to add a new domain (apply the L1 derivation criteria)
5. For **deprecated/removed capabilities**: do not remove from the taxonomy (the framework should be a superset for historical comparison); add a note if needed
6. Update the "Based on" version line and source dates

### Design Principles

- **Domains are capability areas, not products.** Red Hat's product boundaries are unstable (suite vs. product vs. operator vs. feature). Capabilities are stable.
- **Each L2 should be independently scoreable.** "How well does platform X deliver this capability?" on a 1-4 scale.
- **Vendor-neutral naming.** The same framework should work for scoring both OCP and ACP without bias.
- **Console/CLI/API are presentation layers, not capabilities.** They surface capabilities from every domain but don't create them.
- **Cross-cutting dimensions are orthogonal.** They are deployment contexts or compliance constraints, not capability areas. Each L2 can be evaluated against each dimension.

### Totals

- 9 domains
- 53 capabilities
- 6 cross-cutting dimensions
