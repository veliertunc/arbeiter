package auth

import (
	"context"
	"net/http"
	"strings"
)

type orgInput struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type membershipInput struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
}

func (a *App) deleteOrg(w http.ResponseWriter, r *http.Request) {
	u := current(r)
	org := r.PathValue("org")
	tx, err := a.db.BeginTx(r.Context(), nil)
	if err != nil {
		writeJSON(w, 500, message{"internal error"})
		return
	}
	defer tx.Rollback()
	res, err := tx.ExecContext(r.Context(), `UPDATE organizations SET deleted_at=now() WHERE id=$1 AND deleted_at IS NULL AND EXISTS(SELECT 1 FROM memberships WHERE org_id=$1 AND user_id=$2 AND role='owner')`, org, u.ID)
	if err != nil {
		writeJSON(w, 500, message{"internal error"})
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		writeJSON(w, 404, message{"organization not found"})
		return
	}
	if _, err = tx.ExecContext(r.Context(), `DELETE FROM memberships WHERE org_id=$1`, org); err != nil {
		writeJSON(w, 500, message{"internal error"})
		return
	}
	if err = tx.Commit(); err != nil {
		writeJSON(w, 500, message{"internal error"})
		return
	}
	a.audit(r.Context(), u.ID, "", org, "organization.deleted")
	w.WriteHeader(http.StatusNoContent)
}

func (a *App) listOrgs(w http.ResponseWriter, r *http.Request) {
	u := current(r)
	rows, e := a.db.QueryContext(r.Context(), `SELECT o.id::text,o.name,o.slug,m.role FROM organizations o JOIN memberships m ON m.org_id=o.id WHERE m.user_id=$1 AND o.deleted_at IS NULL ORDER BY o.created_at`, u.ID)
	if e != nil {
		writeJSON(w, 500, message{"internal error"})
		return
	}
	defer rows.Close()
	out := make([]map[string]string, 0)
	for rows.Next() {
		var id, name, slug, role string
		if rows.Scan(&id, &name, &slug, &role) != nil {
			continue
		}
		out = append(out, map[string]string{"id": id, "name": name, "slug": slug, "role": role})
	}
	writeJSON(w, 200, out)
}

func (a *App) createOrg(w http.ResponseWriter, r *http.Request) {
	var in orgInput
	if !decode(w, r, &in) {
		return
	}
	in.Name = strings.TrimSpace(in.Name)
	in.Slug = strings.ToLower(strings.TrimSpace(in.Slug))
	if len(in.Name) < 1 || len(in.Name) > 120 || !ValidOrganizationSlug(in.Slug) {
		writeJSON(w, 400, message{"name and a valid slug (lowercase letters, numbers, hyphens) are required"})
		return
	}
	id := newID()
	tx, e := a.db.BeginTx(r.Context(), nil)
	if e != nil {
		writeJSON(w, 500, message{"internal error"})
		return
	}
	defer tx.Rollback()
	if _, e = tx.ExecContext(r.Context(), `INSERT INTO organizations(id,name,slug) VALUES($1,$2,$3)`, id, in.Name, in.Slug); e != nil {
		writeJSON(w, 409, message{"organization could not be created"})
		return
	}
	if _, e = tx.ExecContext(r.Context(), `INSERT INTO memberships(org_id,user_id,role) VALUES($1,$2,'owner')`, id, current(r).ID); e != nil {
		writeJSON(w, 500, message{"internal error"})
		return
	}
	if e = tx.Commit(); e != nil {
		writeJSON(w, 500, message{"internal error"})
		return
	}
	a.audit(r.Context(), current(r).ID, "", id, "organization.created")
	writeJSON(w, 201, map[string]string{"id": id, "name": in.Name, "slug": in.Slug, "role": "owner"})
}

func (a *App) listMembers(w http.ResponseWriter, r *http.Request) {
	u := current(r)
	org := r.PathValue("org")
	if !a.isMember(r.Context(), org, u.ID) {
		writeJSON(w, 404, message{"organization not found"})
		return
	}
	rows, e := a.db.QueryContext(r.Context(), `SELECT u.id::text,u.email,u.name,m.role,u.email_verified FROM memberships m JOIN users u ON u.id=m.user_id WHERE m.org_id=$1 AND u.disabled_at IS NULL ORDER BY u.created_at`, org)
	if e != nil {
		writeJSON(w, 500, message{"internal error"})
		return
	}
	defer rows.Close()
	type member struct {
		ID       string `json:"id"`
		Email    string `json:"email"`
		Name     string `json:"name"`
		Role     string `json:"role"`
		Verified bool   `json:"email_verified"`
	}
	out := make([]member, 0)
	for rows.Next() {
		var x member
		if rows.Scan(&x.ID, &x.Email, &x.Name, &x.Role, &x.Verified) == nil {
			out = append(out, x)
		}
	}
	writeJSON(w, 200, out)
}

func (a *App) putMember(w http.ResponseWriter, r *http.Request) {
	actor := current(r)
	org, target := r.PathValue("org"), r.PathValue("user")
	var in membershipInput
	if !decode(w, r, &in) {
		return
	}
	if in.UserID != "" && in.UserID != target {
		writeJSON(w, 400, message{"user_id does not match path"})
		return
	}
	if in.Role != "admin" && in.Role != "member" {
		writeJSON(w, 400, message{"role must be admin or member"})
		return
	}
	var role string
	e := a.db.QueryRowContext(r.Context(), `SELECT role FROM memberships WHERE org_id=$1 AND user_id=$2`, org, actor.ID).Scan(&role)
	if e != nil || role != "owner" && role != "admin" {
		writeJSON(w, 403, message{"organization admin access required"})
		return
	}
	if role != "owner" && in.Role != "member" {
		writeJSON(w, 403, message{"only owners can grant admin role"})
		return
	}
	res, e := a.db.ExecContext(r.Context(), `INSERT INTO memberships(org_id,user_id,role) SELECT $1,u.id,$3 FROM users u WHERE u.id=$2 AND u.disabled_at IS NULL ON CONFLICT(org_id,user_id) DO UPDATE SET role=EXCLUDED.role WHERE memberships.role<>'owner'`, org, target, in.Role)
	if e != nil {
		writeJSON(w, 404, message{"user or organization not found"})
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		writeJSON(w, 409, message{"owner membership cannot be changed"})
		return
	}
	a.audit(r.Context(), actor.ID, target, org, "organization.member.upserted")
	writeJSON(w, 200, message{"member added or updated"})
}

func (a *App) deleteMember(w http.ResponseWriter, r *http.Request) {
	actor := current(r)
	org, target := r.PathValue("org"), r.PathValue("user")
	var role string
	e := a.db.QueryRowContext(r.Context(), `SELECT role FROM memberships WHERE org_id=$1 AND user_id=$2`, org, actor.ID).Scan(&role)
	if e != nil || role != "owner" && role != "admin" {
		writeJSON(w, 403, message{"organization admin access required"})
		return
	}
	if target == actor.ID {
		writeJSON(w, 400, message{"cannot remove yourself"})
		return
	}
	res, e := a.db.ExecContext(r.Context(), `DELETE FROM memberships WHERE org_id=$1 AND user_id=$2 AND role<>'owner'`, org, target)
	if e != nil {
		writeJSON(w, 500, message{"internal error"})
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		writeJSON(w, 404, message{"member not found or cannot be removed"})
		return
	}
	a.audit(r.Context(), actor.ID, target, org, "organization.member.removed")
	writeJSON(w, 200, message{"member removed"})
}

func (a *App) isMember(ctx context.Context, org, user string) bool {
	var yes bool
	e := a.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM memberships WHERE org_id=$1 AND user_id=$2)`, org, user).Scan(&yes)
	return e == nil && yes
}
