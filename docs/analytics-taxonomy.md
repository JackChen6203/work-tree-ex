# Analytics Event Taxonomy

## Principles
- event names are centralized in `apps/web/src/lib/analytics.ts`
- components should not hardcode event names
- duplicate events within a short window are suppressed client-side
- context must be minimal and useful

## Implemented Events
- `auth.session.login_requested`
- `auth.session.login_failed`
- `auth.session.login_succeeded`
- `auth.session.logged_out`
- `trip_created`

## Recommended Next Events
- `sync.flush_requested`
- `sync.flush_succeeded`
- `sync.flush_conflicted`
- `notification_marked_read`
- `notification_deleted`
- `ai_plan_created`
- `ai_plan_adopted`
