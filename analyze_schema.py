#!/usr/bin/env python3
"""
Independent schema consolidation analyzer for Loyalty Nexus.
Reads all migration SQL files and Go source files, then:
1. Reconstructs the final authoritative schema for every table
2. Identifies all tables referenced in Go code
3. Classifies tables as: ACTIVE, ORPHANED (in DB, not in Go), or GHOST (in Go, not in DB)
4. Finds column mismatches between Go structs and DB schema
5. Finds constraint issues
"""

import os, re, glob
from collections import defaultdict

MIGRATIONS_DIR = "/home/ubuntu/loyalty-nexus-inflight/database/migrations"
GO_DIR = "/home/ubuntu/loyalty-nexus-inflight/backend"

# ─── Step 1: Parse all migrations in order ───────────────────────────────────

migration_files = sorted(glob.glob(f"{MIGRATIONS_DIR}/*.up.sql"))
print(f"Found {len(migration_files)} migration files\n")

# Track: table_name -> {column_name -> column_def}
schema = defaultdict(dict)  # table -> {col -> definition}
table_constraints = defaultdict(list)  # table -> [constraint strings]
dropped_tables = set()
dropped_columns = defaultdict(set)  # table -> {col}
renamed_columns = defaultdict(dict)  # table -> {old_col -> new_col}
table_first_seen = {}  # table -> migration file

def extract_columns_from_create(create_sql):
    """Extract column definitions from a CREATE TABLE statement."""
    cols = {}
    # Find the body between first ( and last )
    m = re.search(r'\((.+)\)', create_sql, re.DOTALL)
    if not m:
        return cols
    body = m.group(1)
    # Split by commas at top level (not inside parens)
    depth = 0
    current = ""
    lines = []
    for ch in body:
        if ch == '(':
            depth += 1
            current += ch
        elif ch == ')':
            depth -= 1
            current += ch
        elif ch == ',' and depth == 0:
            lines.append(current.strip())
            current = ""
        else:
            current += ch
    if current.strip():
        lines.append(current.strip())
    
    for line in lines:
        line = line.strip()
        if not line:
            continue
        # Skip constraints
        if re.match(r'(PRIMARY KEY|UNIQUE|CHECK|FOREIGN KEY|CONSTRAINT|INDEX)', line, re.I):
            continue
        # Extract column name (first word)
        parts = line.split()
        if len(parts) >= 2:
            col_name = parts[0].strip('"')
            col_def = ' '.join(parts[1:])
            cols[col_name] = col_def
    return cols

all_sql = ""
for mf in migration_files:
    mname = os.path.basename(mf)
    with open(mf) as f:
        sql = f.read()
    all_sql += f"\n-- FROM {mname}\n" + sql

    # Find CREATE TABLE statements
    for m in re.finditer(
        r'CREATE TABLE IF NOT EXISTS\s+(\w+)\s*\((.+?)\);',
        sql, re.DOTALL | re.IGNORECASE
    ):
        tname = m.group(1)
        if tname not in table_first_seen:
            table_first_seen[tname] = mname
        cols = extract_columns_from_create(m.group(0))
        # CREATE TABLE IF NOT EXISTS: only add cols if table not yet seen
        # (idempotent — later migrations use ALTER TABLE to add cols)
        if tname not in schema:
            schema[tname] = cols
        if tname in dropped_tables:
            dropped_tables.discard(tname)
            schema[tname] = cols  # recreated

    # Find DROP TABLE
    for m in re.finditer(r'DROP TABLE\s+(?:IF EXISTS\s+)?(\w+)', sql, re.IGNORECASE):
        dropped_tables.add(m.group(1))
        schema.pop(m.group(1), None)

    # Find ALTER TABLE ADD COLUMN
    for m in re.finditer(
        r'ALTER TABLE\s+(\w+)\s+ADD COLUMN IF NOT EXISTS\s+(\w+)\s+([^,;]+)',
        sql, re.IGNORECASE
    ):
        tname, col, coldef = m.group(1), m.group(2), m.group(3).strip()
        schema[tname][col] = coldef

    for m in re.finditer(
        r'ALTER TABLE\s+(\w+)\s+ADD COLUMN\s+(\w+)\s+([^,;]+)',
        sql, re.IGNORECASE
    ):
        tname, col, coldef = m.group(1), m.group(2), m.group(3).strip()
        if col.upper() not in ('IF',):
            schema[tname][col] = coldef

    # Find ALTER TABLE DROP COLUMN
    for m in re.finditer(
        r'ALTER TABLE\s+(\w+)\s+DROP COLUMN\s+(?:IF EXISTS\s+)?(\w+)',
        sql, re.IGNORECASE
    ):
        tname, col = m.group(1), m.group(2)
        schema[tname].pop(col, None)
        dropped_columns[tname].add(col)

    # Find ALTER TABLE RENAME COLUMN
    for m in re.finditer(
        r'ALTER TABLE\s+(\w+)\s+RENAME COLUMN\s+(\w+)\s+TO\s+(\w+)',
        sql, re.IGNORECASE
    ):
        tname, old_col, new_col = m.group(1), m.group(2), m.group(3)
        if old_col in schema[tname]:
            col_def = schema[tname].pop(old_col)
            schema[tname][new_col] = col_def
        renamed_columns[tname][old_col] = new_col

print(f"Tables in final schema: {len(schema)}")
print(f"Dropped tables: {dropped_tables}\n")

# ─── Step 2: Parse Go code for table references ───────────────────────────────

go_files = []
for root, dirs, files in os.walk(GO_DIR):
    for fn in files:
        if fn.endswith('.go'):
            go_files.append(os.path.join(root, fn))

print(f"Found {len(go_files)} Go files\n")

go_table_refs = defaultdict(set)  # table -> set of files referencing it
go_column_refs = defaultdict(set)  # table -> set of columns referenced
go_struct_fields = defaultdict(dict)  # struct_name -> {field -> gorm_col}

for gf in go_files:
    with open(gf) as f:
        content = f.read()
    
    # TableName() method
    for m in re.finditer(r'func\s*\(\w+\)\s*TableName\(\)\s*string\s*\{\s*return\s*"(\w+)"', content):
        go_table_refs[m.group(1)].add(os.path.basename(gf))
    
    # .Table("tablename")
    for m in re.finditer(r'\.Table\("(\w+)"\)', content):
        go_table_refs[m.group(1)].add(os.path.basename(gf))
    
    # Raw SQL: FROM tablename, JOIN tablename, INSERT INTO tablename, UPDATE tablename
    for m in re.finditer(r'(?:FROM|JOIN|INTO|UPDATE)\s+(\w+)(?:\s|,|\()', content, re.IGNORECASE):
        tname = m.group(1).lower()
        if len(tname) > 3 and tname not in ('the', 'set', 'not', 'and', 'for', 'all'):
            go_table_refs[tname].add(os.path.basename(gf))
    
    # gorm:"column:colname" tags — extract struct fields
    struct_name = None
    for line in content.split('\n'):
        sm = re.search(r'^type\s+(\w+)\s+struct', line)
        if sm:
            struct_name = sm.group(1)
        if struct_name:
            cm = re.search(r'gorm:"[^"]*column:(\w+)', line)
            if cm:
                col = cm.group(1)
                # Try to find table from TableName or struct name
                go_struct_fields[struct_name][col] = line.strip()
    
    # Raw SQL column references in strings
    for m in re.finditer(r'"([a-z_]+\s*=\s*\?|[a-z_]+\s+[A-Z]|SELECT\s+[^"]+FROM\s+(\w+))"', content):
        pass  # collected separately below

# ─── Step 3: Extract raw SQL column references ───────────────────────────────

raw_sql_cols = defaultdict(set)  # table -> set of columns used in raw SQL

for gf in go_files:
    with open(gf) as f:
        content = f.read()
    
    # Find raw SQL strings
    for m in re.finditer(r'`([^`]+)`|"([^"]+)"', content):
        sql_str = m.group(1) or m.group(2)
        if not any(kw in sql_str.upper() for kw in ['SELECT', 'WHERE', 'UPDATE', 'INSERT', 'FROM', 'JOIN', 'ORDER']):
            continue
        # Extract table from FROM/UPDATE/INTO
        tbl_m = re.search(r'(?:FROM|UPDATE|INTO|JOIN)\s+(\w+)', sql_str, re.IGNORECASE)
        if tbl_m:
            tbl = tbl_m.group(1).lower()
            # Extract column names (word before = or after SELECT/WHERE/AND/OR/ORDER BY)
            cols = re.findall(r'\b([a-z_][a-z0-9_]*)\s*(?:=\s*\?|IS\s+NULL|IS\s+NOT\s+NULL|ASC|DESC|,|\s+AS\s)', sql_str)
            for c in cols:
                if len(c) > 2 and c not in ('and', 'or', 'not', 'null', 'true', 'false', 'asc', 'desc', 'as', 'is', 'in', 'by', 'on', 'at'):
                    raw_sql_cols[tbl].add(c)

# ─── Step 4: Classify tables ─────────────────────────────────────────────────

all_db_tables = set(schema.keys())
all_go_tables = set(go_table_refs.keys())

active_tables = all_db_tables & all_go_tables
orphaned_tables = all_db_tables - all_go_tables  # in DB, not in Go
ghost_tables = all_go_tables - all_db_tables      # in Go, not in DB

print("=" * 60)
print("TABLE CLASSIFICATION")
print("=" * 60)
print(f"\n✅ ACTIVE (in both DB and Go): {len(active_tables)}")
for t in sorted(active_tables):
    print(f"   {t}")

print(f"\n⚠️  ORPHANED (in DB migrations, NOT referenced in Go): {len(orphaned_tables)}")
for t in sorted(orphaned_tables):
    print(f"   {t}  [first seen: {table_first_seen.get(t,'?')}]")

print(f"\n❌ GHOST (referenced in Go, NOT in DB migrations): {len(ghost_tables)}")
for t in sorted(ghost_tables):
    print(f"   {t}")

# ─── Step 5: Column gap analysis for active tables ───────────────────────────

print("\n" + "=" * 60)
print("COLUMN GAP ANALYSIS (DB schema vs Go struct fields)")
print("=" * 60)

# Build a mapping: table -> set of columns from Go structs
# We need to match struct names to table names
# Use TableName() method output from go_table_refs and struct names

# Build struct->table mapping from TableName methods
struct_to_table = {}
for gf in go_files:
    with open(gf) as f:
        content = f.read()
    # Match: type FooBar struct ... func (FooBar) TableName() string { return "foo_bars" }
    for m in re.finditer(r'func\s*\((\w+)\)\s*TableName\(\)\s*string\s*\{\s*return\s*"(\w+)"', content):
        struct_to_table[m.group(1)] = m.group(2)

# For each active table, compare DB columns vs Go struct columns
issues = []
for tname in sorted(active_tables):
    db_cols = set(schema[tname].keys())
    
    # Find Go struct columns for this table
    go_cols = set()
    for sname, fields in go_struct_fields.items():
        if struct_to_table.get(sname) == tname:
            go_cols.update(fields.keys())
    
    if not go_cols:
        continue  # No struct found for this table
    
    missing_in_db = go_cols - db_cols
    missing_in_go = db_cols - go_cols
    
    if missing_in_db or missing_in_go:
        print(f"\n  Table: {tname}")
        if missing_in_db:
            print(f"    ❌ Go expects but DB lacks: {sorted(missing_in_db)}")
            issues.append(('missing_in_db', tname, sorted(missing_in_db)))
        if missing_in_go:
            # Filter out common auto-managed columns
            significant = [c for c in missing_in_go if c not in 
                          ('created_at', 'updated_at', 'deleted_at', 'id')]
            if significant:
                print(f"    ⚠️  DB has but Go ignores: {sorted(significant)}")

# ─── Step 6: Check renamed/dropped columns still used in Go ─────────────────

print("\n" + "=" * 60)
print("RENAMED/DROPPED COLUMN USAGE IN GO CODE")
print("=" * 60)

for tname, renames in renamed_columns.items():
    for old_col, new_col in renames.items():
        # Check if old_col is still referenced in Go
        for gf in go_files:
            with open(gf) as f:
                content = f.read()
            if f'"{old_col}"' in content or f'`{old_col}`' in content:
                print(f"  ⚠️  {tname}.{old_col} was renamed to {new_col} but still used in {os.path.basename(gf)}")

for tname, dropped_cols in dropped_columns.items():
    for col in dropped_cols:
        for gf in go_files:
            with open(gf) as f:
                content = f.read()
            if f'"{col}"' in content or f'`{col}`' in content or f'column:{col}' in content:
                print(f"  ❌  {tname}.{col} was DROPPED but still referenced in {os.path.basename(gf)}")

# ─── Step 7: Check raw SQL column refs against DB schema ────────────────────

print("\n" + "=" * 60)
print("RAW SQL COLUMN REFERENCES VS DB SCHEMA")
print("=" * 60)

for tname in sorted(raw_sql_cols.keys()):
    if tname not in schema:
        print(f"  ❌ Raw SQL references table '{tname}' which does NOT exist in DB schema")
        continue
    db_cols = set(schema[tname].keys())
    for col in sorted(raw_sql_cols[tname]):
        if col not in db_cols and col not in ('id', 'created_at', 'updated_at'):
            print(f"  ❌ Raw SQL: {tname}.{col} — column not in DB schema")

# ─── Step 8: Print final schema for each active table ───────────────────────

print("\n" + "=" * 60)
print("FINAL AUTHORITATIVE SCHEMA (active tables)")
print("=" * 60)
for tname in sorted(active_tables):
    print(f"\n  [{tname}]")
    for col, defn in sorted(schema[tname].items()):
        print(f"    {col:35s} {defn[:80]}")

print("\n\nDone.")
