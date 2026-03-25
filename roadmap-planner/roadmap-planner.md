# Roadmap Planner & Adapted DORA Metrics

## 1. Executive Summary

The **Roadmap Planner** is a comprehensive full-stack application designed to bridge the gap between high-level product strategy and day-to-day execution. By integrating directly with Jira, it provides a visual, real-time Kanban interface for managing product roadmaps through **Pillars** and **Quarters**. Beyond roadmap visualization, it features a sophisticated **Metrics Engine** that calculates adapted DORA (DevOps Research and Assessment) metrics, providing teams with actionable insights into their release frequency, delivery speed, and quality without requiring access to production environments.

## 2. Key Features & Advantages

* **Direct Jira Integration (Zero Local Storage)**: Operates directly on Jira data via the REST API, ensuring a "single source of truth." No local database is required, simplifying data privacy and security management.
* **Visual Roadmap (Kanban Board)**: Provides a 2D visualization of product strategy.
  * **Vertical Lanes (Pillars)**: Represent product areas or team focuses.
  * **Horizontal Lanes (Quarters)**: Represent the time dimension for delivery.
* **Adapted DORA Metrics**: A specialized metrics engine that adapts traditional DORA metrics (Deployment Frequency, Lead Time for Changes, MTTR, Change Failure Rate) for teams delivering software to external customers.
* **Unified Deployment Model**: A single Docker container serves both the Go API and the React frontend, eliminating CORS issues and simplifying infrastructure requirements.
* **Advanced UI/UX**: Features drag-and-drop support for moving items between milestones and real-time filtering by component versions.

## 3. Technical Architecture

* **Backend**: Built with **Go** (Gin framework) for high performance and type safety.
  * **Interface-based Design**: The metrics engine and Jira client use Go interfaces, allowing for easy extensibility and mocking for tests.
  * **Background Collector**: A persistent background worker fetches and caches enriched Jira data to ensure the UI remains responsive even with complex metrics calculations.
* **Frontend**: Built with **React**, providing a modern, reactive user interface with hooks-based state management.
* **Metrics Engine**: Integrates with **Prometheus**, exposing industry-standard metrics for external monitoring and alerting.

## 4. Innovation: Adapted DORA Metrics

The project's standout innovation is its approach to measuring DevOps performance in environments where production access is restricted (e.g., shipping plugins to on-premise customers).

* **Release Frequency**: Measures how often updates are published.
* **Lead Time to Release**: Measures the time from Epic creation to version release.
* **Cycle Time**: Measures active development time (In Progress to Done).
* **Patch Ratio**: Measures the quality of releases (Patches vs. Features).
* **Time to Patch**: Measures responsiveness to bugs and security vulnerabilities.

## 5. Challenges & Solutions

* **Challenge: Jira Data Consistency**: Jira's flexible nature can lead to inconsistent data.
  * **Solution**: Implemented a "Requirement Validation" layer in the documentation and code that identifies missing data points (e.g., missing release dates) and guides users on how to fix them in Jira.
* **Challenge: Real-time Metrics on Large Datasets**: Calculating metrics from thousands of Jira issues can be slow.
  * **Solution**: Developed a background metrics collector that performs periodic enrichment and caching, exposing data through a fast REST API and Prometheus exporter.
* **Challenge: Complex Roadmap Mapping**: Mapping Jira's issue hierarchy to a roadmap model.
  * **Solution**: Used a structured approach (Pillars as parent issues, Milestones as sub-tasks, Epics via "Blocks" links) to maintain a clear visual hierarchy.

## 6. Project Completeness & Quality Assurance

The project is "production-ready," exceeding typical prototype standards:

* **Testing Strategy**: Comprehensive unit tests for both Go and React, plus REST-based integration tests. Minimum coverage is enforced via CI.
* **Security Built-in**: Includes **Trivy** scanning for container vulnerabilities and **golangci-lint** for code quality. Credentials are never stored locally.
* **DevOps Ready**: Complete with Docker Compose for local development and a unified Dockerfile for cloud deployment.
* **Documentation**: Includes full API specs, architectural diagrams, metrics definitions, and developer guides.

## 7. Conclusion

The Roadmap Planner is not just a visualization tool; it is a performance-driving platform. By combining roadmap management with automated, adapted DORA metrics, it empowers Product Managers and Engineering Leaders to make data-driven decisions. Its clean architecture and high level of completeness make it an ideal candidate for professional software engineering competitions.
