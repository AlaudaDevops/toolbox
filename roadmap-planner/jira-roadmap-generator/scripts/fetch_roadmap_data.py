import json
import subprocess
import urllib.request
import sys
import argparse

def get_json(url, headers):
    req = urllib.request.Request(url, headers=headers)
    with urllib.request.urlopen(req) as response:
        return json.loads(response.read().decode())

def get_jira_desc(key):
    try:
        res = subprocess.run(['jira', 'issue', 'view', key, '--raw'], capture_output=True, text=True, check=True)
        data = json.loads(res.stdout)
        return data.get('fields', {}).get('description', '')
    except Exception as e:
        print(f"Failed to fetch description for {key}: {e}", file=sys.stderr)
        return ""

def main():
    parser = argparse.ArgumentParser(description="Fetch Roadmap Planner Data from Jira and output as JSON.")
    parser.add_argument("--pillar", help="Pillar Name (e.g. 'Tool Integration'). Omit for all pillars.")
    parser.add_argument("--pillar-id", help="Pillar ID (e.g. 292861). Omit for all pillars.")
    parser.add_argument("--quarter", required=True, help="Quarter (e.g. '2026Q2')")
    parser.add_argument("--jira-user", required=True, help="Jira Username")
    parser.add_argument("--jira-pass", required=True, help="Jira Password/Token")
    parser.add_argument("--jira-url", default="https://jira.alauda.cn", help="Jira Base URL")
    parser.add_argument("--jira-project", default="DEVOPS", help="Jira Project Key")
    parser.add_argument("--api-url", default="https://devops-road.alaudatech.net", help="Roadmap Planner API URL")
    parser.add_argument("--output", default="roadmap_raw_data.json", help="Output file path for the JSON data")

    args = parser.parse_args()

    headers = {
        "X-Jira-Username": args.jira_user,
        "X-Jira-Password": args.jira_pass,
        "X-Jira-BaseURL": args.jira_url,
        "X-Jira-Project": args.jira_project
    }

    pillars = []
    
    if args.pillar_id:
        pillars.append({
            "id": args.pillar_id,
            "name": args.pillar or f"Pillar {args.pillar_id}"
        })
    else:
        print("Fetching all pillars...", file=sys.stderr)
        basic_data = get_json(f"{args.api_url}/api/basic", headers)
        for p in basic_data.get("pillars", []):
            pillars.append({
                "id": p["id"],
                "name": p["name"]
            })

    pillars_data = []

    for p in pillars:
        print(f"Fetching milestones for Pillar: {p['name']} ({p['id']}) in {args.quarter}...", file=sys.stderr)
        try:
            milestones_url = f"{args.api_url}/api/milestones?quarter={args.quarter}&pillar_id={p['id']}"
            milestones_data = get_json(milestones_url, headers)
        except Exception as e:
            print(f"Error fetching milestones for {p['name']}: {e}", file=sys.stderr)
            continue

        milestones_list = []
        for ms in milestones_data.get("milestones", []):
            ms_id = ms["id"]
            ms_name = ms["name"]
            
            print(f"  Fetching epics for milestone {ms_name}...", file=sys.stderr)
            epics_url = f"{args.api_url}/api/epics?milestone_id={ms_id}"
            epics_data = get_json(epics_url, headers)
            
            epics_list = []
            for ep in epics_data.get("epics", []):
                key = ep["key"]
                name = ep["name"]
                comps = ep.get("components") or []
                desc = get_jira_desc(key)
                
                epics_list.append({
                    "key": key,
                    "name": name,
                    "components": comps,
                    "description": desc
                })
                
            milestones_list.append({
                "milestone_name": ms_name,
                "epics": epics_list
            })
            
        if milestones_list:
            pillars_data.append({
                "name": p["name"],
                "milestones": milestones_list
            })

    output_data = {
        "quarter": args.quarter,
        "jira_url": args.jira_url,
        "title": f"{args.pillar} Roadmap" if args.pillar else "Global DevOps Roadmap",
        "pillars": pillars_data
    }

    with open(args.output, "w") as f:
        json.dump(output_data, f, indent=2, ensure_ascii=False)

    print(f"Success! Raw JSON data saved to {args.output}", file=sys.stderr)

if __name__ == "__main__":
    main()