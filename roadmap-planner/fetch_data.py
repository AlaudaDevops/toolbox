import json
import subprocess
import urllib.request
import sys

# Jira creds
headers = {
    "X-Jira-Username": "daniel",
    "X-Jira-Password": "KG4gY8lcc4vM",
    "X-Jira-BaseURL": "https://jira.alauda.cn",
    "X-Jira-Project": "DEVOPS"
}

def get_json(url):
    req = urllib.request.Request(url, headers=headers)
    with urllib.request.urlopen(req) as response:
        return json.loads(response.read().decode())

def get_jira_desc(key):
    try:
        res = subprocess.run(['jira', 'issue', 'view', key, '--raw'], capture_output=True, text=True, check=True)
        data = json.loads(res.stdout)
        return data.get('fields', {}).get('description', '')
    except:
        return ""

print("Fetching milestones...")
milestones_data = get_json("https://devops-road.alaudatech.net/api/milestones?quarter=2026Q2&pillar_id=292861")

result = []
for ms in milestones_data.get("milestones", []):
    ms_id = ms["id"]
    ms_name = ms["name"]
    
    print(f"Fetching epics for {ms_name}...", file=sys.stderr)
    epics_data = get_json(f"https://devops-road.alaudatech.net/api/epics?milestone_id={ms_id}")
    
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
        
    result.append({
        "milestone_name": ms_name,
        "epics": epics_list
    })

with open("roadmap_data.json", "w") as f:
    json.dump(result, f, indent=2)

print("Data saved to roadmap_data.json")
