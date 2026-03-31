"""
Definitive schema analysis:
- Uses paren-depth-aware parsing for CREATE TABLE (handles nested CHECK/REFERENCES parens)
- Handles RENAME COLUMN inside DO $$ ... $$ PL/pgSQL blocks
- Splits ALTER TABLE statements at proper statement boundaries
"""
import re, os, subprocess

MIGRATIONS_DIR = '/home/ubuntu/loyalty-nexus-inflight/database/migrations'
ENTITIES_DIR   = '/home/ubuntu/loyalty-nexus-inflight/backend/internal/domain/entities'

# ── 1. Read all migration files in order ─────────────────────────────────────
migration_files = sorted(
    f for f in os.listdir(MIGRATIONS_DIR) if f.endswith('.up.sql')
)

all_sql = ''
for fname in migration_files:
    with open(os.path.join(MIGRATIONS_DIR, fname)) as f:
        all_sql += f.read() + '\n'

# ── 2. Build final schema ─────────────────────────────────────────────────────
tables = {}   # table_name -> set of column names

# 2a. Find all CREATE TABLE IF NOT EXISTS using paren-depth-aware parser
pattern = re.compile(
    r'CREATE\s+TABLE\s+IF\s+NOT\s+EXISTS\s+(\w+)\s*\(',
    re.IGNORECASE
)
for m in pattern.finditer(all_sql):
    tname = m.group(1).lower()
    if tname in tables:
        continue  # IF NOT EXISTS: first definition wins

    # Walk forward to find the matching closing paren
    start = m.end() - 1   # position of the opening (
    depth, pos = 0, start
    while pos < len(all_sql):
        if all_sql[pos] == '(':
            depth += 1
        elif all_sql[pos] == ')':
            depth -= 1
            if depth == 0:
                break
        pos += 1

    body = all_sql[start + 1:pos]
    cols = set()
    for line in body.split('\n'):
        line = re.sub(r'--.*', '', line).strip().rstrip(',')
        if not line or re.match(
            r'^(CONSTRAINT|CHECK|UNIQUE|PRIMARY|FOREIGN|REFERENCES|\))',
            line, re.IGNORECASE
        ):
            continue
        col_m = re.match(r'^(\w+)\s+', line)
        if col_m:
            c = col_m.group(1).lower()
            if c not in ('constraint', 'check', 'unique', 'primary', 'foreign'):
                cols.add(c)
    tables[tname] = cols

# 2b. Extract DO $$ ... $$ blocks for special handling
do_blocks = re.findall(
    r'DO\s+\$\$\s*BEGIN(.+?)END\s*\$\$',
    all_sql, re.DOTALL | re.IGNORECASE
)

# 2c. Remove DO blocks from SQL before splitting by semicolon
sql_no_do = re.sub(
    r'DO\s+\$\$\s*BEGIN.+?END\s*\$\$\s*;?', '',
    all_sql, flags=re.DOTALL | re.IGNORECASE
)

# 2d. Split remaining SQL by semicolons and process ALTER TABLE statements
for stmt in sql_no_do.split(';'):
    stmt = re.sub(r'--[^\n]*', '', stmt).strip()
    if not stmt:
        continue

    m = re.match(r'ALTER\s+TABLE\s+(\w+)\s+(.+)', stmt, re.DOTALL | re.IGNORECASE)
    if not m:
        continue
    tname = m.group(1).lower()
    body  = m.group(2)

    if tname not in tables:
        tables[tname] = set()

    for col_m in re.finditer(
        r'ADD\s+COLUMN\s+(?:IF\s+NOT\s+EXISTS\s+)?(\w+)\s+',
        body, re.IGNORECASE
    ):
        tables[tname].add(col_m.group(1).lower())

    for col_m in re.finditer(
        r'DROP\s+COLUMN\s+(?:IF\s+EXISTS\s+)?(\w+)',
        body, re.IGNORECASE
    ):
        tables[tname].discard(col_m.group(1).lower())

    for col_m in re.finditer(
        r'RENAME\s+COLUMN\s+(\w+)\s+TO\s+(\w+)',
        body, re.IGNORECASE
    ):
        tables[tname].discard(col_m.group(1).lower())
        tables[tname].add(col_m.group(2).lower())

# 2e. Process DO blocks for ALTER TABLE operations
for block in do_blocks:
    for m in re.finditer(
        r'ALTER\s+TABLE\s+(\w+)\s+RENAME\s+COLUMN\s+(\w+)\s+TO\s+(\w+)',
        block, re.IGNORECASE
    ):
        tname, old_col, new_col = m.group(1).lower(), m.group(2).lower(), m.group(3).lower()
        if tname in tables:
            tables[tname].discard(old_col)
            tables[tname].add(new_col)

    for m in re.finditer(
        r'ALTER\s+TABLE\s+(\w+)\s+ADD\s+COLUMN\s+(?:IF\s+NOT\s+EXISTS\s+)?(\w+)\s+',
        block, re.IGNORECASE
    ):
        tname, col = m.group(1).lower(), m.group(2).lower()
        if tname in tables:
            tables[tname].add(col)

    for m in re.finditer(
        r'ALTER\s+TABLE\s+(\w+)\s+DROP\s+COLUMN\s+(?:IF\s+EXISTS\s+)?(\w+)',
        block, re.IGNORECASE
    ):
        tname, col = m.group(1).lower(), m.group(2).lower()
        if tname in tables:
            tables[tname].discard(col)

# ── 3. Parse Go entity structs ────────────────────────────────────────────────
r = subprocess.run(
    ['grep', '-rn', r'func.*TableName.*return', ENTITIES_DIR, '--include=*.go'],
    capture_output=True, text=True
)
table_name_map = {}
for line in r.stdout.split('\n'):
    m = re.search(r'func \((\w+)\) TableName.*return "(\w+)"', line)
    if m:
        table_name_map[m.group(1)] = m.group(2)

go_entity_tables = {}
for fname in os.listdir(ENTITIES_DIR):
    if not fname.endswith('.go'):
        continue
    with open(os.path.join(ENTITIES_DIR, fname)) as f:
        src = f.read()
    for struct_name, body in re.findall(
        r'type\s+(\w+)\s+struct\s*\{(.+?)\}', src, re.DOTALL
    ):
        if struct_name not in table_name_map:
            continue
        tname = table_name_map[struct_name]
        cols = set()
        for col_m in re.finditer(r'gorm:"column:(\w+)', body):
            cols.add(col_m.group(1).lower())
        if cols:
            go_entity_tables[tname] = cols

# ── 4. Print gap report ───────────────────────────────────────────────────────
print("=" * 70)
print("DEFINITIVE SCHEMA GAP REPORT")
print("=" * 70)
print(f"\nTotal tables in DB: {len(tables)}")
print(f"Total tables with Go entities: {len(go_entity_tables)}")

print("\n" + "=" * 70)
print("ORPHAN TABLES (in DB, no Go entity)")
print("=" * 70)
for t in sorted(set(tables) - set(go_entity_tables)):
    print(f"  {t}: {sorted(tables[t])}")

print("\n" + "=" * 70)
print("COLUMN GAPS: Go expects column MISSING from DB")
print("=" * 70)
any_gap = False
for tname in sorted(go_entity_tables):
    go_cols = go_entity_tables[tname]
    db_cols = tables.get(tname, set())
    missing = go_cols - db_cols - {'id'}
    if missing:
        any_gap = True
        print(f"\n  TABLE: {tname}")
        print(f"    Missing in DB: {sorted(missing)}")
        print(f"    DB has:        {sorted(db_cols)}")
if not any_gap:
    print("  NONE — all Go entity columns exist in DB")

print("\n" + "=" * 70)
print("LEGACY COLUMNS: DB has column that Go entity ignores")
print("=" * 70)
for tname in sorted(go_entity_tables):
    go_cols = go_entity_tables[tname]
    db_cols = tables.get(tname, set())
    extra = db_cols - go_cols - {'id'}
    if extra:
        print(f"\n  TABLE: {tname}")
        print(f"    Legacy/unused: {sorted(extra)}")

print("\n" + "=" * 70)
print("FULL FINAL DB SCHEMA (active tables with Go entities)")
print("=" * 70)
for tname in sorted(go_entity_tables):
    print(f"\n  {tname}: {sorted(tables.get(tname, set()))}")

print("\n" + "=" * 70)
print("ORPHAN TABLE DETAILS (full column list)")
print("=" * 70)
for t in sorted(set(tables) - set(go_entity_tables)):
    print(f"\n  {t}: {sorted(tables[t])}")
