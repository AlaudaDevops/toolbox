# Roadmap Planner Application Specification

## Overview
A web-based Kanban application for managing product roadmaps using direct Jira integration. The application displays roadmap items organized by Pillars (vertical lanes) and Quarters (horizontal lanes), enabling Product Managers to visualize and manage their product roadmap efficiently.

## Architecture

### Technology Stack
- **Backend**: Go with Gin framework
- **Frontend**: React with drag-and-drop support
- **Integration**: Jira REST API v8.4 (direct integration, no local storage)
- **Authentication**: Jira Basic Auth (user credentials)

### Core Components

#### 1. Jira Integration Layer
```go
type JiraClient interface {
    GetPillars(project string) ([]Pillar, error)
    GetMilestones(pillarID string) ([]Milestone, error)
    GetEpics(milestoneID string) ([]Epic, error)
    CreateMilestone(pillar Pillar, quarter string, data MilestoneData) (*Milestone, error)
    CreateEpic(milestone Milestone, data EpicData) (*Epic, error)
    UpdateEpicMilestone(epicID, newMilestoneID string) error
    LinkEpicToMilestone(epicID, milestoneID string) error
    GetComponentVersions(project, component string) ([]string, error)
}
```

#### 2. Data Models
```go
type Pillar struct {
    ID          string `json:"id"`
    Key         string `json:"key"`
    Name        string `json:"name"`
    Priority    string `json:"priority"`
    Component   string `json:"component"`
    Milestones  []Milestone `json:"milestones"`
}

type Milestone struct {
    ID          string `json:"id"`
    Key         string `json:"key"`
    Name        string `json:"name"`
    Quarter     string `json:"quarter"` // 2025Q1, 2025Q2, etc.
    PillarID    string `json:"pillar_id"`
    Epics       []Epic `json:"epics"`
    Status      string `json:"status"`
}

type Epic struct {
    ID          string `json:"id"`
    Key         string `json:"key"`
    Name        string `json:"name"`
    Version     string `json:"version"`
    Component   string `json:"component"`
    MilestoneID string `json:"milestone_id"`
    Status      string `json:"status"`
    Priority    string `json:"priority"`
}
```

## Authentication Flow
1. User enters Jira credentials (username/password or API token)
2. Backend validates credentials against Jira server
3. Credentials stored in session for subsequent API calls
4. All Jira operations use user's credentials

## API Endpoints

### Backend REST API
```
POST   /api/auth/login                 # Authenticate with Jira
POST   /api/auth/logout                # Clear session
GET    /api/roadmap                    # Get complete roadmap data
GET    /api/pillars                    # Get all pillars
POST   /api/milestones                 # Create new milestone
PUT    /api/epics/:id/milestone        # Move epic to different milestone
POST   /api/epics                      # Create new epic
GET    /api/components/:name/versions  # Get versions for component
```

## Business Logic

### 1. Milestone Management
- Milestones are Jira sub-tasks of Pillars
- Quarter custom field determines horizontal position
- Validate quarter format (YYYYQX)

### 2. Epic Management
- Epics link to milestones via "blocks" relationship
- Component field enables version filtering
- Version updates validate against component releases

### 3. Data Synchronization
- All data fetched directly from Jira on each request
- No local caching or storage
- Real-time data consistency

## Implementation Requirements

### Phase 1: Core Functionality
1. Jira authentication and session management
2. Basic Kanban board display
3. Milestone creation and management
4. Epic creation and linking

### Phase 2: Advanced Features
1. Drag & drop epic movement
2. Component version filtering
3. Bulk operations

## Technical Constraints

### Jira Integration
- Use Jira REST API v8.4
- Maintain existing issue types and relationships
- Handle API rate limits gracefully
- Direct integration only (no local storage)

### Performance
- Optimize API calls to minimize Jira requests
- Implement request batching where possible
- Handle large datasets efficiently

### Security
- Secure credential handling in sessions
- Input validation and sanitization
- CORS configuration for frontend

## Error Handling
- Graceful degradation when Jira is unavailable
- User-friendly error messages
- Retry mechanisms for transient failures
- Audit logging for all operations
