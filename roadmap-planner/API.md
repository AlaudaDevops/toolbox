# API Documentation

This document describes the REST API endpoints for the Roadmap Planner application.

## Recent Changes

**Version 1.2.0 Updates:**
- **API Cleanup**: Removed unused endpoints `/api/roadmap`, `/api/pillars`, `/api/pillars/:id/milestones`, and `/api/milestones/:id/epics`
- **Streamlined Architecture**: Frontend now uses optimized filtering APIs for better performance
- **Projects API**: Kept `/api/projects` endpoint for future use
- **Assignee Requirements**: Assignee is still required for milestone and epic creation
- **Enhanced Filtering**: Pillars with resolved/cancelled status are automatically filtered out

## Base URL

```
http://localhost:8080
```

## Authentication

The API uses Jira credentials passed via HTTP headers for authentication:

```
X-Jira-Username: your-username
X-Jira-Password: your-api-token
X-Jira-BaseURL: https://your-jira-instance.atlassian.net
X-Jira-Project: PROJECT-KEY
```

## Endpoints

### Health Check

#### GET /health

Returns the health status of the API.

**Response:**
```json
{
  "status": "healthy",
  "service": "roadmap-planner"
}
```

### Authentication

#### POST /api/auth/login

Authenticate with Jira credentials.

**Request Body:**
```json
{
  "username": "your-username",
  "password": "your-api-token",
  "base_url": "https://your-jira-instance.atlassian.net"
}
```

**Response:**
```json
{
  "message": "Authentication successful",
  "user": "your-username"
}
```

#### POST /api/auth/logout

Clear authentication session.

**Response:**
```json
{
  "message": "Logout successful"
}
```

#### GET /api/auth/status

Check authentication status.

**Response:**
```json
{
  "authenticated": true,
  "user": "your-username"
}
```

### Roadmap Data

#### GET /api/basic

Get basic roadmap data including pillars, quarters, components, and versions.

**Response:**
```json
{
  "pillars": [
    {
      "id": "123",
      "key": "DEVOPS-123",
      "name": "Tool Integration",
      "priority": "High",
      "component": "connectors-operator"
    }
  ],
  "quarters": ["2025Q1", "2025Q2", "2025Q3", "2025Q4", "2026Q1", "2026Q2", "2026Q3", "2026Q4"],
  "project": {
    "id": "10000",
    "name": "DevOps Project",
    "key": "DEVOPS",
    "components": [
      {
        "name": "connectors-operator",
        "id": "10001",
        "description": "Connectors Operator Component"
      }
    ],
    "versions": [
      {
        "name": "connectors-operator-1.0.0",
        "id": "10002",
        "archived": false,
        "released": true,
        "releaseDate": "2025-01-01",
        "userReleaseDate": "2025-01-01"
      }
    ]
  }
}
```

#### GET /api/milestones

Get milestones with optional filtering.

**Query Parameters:**
- `pillar_id` (optional): Filter by pillar IDs (can be repeated)
- `quarter` (optional): Filter by quarters (can be repeated)

**Examples:**
- `/api/milestones?pillar_id=123&pillar_id=456` - Get milestones for specific pillars
- `/api/milestones?quarter=2025Q1&quarter=2025Q2` - Get milestones for specific quarters
- `/api/milestones?pillar_id=123&quarter=2025Q1` - Combined filtering

**Response:**
```json
{
  "milestones": [
    {
      "id": "456",
      "key": "DEVOPS-456",
      "name": "Q1 Integration Milestone",
      "quarter": "2025Q1",
      "pillar_id": "123",
      "status": "In Progress"
    }
  ]
}
```

#### GET /api/epics

Get epics with optional filtering.

**Query Parameters:**
- `milestone_id` (optional): Filter by milestone IDs (can be repeated)
- `pillar_id` (optional): Filter by pillar IDs (can be repeated)
- `component` (optional): Filter by components (can be repeated)
- `version` (optional): Filter by versions (can be repeated)

**Examples:**
- `/api/epics?milestone_id=456` - Get epics for a specific milestone
- `/api/epics?component=connectors-operator&version=1.2.0` - Filter by component and version
- `/api/epics?pillar_id=123&component=connectors-operator` - Combined filtering

**Response:**
```json
{
  "epics": [
    {
      "id": "789",
      "key": "DEVOPS-789",
      "name": "Implement OAuth Integration",
      "version": "connectors-operator-1.2.0",
      "component": "connectors-operator",
      "milestone_id": "456",
      "status": "To Do",
      "priority": "High"
    }
  ]
}
```

### Projects

#### GET /api/projects

Get list of available projects (kept for future use).

**Response:**
```json
{
  "projects": [
    {
      "key": "DEVOPS",
      "name": "DevOps Project",
      "id": "10000"
    }
  ]
}
```

### Milestones

#### POST /api/milestones

Create a new milestone.

**Request Body:**
```json
{
  "name": "Q2 Integration Milestone",
  "quarter": "2025Q2",
  "pillar_id": "123",
  "assignee": {
    "account_id": "5d5f0f8e2c0e4f0001a1b2c3",
    "name": "john.doe",
    "display_name": "John Doe",
    "email_address": "john.doe@company.com"
  }
}
```

**Response:**
```json
{
  "id": "457",
  "key": "DEVOPS-457",
  "name": "Q2 Integration Milestone",
  "quarter": "2025Q2",
  "pillar_id": "123",
  "status": "To Do",
  "epics": []
}
```

### Epics

#### POST /api/epics

Create a new epic.

**Request Body:**
```json
{
  "name": "Implement SAML Integration",
  "component": "connectors-operator",
  "version": "connectors-operator-1.3.0",
  "priority": "High",
  "milestone_id": "456",
  "assignee": {
    "account_id": "5d5f0f8e2c0e4f0001a1b2c3",
    "name": "john.doe",
    "display_name": "John Doe",
    "email_address": "john.doe@company.com"
  }
}
```

**Response:**
```json
{
  "id": "790",
  "key": "DEVOPS-790",
  "name": "Implement SAML Integration",
  "version": "connectors-operator-1.3.0",
  "component": "connectors-operator",
  "milestone_id": "456",
  "status": "To Do",
  "priority": "High"
}
```

#### PUT /api/epics/:id/milestone

Move an epic to a different milestone.

**URL Parameters:**
- `id`: Epic ID

**Request Body:**
```json
{
  "milestone_id": "457"
}
```

**Response:**
```json
{
  "message": "Epic milestone updated successfully"
}
```

### Components

#### GET /api/components/:name/versions

Get available versions for a component.

**URL Parameters:**
- `name`: Component name (e.g., "connectors-operator")

**Response:**
```json
{
  "versions": [
    "connectors-operator-1.0.0",
    "connectors-operator-1.1.0",
    "connectors-operator-1.2.0",
    "connectors-operator-1.3.0"
  ]
}
```

### Users

#### GET /api/users/assignable

Get users that can be assigned to issues in the project.

**Query Parameters:**
- `issueKey` (required): The Jira issue key to get assignable users for
- `query` (optional): Search query to filter users by name, display name, or email address

**Examples:**
- `/api/users/assignable?issueKey=DEVOPS-123` - Get all assignable users for issue DEVOPS-123
- `/api/users/assignable?issueKey=DEVOPS-123&query=john` - Search for users containing "john" in name, display name, or email
- `/api/users/assignable?issueKey=DEVOPS-123&query=john.doe@company.com` - Search for users by email address

**Features:**
- Real-time search with debouncing (300ms delay)
- Searches across user's name, display name, and email address
- Returns users that can be assigned to the specific issue context

**Response:**
```json
{
  "users": [
    {
      "account_id": "5d5f0f8e2c0e4f0001a1b2c3",
      "name": "john.doe",
      "display_name": "John Doe",
      "email_address": "john.doe@company.com"
    },
    {
      "account_id": "5d5f0f8e2c0e4f0001a1b2c4",
      "name": "jane.smith",
      "display_name": "Jane Smith",
      "email_address": "jane.smith@company.com"
    }
  ]
}
```

## Error Responses

All endpoints may return error responses in the following format:

```json
{
  "error": "Error message describing what went wrong"
}
```

### Common HTTP Status Codes

- `200 OK`: Request successful
- `201 Created`: Resource created successfully
- `400 Bad Request`: Invalid request data
- `401 Unauthorized`: Authentication required or invalid
- `404 Not Found`: Resource not found
- `500 Internal Server Error`: Server error

## Data Models

### Pillar

```json
{
  "id": "string",
  "key": "string",
  "name": "string",
  "priority": "string",
  "component": "string",
  "milestones": "Milestone[]"
}
```

### Milestone

```json
{
  "id": "string",
  "key": "string",
  "name": "string",
  "quarter": "string",
  "pillar_id": "string",
  "status": "string",
  "epics": "Epic[]"
}
```

### Epic

```json
{
  "id": "string",
  "key": "string",
  "name": "string",
  "version": "string",
  "component": "string",
  "milestone_id": "string",
  "status": "string",
  "priority": "string"
}
```

### User

```json
{
  "account_id": "string",
  "name": "string",
  "display_name": "string",
  "email_address": "string"
}
```

### Create Milestone Request

```json
{
  "name": "string",
  "quarter": "string",
  "pillar_id": "string",
  "assignee_id": "string"
}
```

### Create Epic Request

```json
{
  "name": "string",
  "component": "string",
  "version": "string",
  "priority": "string",
  "milestone_id": "string",
  "assignee_id": "string"
}
```

## Important Notes

### Assignee Requirements

**All milestone and epic creation requests now require an assignee:**
- Use `GET /api/users/assignable?issueKey=<issue_key>` to fetch available users
- For milestones: use the parent pillar's issue key as the `issueKey` parameter
- For epics: use the parent milestone's issue key as the `issueKey` parameter
- Include `assignee` field (full user object) in all `POST /api/milestones` and `POST /api/epics` requests
- The `assignee` should be the complete user object from the users endpoint

**Enhanced Search Functionality:**
- The frontend provides a searchable dropdown with real-time filtering
- Users can search by name, display name, or email address
- Search queries are debounced (300ms) to optimize API performance
- The API filters results server-side for better performance with large user bases

### Pillar Filtering

**Only active pillars are returned:**
- Pillars with status "Resolved", "Cancelled", "Closed", "Done", "Won't Do", or "Won't Fix" are automatically filtered out
- This ensures the roadmap only shows active work

### Quarter Format

**Quarter values must follow the format YYYYQX:**
- Valid examples: "2025Q1", "2025Q2", "2026Q3"
- Invalid examples: "2025-Q1", "Q1-2025", "2025 Q1"

## Rate Limiting

The API respects Jira's rate limiting. If you encounter rate limit errors:

- Reduce the frequency of requests
- Implement exponential backoff
- Cache responses when possible

## Examples

### cURL Examples

**Authenticate:**
```bash
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "your-username",
    "password": "your-api-token",
    "base_url": "https://your-jira-instance.atlassian.net"
  }'
```

**Get basic data:**
```bash
curl -X GET http://localhost:8080/api/basic \
  -H "X-Jira-Username: your-username" \
  -H "X-Jira-Password: your-api-token" \
  -H "X-Jira-BaseURL: https://your-jira-instance.atlassian.net" \
  -H "X-Jira-Project: DEVOPS"
```

**Get milestones:**
```bash
curl -X GET "http://localhost:8080/api/milestones?pillar_id=123&quarter=2025Q1" \
  -H "X-Jira-Username: your-username" \
  -H "X-Jira-Password: your-api-token" \
  -H "X-Jira-BaseURL: https://your-jira-instance.atlassian.net" \
  -H "X-Jira-Project: DEVOPS"
```

**Get epics:**
```bash
curl -X GET "http://localhost:8080/api/epics?milestone_id=456" \
  -H "X-Jira-Username: your-username" \
  -H "X-Jira-Password: your-api-token" \
  -H "X-Jira-BaseURL: https://your-jira-instance.atlassian.net" \
  -H "X-Jira-Project: DEVOPS"
```

**Get assignable users:**
```bash
curl -X GET "http://localhost:8080/api/users/assignable?issueKey=DEVOPS-123" \
  -H "X-Jira-Username: your-username" \
  -H "X-Jira-Password: your-api-token" \
  -H "X-Jira-BaseURL: https://your-jira-instance.atlassian.net" \
  -H "X-Jira-Project: DEVOPS"
```

**Create milestone:**
```bash
curl -X POST http://localhost:8080/api/milestones \
  -H "Content-Type: application/json" \
  -H "X-Jira-Username: your-username" \
  -H "X-Jira-Password: your-api-token" \
  -H "X-Jira-BaseURL: https://your-jira-instance.atlassian.net" \
  -H "X-Jira-Project: DEVOPS" \
  -d '{
    "name": "Q2 Integration Milestone",
    "quarter": "2025Q2",
    "pillar_id": "123",
    "assignee": {
      "account_id": "5d5f0f8e2c0e4f0001a1b2c3",
      "name": "john.doe",
      "display_name": "John Doe",
      "email_address": "john.doe@company.com"
    }
  }'
```

**Create epic:**
```bash
curl -X POST http://localhost:8080/api/epics \
  -H "Content-Type: application/json" \
  -H "X-Jira-Username: your-username" \
  -H "X-Jira-Password: your-api-token" \
  -H "X-Jira-BaseURL: https://your-jira-instance.atlassian.net" \
  -H "X-Jira-Project: DEVOPS" \
  -d '{
    "name": "Implement SAML Integration",
    "component": "connectors-operator",
    "version": "connectors-operator-1.3.0",
    "priority": "High",
    "milestone_id": "456",
    "assignee": {
      "account_id": "5d5f0f8e2c0e4f0001a1b2c3",
      "name": "john.doe",
      "display_name": "John Doe",
      "email_address": "john.doe@company.com"
    }
  }'
```

### JavaScript Examples

**Using fetch API:**
```javascript
// Set up headers
const headers = {
  'Content-Type': 'application/json',
  'X-Jira-Username': 'your-username',
  'X-Jira-Password': 'your-api-token',
  'X-Jira-BaseURL': 'https://your-jira-instance.atlassian.net',
  'X-Jira-Project': 'DEVOPS'
};

// Get assignable users for a specific issue
const usersResponse = await fetch('/api/users/assignable?issueKey=DEVOPS-123', { headers });
const usersData = await usersResponse.json();
console.log('Available users:', usersData.users);

// Get basic data
const response = await fetch('/api/basic', { headers });
const basicData = await response.json();

// Get milestones for a specific pillar and quarter
const milestonesResponse = await fetch('/api/milestones?pillar_id=123&quarter=2025Q1', { headers });
const milestonesData = await milestonesResponse.json();

// Get epics for specific milestones
const epicsResponse = await fetch('/api/epics?milestone_id=456', { headers });
const epicsData = await epicsResponse.json();

// Create milestone
const milestoneData = {
  name: 'Q3 Security Milestone',
  quarter: '2025Q3',
  pillar_id: '123',
  assignee: {
    account_id: '5d5f0f8e2c0e4f0001a1b2c3',
    name: 'john.doe',
    display_name: 'John Doe',
    email_address: 'john.doe@company.com'
  }
};

const milestoneResponse = await fetch('/api/milestones', {
  method: 'POST',
  headers,
  body: JSON.stringify(milestoneData)
});

// Create epic
const epicData = {
  name: 'New Epic',
  component: 'connectors-operator',
  version: 'connectors-operator-1.5.0',
  priority: 'High',
  milestone_id: '456',
  assignee_id: '5d5f0f8e2c0e4f0001a1b2c3'
};

const createResponse = await fetch('/api/epics', {
  method: 'POST',
  headers,
  body: JSON.stringify(epicData)
});
```
