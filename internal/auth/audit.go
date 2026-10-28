package auth

import "context"

func (a *App) audit(ctx context.Context, actor, subject, org, event string) {
	_, err := a.db.ExecContext(ctx, `INSERT INTO audit_events(id,actor_user_id,subject_user_id,org_id,event_type) VALUES($1,NULLIF($2,'')::uuid,NULLIF($3,'')::uuid,NULLIF($4,'')::uuid,$5)`, newID(), actor, subject, org, event)
	if err != nil {
		a.log.Error("audit event write failed", "event", event, "error", err)
	}
}
