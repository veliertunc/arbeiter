#ifndef ARBEITER_USER_REPOSITORY_H
#define ARBEITER_USER_REPOSITORY_H

#include <stddef.h>
#include <stdint.h>
#include "esp_err.h"

typedef struct {
    int64_t id;
    char name[81];
    char email[255];
    char role[16];
} user_record_t;

esp_err_t user_repository_open(const char *database_path);
void user_repository_close(void);
esp_err_t user_repository_list(size_t page, size_t page_size,
                               user_record_t *records, size_t capacity,
                               size_t *count, size_t *total);
esp_err_t user_repository_get(int64_t id, user_record_t *record);
esp_err_t user_repository_create(const user_record_t *record);
esp_err_t user_repository_update(const user_record_t *record);
esp_err_t user_repository_delete(int64_t id);

#endif
