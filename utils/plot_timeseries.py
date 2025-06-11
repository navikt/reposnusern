import json
import pandas as pd
import matplotlib.pyplot as plt
import glob
import re
import platform
import multiprocessing
from datetime import datetime

# 🔍 Finn siste raw_stats-fil
files = sorted(glob.glob("raw_stats_*.json"))
if not files:
    print("❌ Ingen rådatafiler funnet.")
    exit(1)

raw_file = files[-1]
print(f"🗂️ Leser rådata: {raw_file}")

# 🧠 Hent info om systemets CPU
cpu_model = platform.processor() or "ukjent"
cpu_cores = multiprocessing.cpu_count()

rows = []
buffer = ""

# 📥 Parse datafil
with open(raw_file, "r") as f:
    for line in f:
        line = line.strip()
        if not line:
            continue

        buffer += line
        if line.endswith("]"):
            try:
                data_list = json.loads(buffer)
                if not isinstance(data_list, list) or not data_list:
                    continue

                data = data_list[0]
                timestamp = datetime.now()

                # 🎯 Parse minnebruk (eks. "5.112MB / 7.716GB")
                mem_raw = data.get("mem_usage", "").split("/")[0].strip()
                match = re.match(r"([\d.]+)([KMG]?B)", mem_raw)
                if match:
                    num, unit = match.groups()
                    factor = {"KB": 1/1024, "MB": 1, "GB": 1024}.get(unit, 1)
                    mem_mib = float(num) * factor
                else:
                    mem_mib = 0.0

                # 🎯 Parse CPU (eks. "6.05%")
                cpu_percent = float(data.get("cpu_percent", "0").replace("%", "").replace(",", "."))
                cpu_mcpu = cpu_percent * 10

                rows.append({
                    "timestamp": timestamp,
                    "mem_usage_mib": mem_mib,
                    "cpu_percent": cpu_percent,
                    "cpu_mcpu": cpu_mcpu,
                })

            except Exception as e:
                print(f"⚠️ Feil under parsing:\n{buffer[:80]}...\n→ {e}")
            buffer = ""

# 📊 Lag DataFrame og lagre CSV
df = pd.DataFrame(rows)
csv_name = raw_file.replace("raw_stats_", "benchmark_timeseries_").replace(".json", ".csv")
df.to_csv(csv_name, index=False)
print(f"✅ Lagret tidsserie: {csv_name}")

# 🔢 Statistikk og k8s-formatering
def format_cpu(v): return f"{int(round(v))}m"
def format_mem(v): return f"{int(round(v + 8))}Mi"  # legg til buffer

cpu_request = format_cpu(df["cpu_mcpu"].mean())
cpu_limit   = format_cpu(df["cpu_mcpu"].max())
mem_request = format_mem(df["mem_usage_mib"].mean())
mem_limit   = format_mem(df["mem_usage_mib"].max())

# 📈 Plotting
df["timestamp"] = pd.to_datetime(df["timestamp"], errors="coerce")
fig, (ax1, ax2) = plt.subplots(2, 1, figsize=(12, 7), sharex=True)

# Minne
ax1.plot(df["timestamp"], df["mem_usage_mib"], label="Minnebruk (MiB)", color="steelblue")
ax1.set_ylabel("MiB")
ax1.set_title("Minnebruk over tid")
ax1.grid(True)
ax1.legend(loc="upper right")

# CPU
ax2.plot(df["timestamp"], df["cpu_mcpu"], label="CPU-bruk (m)", color="darkorange")
ax2.set_ylabel("CPU (milli-cores)")
ax2.set_xlabel("Tid")
ax2.set_title(f"CPU-bruk over tid ({cpu_model}, {cpu_cores} kjerner)")
ax2.grid(True)
ax2.legend(loc="upper right")

# 💬 Annotasjon
summary = (
    f"Kubernetes resource-anbefaling:\n"
    f"  requests:\n"
    f"    memory: {mem_request}\n"
    f"    cpu:    {cpu_request}\n"
    f"  limits:\n"
    f"    memory: {mem_limit}\n"
    f"    cpu:    {cpu_limit}"
)
fig.text(0.01, 0.05, summary, fontsize=9, bbox=dict(boxstyle='round', facecolor='wheat', alpha=0.5))

plt.tight_layout(rect=[0, 0.15, 1, 1])  # plass til annotasjon

# 💾 Lagre bilde og vis
img_path = raw_file.replace(".json", "_plot.png").replace("raw_stats_", "benchmark_plot_")
plt.savefig(img_path, dpi=150)
print(f"🖼️ Lagret bilde: {img_path}")
plt.show()
