# AI Agent Development Prompt for Roadmap Planner

## Context
You are tasked with developing a roadmap planning application that integrates with Jira to provide a Kanban-style interface for managing product roadmaps. The application should help Product Managers visualize and manage their roadmap using the existing Jira structure.

## Project Structure
Create the following directory structure:
```
roadmap-planner/
├── backend/
│   ├── cmd/
│   │   └── server/
│   │       └── main.go
│   ├── internal/
│   │   ├── api/
│   │   │   ├── handlers/
│   │   │   ├── middleware/
│   │   │   └── routes.go
│   │   ├── jira/
│   │   │   ├── client.go
│   │   │   └── models.go
│   │   ├── models/
│   │   └── config/
│   ├── go.mod
│   └── go.sum
├── frontend/
│   ├── src/
│   │   ├── components/
│   │   │   ├── KanbanBoard.jsx
│   │   │   ├── MilestoneCard.jsx
│   │   │   ├── EpicCard.jsx
│   │   │   └── modals/
│   │   ├── services/
│   │   │   └── api.js
│   │   ├── hooks/
│   │   └── App.jsx
│   ├── package.json
│   └── public/
└── docker-compose.yml
```

## Development Tasks

### 1. Backend Development (Go)

#### Task 1.1: Jira Integration
- Implement Jira client using `github.com/andygrunwald/go-jira` (already used in the codebase)
- Create authentication mechanism (basic auth or OAuth)
- Implement methods to:
  - Fetch pillars (parent issues in DEVOPS project)
  - Fetch milestones (sub-tasks of pillars)
  - Fetch epics (issues linked to milestones via "blocks")
  - Create milestones as sub-tasks
  - Create epics and link them to milestones
  - Update issue relationships

#### Task 1.2: REST API
- Use Gin framework for HTTP server
- Implement endpoints as specified in SPECIFICATION.md
- Add proper error handling and logging
- Implement request validation
- Add CORS middleware for frontend integration

#### Task 1.3: Data Models
- Define Go structs for Pillar, Milestone, Epic
- Implement JSON serialization
- Add validation tags
- Create conversion functions between Jira issues and internal models

### 2. Frontend Development (React)

#### Task 2.1: Kanban Board Component
- Create responsive grid layout with pillars as rows and quarters as columns
- Implement drag-and-drop using `react-beautiful-dnd` or similar
- Add visual feedback during drag operations
- Handle drop events and API calls

#### Task 2.2: Modal Components
- Create milestone creation modal with form validation
- Create epic creation modal with component/version selection
- Implement autocomplete for component versions
- Add loading states and error handling

#### Task 2.3: State Management
- Use React Context or Redux for global state
- Implement optimistic updates for better UX
- Add real-time synchronization with backend
- Handle concurrent edit conflicts

### 3. Integration & Testing

#### Task 3.1: API Integration
- Implement frontend service layer for API calls
- Add proper error handling and retry logic
- Implement caching strategy
- Add loading indicators

#### Task 3.2: Testing
- Write unit tests for Jira client methods
- Add integration tests for API endpoints
- Create frontend component tests
- Implement end-to-end tests for critical workflows

## Key Implementation Guidelines

### Jira Integration Best Practices
1. **Issue Relationships**: Use the existing structure where milestones are sub-tasks of pillars, and epics are linked via "blocks" relationship
2. **Custom Fields**: Utilize the quarter field for milestone positioning
3. **Error Handling**: Implement robust error handling for Jira API failures
4. **Rate Limiting**: Respect Jira API rate limits with proper throttling

### UI/UX Requirements
1. **Responsive Design**: Ensure the Kanban board works on different screen sizes
2. **Drag & Drop**: Smooth drag-and-drop experience with visual feedback
3. **Real-time Updates**: Show changes from other users in real-time
4. **Loading States**: Provide clear feedback during API operations

### Performance Considerations
1. **Caching**: Implement intelligent caching to reduce Jira API calls
2. **Pagination**: Handle large datasets efficiently
3. **Optimistic Updates**: Update UI immediately, sync with backend asynchronously
4. **Debouncing**: Debounce search and filter operations

## Configuration Requirements

### Backend Configuration
```yaml
jira:
  baseURL: "https://your-jira-instance.atlassian.net"
  username: "your-username"
  password: "your-api-token"
  project: "PROJECT-KEY"

server:
  port: 8080
  cors:
    allowedOrigins: ["http://localhost:3000"]

cache:
  ttl: "5m"
  refreshInterval: "1m"
```

### Environment Variables
- `JIRA_BASE_URL`
- `JIRA_USERNAME`
- `JIRA_PASSWORD`
- `SERVER_PORT`
- `DATABASE_URL` (if using database for caching)

## Success Criteria
1. Product Managers can view the complete roadmap in a Kanban format
2. Users can create milestones and assign them to quarters and pillars
3. Users can create epics and link them to milestones
4. Drag-and-drop functionality works smoothly for moving epics
5. All changes are properly synchronized with Jira
6. The application handles errors gracefully
7. The interface is responsive and user-friendly

## Additional Notes
- Follow the existing codebase patterns for Jira integration (see `artifact-scanner` and `plugin-releaser` examples)
- Use the same Go modules and dependencies where possible
- Implement proper logging using uber zap logger
- Consider using the existing Jira client patterns from other tools in the repository