#include "user_service.h"

#include <ctype.h>
#include <string.h>

#include "esp_check.h"
static esp_err_t copy_trimmed(char *destination, size_t capacity, const char *source)
{
    if (!source || !destination || capacity == 0U) return ESP_ERR_INVALID_ARG;
    while (isspace((unsigned char)*source)) ++source;
    size_t length = strlen(source);
    while (length > 0U && isspace((unsigned char)source[length - 1U])) --length;
    if (length == 0U || length >= capacity) return ESP_ERR_INVALID_ARG;
    memcpy(destination, source, length);
    destination[length] = '\0';
    return ESP_OK;
}

static esp_err_t make_record(user_record_t *record, const char *name,
                             const char *email, const char *role)
{
    memset(record, 0, sizeof(*record));
    ESP_RETURN_ON_ERROR(copy_trimmed(record->name, sizeof(record->name), name),
                        "user_service", "Name is required (max 80 characters)");
    ESP_RETURN_ON_ERROR(copy_trimmed(record->email, sizeof(record->email), email),
                        "user_service", "Email is required (max 254 characters)");
    ESP_RETURN_ON_ERROR(copy_trimmed(record->role, sizeof(record->role), role),
                        "user_service", "Role is required");

    char *at = strchr(record->email, '@');
    char *dot = at ? strrchr(at + 1, '.') : NULL;
    if (!at || at == record->email || strchr(at + 1, '@') || !dot ||
        dot == at + 1 || dot[1] == '\0') return ESP_ERR_INVALID_ARG;
    for (const unsigned char *p = (const unsigned char *)record->email; *p; ++p)
        if (isspace(*p)) return ESP_ERR_INVALID_ARG;
    if (strcmp(record->role, "member") != 0 && strcmp(record->role, "admin") != 0 &&
        strcmp(record->role, "owner") != 0) return ESP_ERR_INVALID_ARG;
    return ESP_OK;
}

esp_err_t user_service_list(size_t page, user_record_t *records,
                            size_t *count, size_t *total)
{
    return user_repository_list(page, USER_SERVICE_PAGE_SIZE, records,
                                USER_SERVICE_PAGE_SIZE, count, total);
}

esp_err_t user_service_get(int64_t id, user_record_t *record)
{
    return user_repository_get(id, record);
}

esp_err_t user_service_create(const char *name, const char *email, const char *role)
{
    user_record_t record;
    esp_err_t err = make_record(&record, name, email, role);
    if (err != ESP_OK) return err;
    return user_repository_create(&record);
}

esp_err_t user_service_update(int64_t id, const char *name,
                              const char *email, const char *role)
{
    if (id <= 0) return ESP_ERR_INVALID_ARG;
    user_record_t record;
    esp_err_t err = make_record(&record, name, email, role);
    if (err != ESP_OK) return err;
    record.id = id;
    return user_repository_update(&record);
}

esp_err_t user_service_delete(int64_t id)
{
    return user_repository_delete(id);
}
