---
name: jira-roadmap-generator
description: Generate a detailed markdown roadmap based on Jira and Roadmap Planner API data. Use when you need to fetch epics and milestones for a specific pillar or all pillars for a given quarter and format it into a standardized roadmap capability document.
---

# Jira Roadmap Generator

This skill automates the process of fetching roadmap data (pillars, milestones, epics) from the Roadmap Planner API and extracting detailed Epic descriptions using the Jira CLI. 

Crucially, **YOU (the AI Agent) are responsible for interpreting and translating the raw Chinese data into concise, readable English capability descriptions** before formatting the final Markdown document.

## Prerequisites

- You must have the `jira` CLI installed and authenticated on your machine.
- You must have a `.env` file or provide the required credentials for the Roadmap Planner API and Jira (Username, Password/Token, Base URL, Project).

A template is provided in `assets/.env.template` which looks like this:
```env
JIRA_USERNAME=your_username
JIRA_PASSWORD=your_password
JIRA_BASE_URL=https://jira.example.com
JIRA_PROJECT=DEVOPS
API_URL=https://your-roadmap-api.example.com
```

## Usage

### Step 1: Fetch Raw Data as JSON

Use the bundled python script to fetch the raw data from Jira and the Roadmap API. The script will output a JSON file containing all the raw, un-translated data. If the user has a `.env` file, you can source the environment variables from it to fill in the script arguments.

**For a Specific Pillar:**

```bash
python3 scripts/fetch_roadmap_data.py \
    --pillar "Tool Integration" \
    --pillar-id "292861" \
    --quarter "2026Q2" \
    --jira-user "$JIRA_USERNAME" \
    --jira-pass "$JIRA_PASSWORD" \
    --jira-url "$JIRA_BASE_URL" \
    --jira-project "$JIRA_PROJECT" \
    --api-url "$API_URL" \
    --output "roadmap_raw_data.json"
```

**For the Global Roadmap (All Pillars):**

Omit `--pillar` and `--pillar-id`:

```bash
python3 scripts/fetch_roadmap_data.py \
    --quarter "2026Q2" \
    --jira-user "$JIRA_USERNAME" \
    --jira-pass "$JIRA_PASSWORD" \
    --jira-url "$JIRA_BASE_URL" \
    --jira-project "$JIRA_PROJECT" \
    --api-url "$API_URL" \
    --output "roadmap_raw_data.json"
```

### Step 2: Interpret, Translate, and Generate Markdown

Once the JSON file is generated, you (the AI agent) must read it and generate the final markdown file.

**Formatting Rules for the AI:**
1. **Title:** Use the `title` and `quarter` from the JSON to create an H1 title.
2. **Domain Table:** Create a "Domains" table summarizing the total Epics per Pillar (Domain). Translate Pillar names to English.
3. **Pillar Sections (H2):** For each pillar, create an H2 section. Translate Pillar names to English.
4. **Milestone Sections (H3):** For each milestone, create an H3 section. **Remove the quarter prefix** from the milestone name (e.g., change "2026Q2：Connectors security enhancement" to "Connectors security enhancement") and translate it to English.
5. **Capability Tables:** Under each Milestone, create a table with columns: `| # | Capability | Scope | Components |`.
6. **Capability (Epic Name):** Add the Epic key as a markdown link in front of the translated epic name: `[DEVOPS-XXXX](jira_url/browse/DEVOPS-XXXX) English Epic Name`.
7. **Scope (Epic Description):** Read the raw Chinese description. **Use your intelligence to interpret, translate, and summarize the description** into a concise, readable English paragraph optimized for the table format. Do not just blindly translate; ensure it makes sense as a capability description.
8. **Components:** List the components separated by commas, or "N/A" if none.

Write the final formatted text to the desired output markdown file (e.g., `roadmap/devops-roadmap-2026q2.md`), and clean up the temporary JSON file.
