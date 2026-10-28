#include "arbeiter_auth/client.hpp"

#include <cassert>
#include <memory>

using namespace arbeiter::auth;
extern "C" int c_api_smoke(void);

class FakeTransport final : public Transport {
public:
    int calls = 0;
    std::string last_cookie;
    HttpResponse send(const std::string&, const std::string& url, const std::string&, const std::string& cookie) override {
        ++calls; last_cookie = cookie;
        const std::string sign_in_path = "/v1/sign-in";
        const std::string me_path = "/v1/me";
        if (url.size() >= sign_in_path.size() && url.compare(url.size()-sign_in_path.size(),sign_in_path.size(),sign_in_path)==0) return {200, R"({"id":"u1"})", {{"Set-Cookie","session=cpp-session; Path=/; HttpOnly"}}};
        if (url.size() >= me_path.size() && url.compare(url.size()-me_path.size(),me_path.size(),me_path)==0) return {200, R"({"email":"user@example.test"})", {}};
        return {401, R"({"message":"authentication required"})", {}};
    }
};

int main() {
    auto transport = std::make_shared<FakeTransport>();
    AuthClient client("https://auth.example.test/", transport);
    client.sign_in("user@example.test", "password");
    auto user = client.me();
    assert(user.at("email") == "user@example.test");
    assert(transport->last_cookie == "session=cpp-session");
    try { client.list_sessions(); assert(false); }
    catch (const AuthError& error) { assert(error.status() == 401); }
    assert(c_api_smoke() == 0);
    return 0;
}
