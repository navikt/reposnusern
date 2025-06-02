import json

def analyze_structure(obj, indent=0):
    pad = "  " * indent
    if isinstance(obj, dict):
        for key, value in obj.items():
            print(f"{pad}{key}: {type(value).__name__}")
            analyze_structure(value, indent + 1)
    elif isinstance(obj, list) and obj:
        print(f"{pad}List[{len(obj)}] → {type(obj[0]).__name__}")
        analyze_structure(obj[0], indent + 1)

with open("data/navikt_analysis_data.json", "r") as f:
    data = json.load(f)

print("Top-level structure:")
for key, value in data.items():
    print(f"{key}: {type(value).__name__}")
    if key == "repos":
        print("\nStructure of a single item in repos:")
        analyze_structure(value[0], indent=1)