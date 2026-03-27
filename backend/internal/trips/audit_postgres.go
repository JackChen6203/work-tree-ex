package trips

import (
	"context"
	"encoding/json"
)

func writeTripAuditLog(ctx context.Context, action, resourceType, resourceID string, beforeState, afterState any) error {
	p := getCollaborationPool()
	if p == nil {
		return nil
	}

	var beforePayload any
	if beforeState != nil {
		raw, err := json.Marshal(beforeState)
		if err != nil {
			return err
		}
		beforePayload = raw
	}
	var afterPayload any
	if afterState != nil {
		raw, err := json.Marshal(afterState)
		if err != nil {
			return err
		}
		afterPayload = raw
	}

	_, err := p.Exec(ctx, `
		INSERT INTO audit_logs (
			actor_user_id, action, resource_type, resource_id, before_state, after_state, request_id, created_at
		)
		VALUES (
			NULL, $1, $2, $3, $4::jsonb, $5::jsonb, NULL, now()
		)
	`, action, resourceType, resourceID, beforePayload, afterPayload)
	return err
}
