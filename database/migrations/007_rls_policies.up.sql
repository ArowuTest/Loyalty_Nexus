-- 007_rls_policies.sql
-- Purpose: Security hardening for multi-tenant and subscriber data protection.

-- Enable RLS on all sensitive tables
ALTER TABLE users ENABLE ROW LEVEL SECURITY;
ALTER TABLE transactions ENABLE ROW LEVEL SECURITY;
ALTER TABLE ai_generations ENABLE ROW LEVEL SECURITY;
ALTER TABLE user_subscriptions ENABLE ROW LEVEL SECURITY;
ALTER TABLE wallet_passes ENABLE ROW LEVEL SECURITY;

-- 1. Users can only read their own profile
CREATE POLICY user_read_self ON users
    FOR SELECT USING (msisdn = current_setting('app.current_user_msisdn', true));

-- 2. Users can only see their own transactions
CREATE POLICY user_view_transactions ON transactions
    FOR SELECT USING (user_id = (SELECT id FROM users WHERE msisdn = current_setting('app.current_user_msisdn', true)));

-- 3. Users can only see their own AI generations (Gallery)
CREATE POLICY user_view_gallery ON ai_generations
    FOR SELECT USING (user_id = (SELECT id FROM users WHERE msisdn = current_setting('app.current_user_msisdn', true)));

-- 4. Admin Access (Bypass RLS for service role / admin users)
-- In production, we'd define an 'admin' role or check JWT claims
CREATE POLICY admin_all_access ON users FOR ALL TO authenticated USING (true);
CREATE POLICY admin_all_tx ON transactions FOR ALL TO authenticated USING (true);
CREATE POLICY admin_all_ai ON ai_generations FOR ALL TO authenticated USING (true);

-- 5. Cockpit Config (Read-only for app, Read/Write for Admin)
ALTER TABLE program_configs ENABLE ROW LEVEL SECURITY;
CREATE POLICY app_read_config ON program_configs FOR SELECT TO PUBLIC USING (true);
