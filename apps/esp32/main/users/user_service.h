#ifndef ARBEITER_USER_SERVICE_H
#define ARBEITER_USER_SERVICE_H

#include "user_repository.h"

#define USER_SERVICE_PAGE_SIZE 5U

esp_err_t user_service_list(size_t page, user_record_t *records,
                            size_t *count, size_t *total);
esp_err_t user_service_get(int64_t id, user_record_t *record);
esp_err_t user_service_create(const char *name, const char *email, const char *role);
esp_err_t user_service_update(int64_t id, const char *name,
                              const char *email, const char *role);
esp_err_t user_service_delete(int64_t id);

#endif
