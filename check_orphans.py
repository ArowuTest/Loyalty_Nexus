import os, re

orphan_tables = [
    'arpu_uplift_tracking', 'chat_messages', 'chat_session_summaries', 'chat_sessions',
    'churn_recovery_bounties', 'draw_entries', 'draw_schedules', 'draw_winners', 'draws',
    'fraud_events', 'fulfilment_webhooks', 'ledger_entries', 'msisdn_blacklist',
    'mtn_push_csv_rows', 'mtn_push_csv_uploads', 'mtn_push_events', 'multiplier_audit_logs',
    'network_cache', 'network_configs', 'notification_broadcasts', 'notification_preferences',
    'notifications', 'passport_push_log', 'points_expiry_policies', 'prize_claims',
    'prize_fulfillment_logs', 'program_bonuses', 'program_configs', 'pulse_point_awards',
    'push_tokens', 'qr_scan_log', 'recharge_tiers', 'region_tournaments', 'regional_settings',
    'regional_stats', 'regional_wars_cycles', 'regional_wars_snapshots', 'scheduled_multipliers',
    'segment_multipliers', 'session_summaries', 'sms_templates', 'spin_tiers', 'studio_config',
    'studio_usage_metrics', 'subscription_events', 'subscription_plans', 'user_badges',
    'user_subscriptions', 'users', 'wallet_passes'
]

go_files = []
for root, dirs, files in os.walk('/home/ubuntu/loyalty-nexus-inflight/backend/internal'):
    for file in files:
        if file.endswith('.go'):
            go_files.append(os.path.join(root, file))

used_tables = set()
for file in go_files:
    with open(file, 'r') as f:
        content = f.read()
        for table in orphan_tables:
            # Look for the table name as a whole word
            if re.search(r'\b' + table + r'\b', content):
                used_tables.add(table)

print("Orphan tables actually used in Go code:")
for table in sorted(used_tables):
    print(f"  - {table}")

print("\nTruly orphaned tables (safe to drop):")
for table in sorted(set(orphan_tables) - used_tables):
    print(f"  - {table}")
