-- Migration 039: Seed ussd_sms_number config key for USSD Knowledge Tools SMS instruction.
--
-- REQ-6.4: When ussd_sms_number is set, the USSD Knowledge Tools sub-menu (option 7)
-- prepends an instruction line: "Send topic via SMS to <number>".
-- This key was referenced in the handler but never seeded in any prior migration.
--
-- The default value is empty so the instruction line is hidden until an admin sets
-- the real SMS number via the admin config panel (PUT /api/v1/admin/config/ussd_sms_number).
-- ─── Seed ussd_sms_number ─────────────────────────────────────────────────────
INSERT INTO network_configs (key, value, description, is_public, updated_by)
VALUES
    ('ussd_sms_number', '',
     'SMS number displayed in the USSD Knowledge Tools sub-menu as an alternative entry point. '
     'Leave empty to hide the instruction line. Set to e.g. "08012345678" to show it.',
     false, 'system')
ON CONFLICT (key) DO NOTHING;
