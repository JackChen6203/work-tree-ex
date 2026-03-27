-- Phase 2: Supabase-compatible Row Level Security
-- - enable RLS on all application tables
-- - allow backend service role to bypass via app.role=request context
-- - define user-facing select policies for key tables

CREATE OR REPLACE FUNCTION public.app_uid()
RETURNS UUID
LANGUAGE plpgsql
STABLE
AS $$
DECLARE
    claim TEXT;
BEGIN
    claim := NULLIF(current_setting('request.jwt.claim.sub', true), '');
    IF claim IS NULL THEN
        claim := NULLIF(current_setting('app.current_user_id', true), '');
    END IF;
    IF claim IS NULL THEN
        RETURN NULL;
    END IF;
    RETURN claim::UUID;
EXCEPTION
    WHEN others THEN
        RETURN NULL;
END;
$$;

CREATE OR REPLACE FUNCTION public.app_is_service_role()
RETURNS BOOLEAN
LANGUAGE sql
STABLE
AS $$
SELECT
    COALESCE(
        NULLIF(current_setting('request.jwt.claim.role', true), ''),
        NULLIF(current_setting('app.role', true), '')
    ) = 'service_role'
    OR current_user = 'service_role'
    OR current_user = 'postgres'
    OR current_user LIKE 'postgres.%';
$$;

ALTER TABLE users ENABLE ROW LEVEL SECURITY;
ALTER TABLE user_preferences ENABLE ROW LEVEL SECURITY;
ALTER TABLE llm_provider_configs ENABLE ROW LEVEL SECURITY;
ALTER TABLE sessions ENABLE ROW LEVEL SECURITY;
ALTER TABLE trips ENABLE ROW LEVEL SECURITY;
ALTER TABLE trip_memberships ENABLE ROW LEVEL SECURITY;
ALTER TABLE trip_invitations ENABLE ROW LEVEL SECURITY;
ALTER TABLE itinerary_days ENABLE ROW LEVEL SECURITY;
ALTER TABLE place_snapshots ENABLE ROW LEVEL SECURITY;
ALTER TABLE route_snapshots ENABLE ROW LEVEL SECURITY;
ALTER TABLE itinerary_items ENABLE ROW LEVEL SECURITY;
ALTER TABLE trip_idempotency_keys ENABLE ROW LEVEL SECURITY;
ALTER TABLE budget_profiles ENABLE ROW LEVEL SECURITY;
ALTER TABLE expenses ENABLE ROW LEVEL SECURITY;
ALTER TABLE ai_plan_requests ENABLE ROW LEVEL SECURITY;
ALTER TABLE ai_plan_drafts ENABLE ROW LEVEL SECURITY;
ALTER TABLE ai_plan_validation_results ENABLE ROW LEVEL SECURITY;
ALTER TABLE notifications ENABLE ROW LEVEL SECURITY;
ALTER TABLE share_links ENABLE ROW LEVEL SECURITY;
ALTER TABLE outbox_events ENABLE ROW LEVEL SECURITY;
ALTER TABLE audit_logs ENABLE ROW LEVEL SECURITY;
ALTER TABLE fcm_tokens ENABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS users_service_role_all ON users;
CREATE POLICY users_service_role_all ON users
    FOR ALL
    USING (public.app_is_service_role())
    WITH CHECK (public.app_is_service_role());

DROP POLICY IF EXISTS user_preferences_service_role_all ON user_preferences;
CREATE POLICY user_preferences_service_role_all ON user_preferences
    FOR ALL
    USING (public.app_is_service_role())
    WITH CHECK (public.app_is_service_role());

DROP POLICY IF EXISTS llm_provider_configs_service_role_all ON llm_provider_configs;
CREATE POLICY llm_provider_configs_service_role_all ON llm_provider_configs
    FOR ALL
    USING (public.app_is_service_role())
    WITH CHECK (public.app_is_service_role());

DROP POLICY IF EXISTS sessions_service_role_all ON sessions;
CREATE POLICY sessions_service_role_all ON sessions
    FOR ALL
    USING (public.app_is_service_role())
    WITH CHECK (public.app_is_service_role());

DROP POLICY IF EXISTS trips_service_role_all ON trips;
CREATE POLICY trips_service_role_all ON trips
    FOR ALL
    USING (public.app_is_service_role())
    WITH CHECK (public.app_is_service_role());

DROP POLICY IF EXISTS trip_memberships_service_role_all ON trip_memberships;
CREATE POLICY trip_memberships_service_role_all ON trip_memberships
    FOR ALL
    USING (public.app_is_service_role())
    WITH CHECK (public.app_is_service_role());

DROP POLICY IF EXISTS trip_invitations_service_role_all ON trip_invitations;
CREATE POLICY trip_invitations_service_role_all ON trip_invitations
    FOR ALL
    USING (public.app_is_service_role())
    WITH CHECK (public.app_is_service_role());

DROP POLICY IF EXISTS itinerary_days_service_role_all ON itinerary_days;
CREATE POLICY itinerary_days_service_role_all ON itinerary_days
    FOR ALL
    USING (public.app_is_service_role())
    WITH CHECK (public.app_is_service_role());

DROP POLICY IF EXISTS place_snapshots_service_role_all ON place_snapshots;
CREATE POLICY place_snapshots_service_role_all ON place_snapshots
    FOR ALL
    USING (public.app_is_service_role())
    WITH CHECK (public.app_is_service_role());

DROP POLICY IF EXISTS route_snapshots_service_role_all ON route_snapshots;
CREATE POLICY route_snapshots_service_role_all ON route_snapshots
    FOR ALL
    USING (public.app_is_service_role())
    WITH CHECK (public.app_is_service_role());

DROP POLICY IF EXISTS itinerary_items_service_role_all ON itinerary_items;
CREATE POLICY itinerary_items_service_role_all ON itinerary_items
    FOR ALL
    USING (public.app_is_service_role())
    WITH CHECK (public.app_is_service_role());

DROP POLICY IF EXISTS trip_idempotency_keys_service_role_all ON trip_idempotency_keys;
CREATE POLICY trip_idempotency_keys_service_role_all ON trip_idempotency_keys
    FOR ALL
    USING (public.app_is_service_role())
    WITH CHECK (public.app_is_service_role());

DROP POLICY IF EXISTS budget_profiles_service_role_all ON budget_profiles;
CREATE POLICY budget_profiles_service_role_all ON budget_profiles
    FOR ALL
    USING (public.app_is_service_role())
    WITH CHECK (public.app_is_service_role());

DROP POLICY IF EXISTS expenses_service_role_all ON expenses;
CREATE POLICY expenses_service_role_all ON expenses
    FOR ALL
    USING (public.app_is_service_role())
    WITH CHECK (public.app_is_service_role());

DROP POLICY IF EXISTS ai_plan_requests_service_role_all ON ai_plan_requests;
CREATE POLICY ai_plan_requests_service_role_all ON ai_plan_requests
    FOR ALL
    USING (public.app_is_service_role())
    WITH CHECK (public.app_is_service_role());

DROP POLICY IF EXISTS ai_plan_drafts_service_role_all ON ai_plan_drafts;
CREATE POLICY ai_plan_drafts_service_role_all ON ai_plan_drafts
    FOR ALL
    USING (public.app_is_service_role())
    WITH CHECK (public.app_is_service_role());

DROP POLICY IF EXISTS ai_plan_validation_results_service_role_all ON ai_plan_validation_results;
CREATE POLICY ai_plan_validation_results_service_role_all ON ai_plan_validation_results
    FOR ALL
    USING (public.app_is_service_role())
    WITH CHECK (public.app_is_service_role());

DROP POLICY IF EXISTS notifications_service_role_all ON notifications;
CREATE POLICY notifications_service_role_all ON notifications
    FOR ALL
    USING (public.app_is_service_role())
    WITH CHECK (public.app_is_service_role());

DROP POLICY IF EXISTS share_links_service_role_all ON share_links;
CREATE POLICY share_links_service_role_all ON share_links
    FOR ALL
    USING (public.app_is_service_role())
    WITH CHECK (public.app_is_service_role());

DROP POLICY IF EXISTS outbox_events_service_role_all ON outbox_events;
CREATE POLICY outbox_events_service_role_all ON outbox_events
    FOR ALL
    USING (public.app_is_service_role())
    WITH CHECK (public.app_is_service_role());

DROP POLICY IF EXISTS audit_logs_service_role_all ON audit_logs;
CREATE POLICY audit_logs_service_role_all ON audit_logs
    FOR ALL
    USING (public.app_is_service_role())
    WITH CHECK (public.app_is_service_role());

DROP POLICY IF EXISTS fcm_tokens_service_role_all ON fcm_tokens;
CREATE POLICY fcm_tokens_service_role_all ON fcm_tokens
    FOR ALL
    USING (public.app_is_service_role())
    WITH CHECK (public.app_is_service_role());

DROP POLICY IF EXISTS trips_select_member ON trips;
CREATE POLICY trips_select_member ON trips
    FOR SELECT
    USING (
        owner_user_id = public.app_uid()
        OR EXISTS (
            SELECT 1
            FROM trip_memberships tm
            WHERE tm.trip_id = trips.id
              AND tm.user_id = public.app_uid()
              AND tm.status = 'active'
        )
    );

DROP POLICY IF EXISTS expenses_select_trip_member ON expenses;
CREATE POLICY expenses_select_trip_member ON expenses
    FOR SELECT
    USING (
        EXISTS (
            SELECT 1
            FROM trip_memberships tm
            WHERE tm.trip_id = expenses.trip_id
              AND tm.user_id = public.app_uid()
              AND tm.status = 'active'
        )
    );

DROP POLICY IF EXISTS notifications_select_self ON notifications;
CREATE POLICY notifications_select_self ON notifications
    FOR SELECT
    USING (user_id = public.app_uid());

DROP POLICY IF EXISTS user_preferences_select_self ON user_preferences;
CREATE POLICY user_preferences_select_self ON user_preferences
    FOR SELECT
    USING (user_id = public.app_uid());

DROP POLICY IF EXISTS trip_memberships_select_self ON trip_memberships;
CREATE POLICY trip_memberships_select_self ON trip_memberships
    FOR SELECT
    USING (user_id = public.app_uid());
