#include "arbeiter_auth/c_api.h"

#include <assert.h>
#include <string.h>

int c_api_smoke(void) {
    arbeiter_auth_client *client = arbeiter_auth_client_new("not-a-valid-url");
    assert(client != 0);
    char *response = arbeiter_auth_sign_in(client, "user@example.test", "password");
    assert(response == 0);
    assert(strlen(arbeiter_auth_client_error(client)) > 0);
    arbeiter_auth_client_free(client);
    return 0;
}
