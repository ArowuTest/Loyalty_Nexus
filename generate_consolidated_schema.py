"""
Generates a clean, consolidated baseline schema (V2) representing the final state
of all tables after all 72 migrations.
"""
import re, os

MIGRATIONS_DIR = '/home/ubuntu/loyalty-nexus-inflight/database/migrations'

# 1. Read all migrations
migration_files = sorted(f for f in os.listdir(MIGRATIONS_DIR) if f.endswith('.up.sql'))
all_sql = ''
for fname in migration_files:
    with open(os.path.join(MIGRATIONS_DIR, fname)) as f:
        all_sql += f.read() + '\n'

# 2. Extract all CREATE TABLE statements (paren-depth-aware)
tables = {} # name -> full CREATE TABLE statement
pattern = re.compile(r'CREATE\s+TABLE\s+IF\s+NOT\s+EXISTS\s+(\w+)\s*\(', re.IGNORECASE)
for m in pattern.finditer(all_sql):
    tname = m.group(1).lower()
    if tname in tables:
        continue
    
    start = m.end() - 1
    depth, pos = 0, start
    while pos < len(all_sql):
        if all_sql[pos] == '(': depth += 1
        elif all_sql[pos] == ')':
            depth -= 1
            if depth == 0: break
        pos += 1
    
    # Extract the full statement including the closing paren and semicolon
    stmt = all_sql[m.start():pos+1] + ';'
    tables[tname] = stmt

# 3. Extract all CREATE INDEX statements
indexes = []
for line in all_sql.split('\n'):
    if re.match(r'^\s*CREATE\s+(UNIQUE\s+)?INDEX', line, re.IGNORECASE):
        indexes.append(line.strip().rstrip(';') + ';')

# 4. We need to apply the ALTER TABLE modifications to the CREATE TABLE statements.
# This is complex to do purely via regex. A better approach for a consolidated schema
# is to dump the schema from Postgres directly if we can load it, or we can just 
# provide the consolidated schema based on the parsed columns.

# Since we already have the precise final columns from precise_schema.py, we can 
# construct the consolidated schema by taking the original CREATE TABLE and adding 
# the missing columns, or we can just use pg_dump if we load the migrations into a DB.
"""
