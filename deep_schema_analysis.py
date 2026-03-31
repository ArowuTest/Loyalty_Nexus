import re, json

with open('/home/ubuntu/loyalty-nexus-inflight/all_migrations.sql') as f:
    content = f.read()

# ─── 1. Build final schema state for every table ─────────────────────────────
# Strategy: replay in order. CREATE TABLE IF NOT EXISTS = only first wins.
# ALTER TABLE ADD COLUMN IF NOT EXISTS = always applies.
# ALTER TABLE DROP COLUMN = removes.
# ALTER TABLE RENAME COLUMN = renames.

tables = {}  # table_name -> set of columns

# Find all CREATE TABLE IF NOT EXISTS blocks
for m in re.finditer(r'CREATE TABLE IF NOT EXISTS (\w+)\s*\((.+?)\);', content, re.DOTALL):
    tname = m.group(1)
    if tname in tables:
        continue  # IF NOT EXISTS: first wins
    body = m.group(2)
    cols = set()
    for line in body.split('\n'):
        line = line.strip()
        if not line or line.startswith('--') or line.startswith('CONSTRAINT') or line.startswith('CHECK') or line.startswith('UNIQUE') or line.startswith('PRIMARY') or line.startswith('FOREIGN'):
            continue
        col_match = re.match(r'^(\w+)\s+', line)
        if col_match:
            col = col_match.group(1).lower()
            if col not in ('id', 'constraint', 'check', 'unique', 'primary', 'foreign', 'references'):
                cols.add(col)
    tables[tname] = cols

# Process ALTER TABLE ADD COLUMN
for m in re.finditer(r'ALTER TABLE (\w+)\s+ADD COLUMN IF NOT EXISTS\s+(\w+)\s+', content):
    tname, col = m.group(1), m.group(2).lower()
    if tname in tables:
        tables[tname].add(col)

# Also handle multi-line ADD COLUMN blocks
for m in re.finditer(r'ALTER TABLE (\w+)\s*\n\s+ADD COLUMN IF NOT EXISTS\s+(\w+)\s+', content):
    tname, col = m.group(1), m.group(2).lower()
    if tname in tables:
        tables[tname].add(col)

# Process ALTER TABLE ADD COLUMN (without IF NOT EXISTS)
for m in re.finditer(r'ALTER TABLE (\w+)\s+ADD COLUMN\s+(?!IF)(\w+)\s+', content):
    tname, col = m.group(1), m.group(2).lower()
    if tname in tables:
        tables[tname].add(col)

# Process ALTER TABLE DROP COLUMN
for m in re.finditer(r'ALTER TABLE (\w+)\s+DROP COLUMN\s+(?:IF EXISTS\s+)?(\w+)', content):
    tname, col = m.group(1), m.group(2).lower()
    if tname in tables and col in tables[tname]:
        tables[tname].discard(col)

# Process ALTER TABLE RENAME COLUMN
for m in re.finditer(r'ALTER TABLE (\w+)\s+RENAME COLUMN\s+(\w+)\s+TO\s+(\w+)', content):
    tname, old_col, new_col = m.group(1), m.group(2).lower(), m.group(3).lower()
    if tname in tables:
        tables[tname].discard(old_col)
        tables[tname].add(new_col)

# ─── 2. Extract Go entity struct fields ──────────────────────────────────────
import subprocess
result = subprocess.run(
    ['grep', '-rn', r'gorm:"column:', '/home/ubuntu/loyalty-nexus-inflight/backend/', '--include=*.go'],
    capture_output=True, text=True
)

go_tables = {}  # table_name -> set of columns
# Also get TableName() mappings
result2 = subprocess.run(
    ['grep', '-rn', r'TableName.*return', '/home/ubuntu/loyalty-nexus-inflight/backend/', '--include=*.go'],
    capture_output=True, text=True
)
# Parse TableName -> struct name
table_name_map = {}
for line in result2.stdout.split('\n'):
    m = re.search(r'func \((\w+)\) TableName.*return "(\w+)"', line)
    if m:
        table_name_map[m.group(1)] = m.group(2)

# Parse gorm column tags
for line in result.stdout.split('\n'):
    col_m = re.search(r'gorm:"column:(\w+)', line)
    if not col_m:
        continue
    col = col_m.group(1).lower()
    # Try to find the file and struct
    file_m = re.match(r'(/[^:]+\.go):', line)
    if file_m:
        fname = file_m.group(1)
        # We'll group by file for now
        if fname not in go_tables:
            go_tables[fname] = set()
        go_tables[fname].add(col)

# Better: parse each entity file to get struct -> table mapping
import os
entity_dir = '/home/ubuntu/loyalty-nexus-inflight/backend/internal/domain/entities'
go_entity_tables = {}  # table_name -> set of columns

for fname in os.listdir(entity_dir):
    if not fname.endswith('.go'):
        continue
    fpath = os.path.join(entity_dir, fname)
    with open(fpath) as f:
        src = f.read()
    
    # Find all struct definitions and their TableName
    structs = re.findall(r'type (\w+) struct \{(.+?)\}', src, re.DOTALL)
    for struct_name, body in structs:
        if struct_name not in table_name_map:
            continue
        tname = table_name_map[struct_name]
        cols = set()
        for col_m in re.finditer(r'gorm:"column:(\w+)', body):
            cols.add(col_m.group(1).lower())
        if cols:
            go_entity_tables[tname] = cols

# ─── 3. Print gap analysis ────────────────────────────────────────────────────
print("=" * 70)
print("DEFINITIVE SCHEMA GAP ANALYSIS")
print("=" * 70)
print(f"\nTotal tables in migrations: {len(tables)}")
print(f"Total tables with Go entities: {len(go_entity_tables)}")

print("\n" + "=" * 70)
print("TABLES WITH NO GO ENTITY (orphan/stale tables)")
print("=" * 70)
orphan = sorted(set(tables.keys()) - set(go_entity_tables.keys()))
for t in orphan:
    cols = sorted(tables[t])
    print(f"\n  {t}")
    print(f"    columns: {', '.join(cols)}")

print("\n" + "=" * 70)
print("COLUMN GAPS: Go expects column that DB does NOT have")
print("=" * 70)
for tname in sorted(go_entity_tables.keys()):
    go_cols = go_entity_tables[tname]
    db_cols = tables.get(tname, set())
    missing_in_db = go_cols - db_cols
    if missing_in_db:
        print(f"\n  TABLE: {tname}")
        print(f"    Go expects but DB lacks: {sorted(missing_in_db)}")
        print(f"    DB has: {sorted(db_cols)}")

print("\n" + "=" * 70)
print("COLUMN GAPS: DB has column that Go entity does NOT reference")
print("=" * 70)
for tname in sorted(go_entity_tables.keys()):
    go_cols = go_entity_tables[tname]
    db_cols = tables.get(tname, set())
    missing_in_go = db_cols - go_cols
    if missing_in_go:
        print(f"\n  TABLE: {tname}")
        print(f"    DB has but Go ignores: {sorted(missing_in_go)}")

print("\n" + "=" * 70)
print("FULL FINAL DB SCHEMA (all tables)")
print("=" * 70)
for tname in sorted(tables.keys()):
    print(f"\n  {tname}: {sorted(tables[tname])}")

