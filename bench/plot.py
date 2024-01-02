#!/usr/bin/python

# Install matlibplot e.g. via 'pip install matplotlib' or via 'apt install python3-matplotlib'
import matplotlib.pyplot as plt

# Specify the path to your file
file_path = "bench.out"

goos = ""
goarch = ""
pkg = ""
cpu = ""
nCPU = "1"
funcName = ""

# Read data from the file and parse it into the data variable
data = []
with (open(file_path, "r") as file):
    for line in file:
        # Assuming each line has the format "BenchmarkName value"
        parts = line.split()
        if len(parts) >= 2 and parts[0].endswith(":"):
            if parts[0] == "goos:":
                goos = parts[1]
            if parts[0] == "goarch:":
                goarch = parts[1]
            if parts[0] == "cpu:":
                cpu = " ".join(parts[1:])
            continue
        if len(parts) == 3:  # skip summary
            continue
        if len(parts) >= 3:
            if '-' in parts[0]:
                nCPU = parts[0].split('-')[1]
            s = parts[0].split('-')[0]
            if s.endswith('Get'):
                keyType = 'int'
                valType = 'int'
            else:
                keyType = s.split('_')[1]
                valType = s.split('_')[2]
                s = s.split('_')[0]
            name = s.replace('Benchmark', '')
            name = name.replace('Parallel', '')
            for suffix in ('Add', 'Get'):
                if name.endswith(suffix):
                    name = name[:-len(suffix)]
                    funcName = suffix
                    break

            value = float(parts[2])
            data.append((name, value))

# Sort data by value
data = sorted(data, key=lambda x: x[1])

# Extracting names and values from the data
names, values = zip(*data)

# Creating the bar chart
fig, ax = plt.subplots(figsize=(10, 8))
bars = ax.barh(names, values, color='skyblue')
plt.xlabel('Execution Time (ns/op)')
plt.title(f'OS: {goos}\nArch: {goarch}\nCPU: {cpu}\nGOMAXPROCS: {nCPU}\n\n{funcName}({keyType}, {valType}) Execution Time (smaller is better)')
ax.invert_yaxis()

# Adding text inside the bars
for bar in bars:
    value = bar.get_width()
    ax.text(2, bar.get_y() + bar.get_height() / 2, f'{value}', va='center', ha='left', fontsize=8, color='black')

plt.savefig('bench.png', bbox_inches='tight')
plt.show()
