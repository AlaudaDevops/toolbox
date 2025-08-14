# Jira Structure

For each project there are a few issue types that are used to do roadmap planning:

1. Pillar
2. Milestone (Sub-task of Pillar)
3. Epics
4. Story / Bugs / Tech Debt / Job / etc.


A brief introduction of each type of issue is as follows:

## Pillar

Generally static, it represents a high-level domain that is involved in the project. Our `DEVOPS` has the following pillars:


1. DEVOPS-39715: Essentials ⭐ L0 - Critical
3. DEVOPS-37001: Tool Integration ⭐ L0 - Critical
6. DEVOPS-11210: CI/CD ⭐ L0 - Critical
7. DEVOPS-11209: Jenkins ⭐ L0 - Critical
8. DEVOPS-11208: Tool Deployment ⭐ L0 - Critical
5. DEVOPS-30074: Software Engineering Insights ⭐ L0 - Critical
2. DEVOPS-37002: Developer Productivity ⭐ L0 - Critical
6. DEVOPS-33725: DevOps - RFEs & NFR L1 - High

Each pillar has a list of milestones that are the deliverables for the pillar.


## Milestone

Sub-task type of Pillar, each Milestone is represents a goal that should be achieved in that quarter. Each quarter represents an ACP version.

> Our current quarter is 2025Q4, that directly relates to ACP 4.2 version

Each Milestone has a list of epics that are the deliverables for the milestone. The relationship is done using issue links of `blocks` type, meaning once an Epic or issue `blocks` a milestone, it should be considered as part of the milestone.


## Epics

Epics represent a product requirement or a set of stories to implement a change into the product. Epics are the main unit of work in the product and are the building blocks for the product roadmap. Epics are `epic` type of issues and are linked to a milestone using the `blocks` issue link type.


## Stories / Bugs / Tech Debt / Job / etc.

These are the smallest unit of work in the product and are the building blocks for the epics. These issues are linked to an epic using the `epic link` field.


