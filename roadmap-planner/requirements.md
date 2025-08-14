# Requirements

Jira structure: [jira-structure.md](./jira-structure.md)

## User stories

1. Kanban:

As a Product Manager, I want to be able to easily see our product roadmap to understand our current plan as a Kanban diveded by pillar (different vertical lanes), and quarter (different horizontal lanes)

2. Roadmap planning

As a Product Manager, I want to quickly and easily create new Milestones in the Kanban. Each milestone is a sub-task of a Pillar and has a quarter field that needs to be filled. This will make the milestone appear in the correct quarter and pillar in the Kanban

3. Epic creation

As a Product Manager, I want to be able to create new Epics and link them to the correct milestone in the Kanban. Epics are linked to a milestone using the `blocks` issue link type, which will make it appear in the correct milestone in the Kanban

4. Update roadmap

As a Product Manager, I want to be able to quickly and easily move epics to different milestones to reflect changes in our roadmap

5. Update epic version

As a Product Manager, I want to be able to update the version of an epic to reflect changes in our roadmap, typically an epic reflects a change in one component/plugin, and the Pillar should have the related plugin name, i.e Tool Integration pillar is directly related to `connectors-operator` plugin. The user should only be able to pick and filter through related versions of the component (using the name as prefix)



