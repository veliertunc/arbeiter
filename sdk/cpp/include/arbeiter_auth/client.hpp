#pragma once

#include <memory>
#include <stdexcept>
#include <string>
#include <utility>
#include <vector>

#include <nlohmann/json.hpp>

namespace arbeiter::auth {

struct HttpResponse {
    long status{};
    std::string body;
    std::vector<std::pair<std::string, std::string>> headers;
};

class Transport {
public:
    virtual ~Transport() = default;
    virtual HttpResponse send(const std::string& method, const std::string& url,
                              const std::string& body, const std::string& cookie) = 0;
};

class AuthError : public std::runtime_error {
public:
    AuthError(long status, std::string message);
    long status() const noexcept { return status_; }
private:
    long status_;
};

class AuthClient {
public:
    explicit AuthClient(std::string base_url);
    AuthClient(std::string base_url, std::shared_ptr<Transport> transport);

    nlohmann::json sign_up(const std::string& email, const std::string& password, const std::string& name = {});
    nlohmann::json sign_in(const std::string& email, const std::string& password);
    nlohmann::json sign_out();
    nlohmann::json me();
    nlohmann::json update_me(const std::string& name);
    nlohmann::json delete_account(const std::string& password);
    nlohmann::json list_sessions();
    nlohmann::json revoke_session(const std::string& id);
    nlohmann::json request_email_verification();
    nlohmann::json verify_email(const std::string& token);
    nlohmann::json request_password_reset(const std::string& email);
    nlohmann::json reset_password(const std::string& token, const std::string& password);
    nlohmann::json list_organizations();
    nlohmann::json create_organization(const std::string& name, const std::string& slug);
    nlohmann::json delete_organization(const std::string& id);
    nlohmann::json list_members(const std::string& organization_id);
    nlohmann::json upsert_member(const std::string& organization_id, const std::string& user_id, const std::string& role);
    nlohmann::json delete_member(const std::string& organization_id, const std::string& user_id);

private:
    nlohmann::json request(const std::string& method, const std::string& path, const nlohmann::json* body = nullptr);
    std::string base_url_;
    std::shared_ptr<Transport> transport_;
    std::string session_cookie_;
};

} // namespace arbeiter::auth
