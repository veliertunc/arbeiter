package integration

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOrganizationCRUDAndAuthorization(t *testing.T) {
	withTestApp(t, func(t *testing.T, db *sql.DB, server *httptest.Server) {
		ownerID := signUp(t, server, "owner@example.test", "owner password is long enough")
		owner := signIn(t, server, "owner@example.test", "owner password is long enough")
		if res := call(t, server, http.MethodPost, "/v1/organizations", `{"name":"Bad","slug":"a--b"}`, owner); res.status != http.StatusBadRequest {
			t.Fatalf("invalid slug status=%d", res.status)
		}
		res := call(t, server, http.MethodPost, "/v1/organizations", `{"name":"Example Org","slug":"example-org"}`, owner)
		if res.status != http.StatusCreated {
			t.Fatalf("create organization status=%d %s", res.status, res.body)
		}
		orgID := object(t, res)["id"].(string)
		res = call(t, server, http.MethodGet, "/v1/organizations", "", owner)
		var orgs []map[string]any
		if err := json.Unmarshal(res.body, &orgs); err != nil {
			t.Fatal(err)
		}
		if res.status != http.StatusOK || len(orgs) != 1 {
			t.Fatalf("list organizations: %d %s", res.status, res.body)
		}
		memberID := signUp(t, server, "member@example.test", "member password is long enough")
		adminID := signUp(t, server, "admin@example.test", "admin password is long enough")
		memberPath := "/v1/organizations/" + orgID + "/members/" + memberID
		if res = call(t, server, http.MethodPut, memberPath, `{"role":"member"}`, owner); res.status != http.StatusOK {
			t.Fatalf("add member status=%d", res.status)
		}
		adminPath := "/v1/organizations/" + orgID + "/members/" + adminID
		if res = call(t, server, http.MethodPut, adminPath, `{"role":"admin"}`, owner); res.status != http.StatusOK {
			t.Fatalf("grant admin status=%d", res.status)
		}
		admin := signIn(t, server, "admin@example.test", "admin password is long enough")
		if res = call(t, server, http.MethodPut, memberPath, `{"role":"admin"}`, admin); res.status != http.StatusForbidden {
			t.Fatalf("admin granted admin role: %d", res.status)
		}
		if res = call(t, server, http.MethodDelete, "/v1/organizations/"+orgID+"/members/"+ownerID, "", admin); res.status != http.StatusNotFound {
			t.Fatalf("owner membership removed: %d", res.status)
		}
		res = call(t, server, http.MethodGet, "/v1/organizations/"+orgID+"/members", "", owner)
		var members []map[string]any
		if err := json.Unmarshal(res.body, &members); err != nil {
			t.Fatal(err)
		}
		if len(members) != 3 {
			t.Fatalf("member list count=%d", len(members))
		}
		if res = call(t, server, http.MethodDelete, "/v1/organizations/"+orgID, "", admin); res.status != http.StatusNotFound {
			t.Fatalf("non-owner deleted organization: %d", res.status)
		}
		if res = call(t, server, http.MethodDelete, memberPath, "", owner); res.status != http.StatusOK {
			t.Fatalf("remove member status=%d", res.status)
		}
		if res = call(t, server, http.MethodDelete, "/v1/organizations/"+orgID, "", owner); res.status != http.StatusNoContent {
			t.Fatalf("delete organization status=%d", res.status)
		}
		if res = call(t, server, http.MethodGet, "/v1/organizations", "", owner); res.status != http.StatusOK || string(res.body) != "[]\n" {
			t.Fatalf("organization not removed from owner list: %d %s", res.status, res.body)
		}
		if res = call(t, server, http.MethodDelete, "/v1/me", `{"password":"owner password is long enough"}`, owner); res.status != http.StatusNoContent {
			t.Fatalf("owner account could not be deleted after org deletion: %d %s", res.status, res.body)
		}
		var eventCount int
		if err := db.QueryRow(`SELECT count(*) FROM audit_events WHERE org_id=$1`, orgID).Scan(&eventCount); err != nil {
			t.Fatal(err)
		}
		if eventCount < 4 {
			t.Fatalf("organization audit events=%d", eventCount)
		}
	})
}
