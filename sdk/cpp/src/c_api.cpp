#include "arbeiter_auth/c_api.h"
#include "arbeiter_auth/client.hpp"

#include <cstdlib>
#include <cstring>
#include <functional>
#include <memory>
#include <string>

struct arbeiter_auth_client {
    explicit arbeiter_auth_client(const char* base) : sdk(base ? base : "") {}
    arbeiter::auth::AuthClient sdk;
    std::string error;
};

namespace {
char* copy_string(const std::string& value) {
    auto* result = static_cast<char*>(std::malloc(value.size() + 1));
    if (!result) return nullptr;
    std::memcpy(result, value.c_str(), value.size() + 1);
    return result;
}
char* invoke(arbeiter_auth_client* client, const std::function<nlohmann::json()>& operation) {
    if (!client) return nullptr;
    try { auto result = operation(); client->error.clear(); return copy_string(result.dump()); }
    catch (const std::exception& error) { client->error = error.what(); return nullptr; }
}
}

extern "C" {
arbeiter_auth_client* arbeiter_auth_client_new(const char* base_url) {
    if (!base_url) return nullptr;
    try { return new arbeiter_auth_client(base_url); } catch (...) { return nullptr; }
}
void arbeiter_auth_client_free(arbeiter_auth_client* client) { delete client; }
const char* arbeiter_auth_client_error(const arbeiter_auth_client* client) { return client ? client->error.c_str() : "client is null"; }
void arbeiter_auth_string_free(char* value) { std::free(value); }
char* arbeiter_auth_sign_up(arbeiter_auth_client* c,const char* e,const char* p,const char* n){return invoke(c,[&]{return c->sdk.sign_up(e?e:"",p?p:"",n?n:"");});}
char* arbeiter_auth_sign_in(arbeiter_auth_client* c,const char* e,const char* p){return invoke(c,[&]{return c->sdk.sign_in(e?e:"",p?p:"");});}
char* arbeiter_auth_sign_out(arbeiter_auth_client* c){return invoke(c,[&]{return c->sdk.sign_out();});}
char* arbeiter_auth_me(arbeiter_auth_client* c){return invoke(c,[&]{return c->sdk.me();});}
char* arbeiter_auth_update_me(arbeiter_auth_client* c,const char* n){return invoke(c,[&]{return c->sdk.update_me(n?n:"");});}
char* arbeiter_auth_delete_account(arbeiter_auth_client* c,const char* p){return invoke(c,[&]{return c->sdk.delete_account(p?p:"");});}
char* arbeiter_auth_list_sessions(arbeiter_auth_client* c){return invoke(c,[&]{return c->sdk.list_sessions();});}
char* arbeiter_auth_revoke_session(arbeiter_auth_client* c,const char* id){return invoke(c,[&]{return c->sdk.revoke_session(id?id:"");});}
char* arbeiter_auth_request_email_verification(arbeiter_auth_client* c){return invoke(c,[&]{return c->sdk.request_email_verification();});}
char* arbeiter_auth_verify_email(arbeiter_auth_client* c,const char* token){return invoke(c,[&]{return c->sdk.verify_email(token?token:"");});}
char* arbeiter_auth_request_password_reset(arbeiter_auth_client* c,const char* email){return invoke(c,[&]{return c->sdk.request_password_reset(email?email:"");});}
char* arbeiter_auth_reset_password(arbeiter_auth_client* c,const char* token,const char* password){return invoke(c,[&]{return c->sdk.reset_password(token?token:"",password?password:"");});}
char* arbeiter_auth_list_organizations(arbeiter_auth_client* c){return invoke(c,[&]{return c->sdk.list_organizations();});}
char* arbeiter_auth_create_organization(arbeiter_auth_client* c,const char* n,const char* s){return invoke(c,[&]{return c->sdk.create_organization(n?n:"",s?s:"");});}
char* arbeiter_auth_delete_organization(arbeiter_auth_client* c,const char* id){return invoke(c,[&]{return c->sdk.delete_organization(id?id:"");});}
char* arbeiter_auth_list_members(arbeiter_auth_client* c,const char* org){return invoke(c,[&]{return c->sdk.list_members(org?org:"");});}
char* arbeiter_auth_upsert_member(arbeiter_auth_client* c,const char* org,const char* user,const char* role){return invoke(c,[&]{return c->sdk.upsert_member(org?org:"",user?user:"",role?role:"");});}
char* arbeiter_auth_delete_member(arbeiter_auth_client* c,const char* org,const char* user){return invoke(c,[&]{return c->sdk.delete_member(org?org:"",user?user:"");});}
}
