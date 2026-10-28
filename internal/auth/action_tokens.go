package auth

import (
	"context"
	"fmt"
	"time"
)

func (a *App) issueActionToken(ctx context.Context, userID, purpose string) error {
	token, e := randomToken(32)
	if e != nil {
		return e
	}
	var email string
	if e = a.db.QueryRowContext(ctx, `SELECT email FROM users WHERE id=$1`, userID).Scan(&email); e != nil {
		return e
	}
	tx, e := a.db.BeginTx(ctx, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	_, e = tx.ExecContext(ctx, `UPDATE action_tokens SET consumed_at=now() WHERE user_id=$1 AND purpose=$2 AND consumed_at IS NULL`, userID, purpose)
	if e != nil {
		return e
	}
	_, e = tx.ExecContext(ctx, `INSERT INTO action_tokens(id,user_id,token_hash,purpose,expires_at) VALUES($1,$2,$3,$4,$5)`, newID(), userID, a.hash(token), purpose, time.Now().Add(30*time.Minute))
	if e != nil {
		return e
	}
	if e = tx.Commit(); e != nil {
		return e
	}
	if a.origin == "" {
		return fmt.Errorf("APP_ORIGIN is required for email actions")
	}
	link := a.origin + "/email/verification?token=" + token
	subject, body := "Verify your email", "Verify your email: "+link
	if purpose == "reset" {
		subject = "Reset your password"
		body = "Reset your password: " + a.origin + "/password/reset?token=" + token
	}
	return a.sendEmail(email, subject, body)
}

func (a *App) consumeActionToken(ctx context.Context, token, purpose, sqlUpdate string) bool {
	tx, e := a.db.BeginTx(ctx, nil)
	if e != nil {
		return false
	}
	defer tx.Rollback()
	var id string
	e = tx.QueryRowContext(ctx, `UPDATE action_tokens SET consumed_at=now() WHERE token_hash=$1 AND purpose=$2 AND consumed_at IS NULL AND expires_at>now() RETURNING user_id::text`, a.hash(token), purpose).Scan(&id)
	if e != nil {
		return false
	}
	if _, e = tx.ExecContext(ctx, sqlUpdate, id); e != nil {
		return false
	}
	return tx.Commit() == nil
}
