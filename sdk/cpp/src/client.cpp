#include "arbeiter_auth/client.hpp"

#include <curl/curl.h>

#include <algorithm>
#include <cctype>
#include <mutex>
#include <sstream>

namespace arbeiter::auth {
namespace {

class CurlTransport final : public Transport {
public:
    CurlTransport() {
        static std::once_flag initialized;
        std::call_once(initialized, [] { if (curl_global_init(CURL_GLOBAL_DEFAULT) != CURLE_OK) throw std::runtime_error("curl initialization failed"); });
    }

    HttpResponse send(const std::string& method, const std::string& url, const std::string& body, const std::string& cookie) override {
        CURL* handle = curl_easy_init();
        if (!handle) throw std::runtime_error("could not create HTTP request");
        HttpResponse response;
        curl_slist* headers = nullptr;
        headers = curl_slist_append(headers, "Accept: application/json");
        if (!body.empty()) headers = curl_slist_append(headers, "Content-Type: application/json");
        if (!cookie.empty()) headers = curl_slist_append(headers, ("Cookie: " + cookie).c_str());
        curl_easy_setopt(handle, CURLOPT_URL, url.c_str());
        curl_easy_setopt(handle, CURLOPT_CUSTOMREQUEST, method.c_str());
        curl_easy_setopt(handle, CURLOPT_HTTPHEADER, headers);
        curl_easy_setopt(handle, CURLOPT_FOLLOWLOCATION, 0L);
        curl_easy_setopt(handle, CURLOPT_CONNECTTIMEOUT, 10L);
        curl_easy_setopt(handle, CURLOPT_TIMEOUT, 20L);
        curl_easy_setopt(handle, CURLOPT_SSL_VERIFYPEER, 1L);
        curl_easy_setopt(handle, CURLOPT_SSL_VERIFYHOST, 2L);
        curl_easy_setopt(handle, CURLOPT_WRITEFUNCTION, +[](char* data, size_t size, size_t count, void* context) -> size_t {
            auto* output = static_cast<std::string*>(context); output->append(data, size * count); return size * count;
        });
        curl_easy_setopt(handle, CURLOPT_WRITEDATA, &response.body);
        curl_easy_setopt(handle, CURLOPT_HEADERFUNCTION, +[](char* data, size_t size, size_t count, void* context) -> size_t {
            auto* output = static_cast<HttpResponse*>(context); const size_t bytes = size * count;
            std::string line(data, bytes); const auto colon = line.find(':');
            if (colon != std::string::npos) {
                std::string key = line.substr(0, colon); std::transform(key.begin(), key.end(), key.begin(), [](unsigned char c){return static_cast<char>(std::tolower(c));});
                auto value = line.substr(colon + 1); const auto first = value.find_first_not_of(" \t\r\n"); const auto last = value.find_last_not_of(" \t\r\n");
                value = first == std::string::npos ? "" : value.substr(first, last - first + 1);
                output->headers.emplace_back(std::move(key), std::move(value));
            }
            return bytes;
        });
        curl_easy_setopt(handle, CURLOPT_HEADERDATA, &response);
        if (!body.empty()) curl_easy_setopt(handle, CURLOPT_POSTFIELDS, body.c_str());
        const CURLcode result = curl_easy_perform(handle);
        if (result != CURLE_OK) {
            const std::string error = curl_easy_strerror(result); curl_slist_free_all(headers); curl_easy_cleanup(handle); throw std::runtime_error(error);
        }
        curl_easy_getinfo(handle, CURLINFO_RESPONSE_CODE, &response.status);
        curl_slist_free_all(headers); curl_easy_cleanup(handle);
        return response;
    }
};

std::string encode_segment(const std::string& value) {
    std::ostringstream out;
    for (unsigned char byte : value) {
        if (std::isalnum(byte) || byte == '-' || byte == '_' || byte == '.' || byte == '~') out << byte;
        else { const char* hex = "0123456789ABCDEF"; out << '%' << hex[byte >> 4] << hex[byte & 15]; }
    }
    return out.str();
}

} // namespace

AuthError::AuthError(long status, std::string message) : std::runtime_error(std::move(message)), status_(status) {}

AuthClient::AuthClient(std::string base_url) : AuthClient(std::move(base_url), std::make_shared<CurlTransport>()) {}
AuthClient::AuthClient(std::string base_url, std::shared_ptr<Transport> transport)
    : base_url_(std::move(base_url)), transport_(std::move(transport)) {
    while (!base_url_.empty() && base_url_.back() == '/') base_url_.pop_back();
    if (base_url_.empty() || !transport_) throw std::invalid_argument("base URL and transport are required");
}

nlohmann::json AuthClient::request(const std::string& method, const std::string& path, const nlohmann::json* body) {
    const auto response = transport_->send(method, base_url_ + path, body ? body->dump() : "", session_cookie_);
    for (const auto& [key, value] : response.headers) {
        std::string lowered = key; std::transform(lowered.begin(), lowered.end(), lowered.begin(), [](unsigned char c){return static_cast<char>(std::tolower(c));});
        if (lowered == "set-cookie" && value.rfind("session=", 0) == 0) session_cookie_ = value.substr(0, value.find(';'));
    }
    if (response.status == 204) return nullptr;
    nlohmann::json payload = nullptr;
    try { if (!response.body.empty()) payload = nlohmann::json::parse(response.body); }
    catch (...) { payload = nlohmann::json{{"message", response.body}}; }
    if (response.status < 200 || response.status >= 300) {
        const auto message = payload.is_object() ? payload.value("message", std::string("request failed")) : std::string("request failed");
        throw AuthError(response.status, message);
    }
    return payload;
}

nlohmann::json AuthClient::sign_up(const std::string& email,const std::string& password,const std::string& name){auto b=nlohmann::json{{"email",email},{"password",password}};if(!name.empty())b["name"]=name;return request("POST","/v1/sign-up",&b);}
nlohmann::json AuthClient::sign_in(const std::string& email,const std::string& password){auto b=nlohmann::json{{"email",email},{"password",password}};return request("POST","/v1/sign-in",&b);}
nlohmann::json AuthClient::sign_out(){auto b=nlohmann::json::object();return request("POST","/v1/sign-out",&b);}
nlohmann::json AuthClient::me(){return request("GET","/v1/me");}
nlohmann::json AuthClient::update_me(const std::string& name){auto b=nlohmann::json{{"name",name}};return request("PATCH","/v1/me",&b);}
nlohmann::json AuthClient::delete_account(const std::string& password){auto b=nlohmann::json{{"password",password}};return request("DELETE","/v1/me",&b);}
nlohmann::json AuthClient::list_sessions(){return request("GET","/v1/sessions");}
nlohmann::json AuthClient::revoke_session(const std::string& id){return request("DELETE","/v1/sessions/"+encode_segment(id));}
nlohmann::json AuthClient::request_email_verification(){auto b=nlohmann::json::object();return request("POST","/v1/email/verification",&b);}
nlohmann::json AuthClient::verify_email(const std::string& token){auto b=nlohmann::json{{"token",token}};return request("POST","/v1/email/verification/consume",&b);}
nlohmann::json AuthClient::request_password_reset(const std::string& email){auto b=nlohmann::json{{"email",email}};return request("POST","/v1/password/reset",&b);}
nlohmann::json AuthClient::reset_password(const std::string& token,const std::string& password){auto b=nlohmann::json{{"token",token},{"password",password}};return request("POST","/v1/password/reset/consume",&b);}
nlohmann::json AuthClient::list_organizations(){return request("GET","/v1/organizations");}
nlohmann::json AuthClient::create_organization(const std::string& name,const std::string& slug){auto b=nlohmann::json{{"name",name},{"slug",slug}};return request("POST","/v1/organizations",&b);}
nlohmann::json AuthClient::delete_organization(const std::string& id){return request("DELETE","/v1/organizations/"+encode_segment(id));}
nlohmann::json AuthClient::list_members(const std::string& id){return request("GET","/v1/organizations/"+encode_segment(id)+"/members");}
nlohmann::json AuthClient::upsert_member(const std::string& org,const std::string& user,const std::string& role){auto b=nlohmann::json{{"role",role}};return request("PUT","/v1/organizations/"+encode_segment(org)+"/members/"+encode_segment(user),&b);}
nlohmann::json AuthClient::delete_member(const std::string& org,const std::string& user){return request("DELETE","/v1/organizations/"+encode_segment(org)+"/members/"+encode_segment(user));}

} // namespace arbeiter::auth
