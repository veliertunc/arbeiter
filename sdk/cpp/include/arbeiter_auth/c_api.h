#ifndef ARBEITER_AUTH_C_API_H
#define ARBEITER_AUTH_C_API_H

#ifdef __cplusplus
extern "C" {
#endif

typedef struct arbeiter_auth_client arbeiter_auth_client;

arbeiter_auth_client *arbeiter_auth_client_new(const char *base_url);
void arbeiter_auth_client_free(arbeiter_auth_client *client);
const char *arbeiter_auth_client_error(const arbeiter_auth_client *client);
void arbeiter_auth_string_free(char *value);

char *arbeiter_auth_sign_up(arbeiter_auth_client *, const char *email, const char *password, const char *name);
char *arbeiter_auth_sign_in(arbeiter_auth_client *, const char *email, const char *password);
char *arbeiter_auth_sign_out(arbeiter_auth_client *);
char *arbeiter_auth_me(arbeiter_auth_client *);
char *arbeiter_auth_update_me(arbeiter_auth_client *, const char *name);
char *arbeiter_auth_delete_account(arbeiter_auth_client *, const char *password);
char *arbeiter_auth_list_sessions(arbeiter_auth_client *);
char *arbeiter_auth_revoke_session(arbeiter_auth_client *, const char *session_id);
char *arbeiter_auth_request_email_verification(arbeiter_auth_client *);
char *arbeiter_auth_verify_email(arbeiter_auth_client *, const char *token);
char *arbeiter_auth_request_password_reset(arbeiter_auth_client *, const char *email);
char *arbeiter_auth_reset_password(arbeiter_auth_client *, const char *token, const char *password);
char *arbeiter_auth_list_organizations(arbeiter_auth_client *);
char *arbeiter_auth_create_organization(arbeiter_auth_client *, const char *name, const char *slug);
char *arbeiter_auth_delete_organization(arbeiter_auth_client *, const char *id);
char *arbeiter_auth_list_members(arbeiter_auth_client *, const char *organization_id);
char *arbeiter_auth_upsert_member(arbeiter_auth_client *, const char *organization_id, const char *user_id, const char *role);
char *arbeiter_auth_delete_member(arbeiter_auth_client *, const char *organization_id, const char *user_id);

#ifdef __cplusplus
}
#endif
#endif
