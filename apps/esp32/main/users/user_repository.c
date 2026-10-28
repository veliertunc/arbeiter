#include "user_repository.h"

#include <stdio.h>
#include <stdbool.h>
#include <string.h>

#include "esp_check.h"
#include "esp_log.h"
#include "esp_sqlite3.h"
#include "freertos/FreeRTOS.h"
#include "freertos/semphr.h"
#include "sqlite3.h"

static const char *TAG = "user_repo";
static esp_sqlite3_t *s_database;
static sqlite3 *s_sqlite;
static SemaphoreHandle_t s_mutex;

static esp_err_t prepare(const char *sql, sqlite3_stmt **statement)
{
    return sqlite3_prepare_v2(s_sqlite, sql, -1, statement, NULL) == SQLITE_OK
               ? ESP_OK : ESP_FAIL;
}

static esp_err_t bind_record(sqlite3_stmt *statement, const user_record_t *record, bool with_id)
{
    int index = 1;
    if (with_id && sqlite3_bind_int64(statement, index++, record->id) != SQLITE_OK) return ESP_FAIL;
    if (sqlite3_bind_text(statement, index++, record->name, -1, SQLITE_TRANSIENT) != SQLITE_OK ||
        sqlite3_bind_text(statement, index++, record->email, -1, SQLITE_TRANSIENT) != SQLITE_OK ||
        sqlite3_bind_text(statement, index, record->role, -1, SQLITE_TRANSIENT) != SQLITE_OK) {
        return ESP_FAIL;
    }
    return ESP_OK;
}

static void copy_column(char *destination, size_t capacity, sqlite3_stmt *statement, int column)
{
    const unsigned char *value = sqlite3_column_text(statement, column);
    if (!value) {
        destination[0] = '\0';
        return;
    }
    (void)snprintf(destination, capacity, "%s", (const char *)value);
}

static esp_err_t statement_result(int sqlite_result)
{
    if (sqlite_result == SQLITE_DONE || sqlite_result == SQLITE_ROW) return ESP_OK;
    if ((sqlite_result & 0xff) == SQLITE_CONSTRAINT) return ESP_ERR_INVALID_STATE;
    ESP_LOGE(TAG, "SQLite operation failed (%d)", sqlite_result);
    return ESP_FAIL;
}

esp_err_t user_repository_open(const char *database_path)
{
    if (!database_path || database_path[0] != '/') return ESP_ERR_INVALID_ARG;
    if (s_database) return ESP_ERR_INVALID_STATE;
    ESP_RETURN_ON_ERROR(esp_sqlite3_init(), TAG, "SQLite initialization failed");

    esp_sqlite3_config_t config;
    esp_sqlite3_config_init(&config);
    config.path = database_path;
    config.cache_size_kib = 8;
    ESP_RETURN_ON_ERROR(esp_sqlite3_open(&config, &s_database), TAG, "Database open failed");
    s_sqlite = esp_sqlite3_handle(s_database);
    if (!s_sqlite) {
        user_repository_close();
        return ESP_FAIL;
    }

    /* Use rollback journaling and full sync on FAT; do not depend on FAT WAL locks. */
    esp_err_t err = esp_sqlite3_exec(s_database, "PRAGMA journal_mode=DELETE;");
    if (err == ESP_OK) err = esp_sqlite3_exec(s_database, "PRAGMA synchronous=FULL;");
    if (err == ESP_OK) {
        err = esp_sqlite3_exec(s_database,
            "CREATE TABLE IF NOT EXISTS users ("
            "id INTEGER PRIMARY KEY AUTOINCREMENT,"
            "name TEXT NOT NULL CHECK(length(name) BETWEEN 1 AND 80),"
            "email TEXT NOT NULL COLLATE NOCASE UNIQUE CHECK(length(email) BETWEEN 3 AND 254),"
            "role TEXT NOT NULL DEFAULT 'member' CHECK(role IN ('member','admin','owner')),"
            "created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))," 
            "updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))"
            ");");
    }
    if (err != ESP_OK) {
        user_repository_close();
        return err;
    }
    s_mutex = xSemaphoreCreateMutex();
    if (!s_mutex) {
        user_repository_close();
        return ESP_ERR_NO_MEM;
    }
    ESP_LOGI(TAG, "SQLite user database ready at %s", database_path);
    return ESP_OK;
}

void user_repository_close(void)
{
    if (s_mutex) {
        vSemaphoreDelete(s_mutex);
        s_mutex = NULL;
    }
    if (s_database) {
        esp_sqlite3_close(s_database);
        s_database = NULL;
        s_sqlite = NULL;
    }
}

esp_err_t user_repository_list(size_t page, size_t page_size,
                               user_record_t *records, size_t capacity,
                               size_t *count, size_t *total)
{
    if (!s_database || !records || !count || !total || page_size == 0U || page_size > capacity)
        return ESP_ERR_INVALID_ARG;
    *count = 0U;
    *total = 0U;
    if (xSemaphoreTake(s_mutex, portMAX_DELAY) != pdTRUE) return ESP_ERR_TIMEOUT;

    sqlite3_stmt *statement = NULL;
    esp_err_t err = prepare("SELECT COUNT(*) FROM users;", &statement);
    if (err == ESP_OK && sqlite3_step(statement) == SQLITE_ROW)
        *total = (size_t)sqlite3_column_int64(statement, 0);
    else if (err == ESP_OK) err = ESP_FAIL;
    sqlite3_finalize(statement);
    statement = NULL;

    if (err == ESP_OK) err = prepare(
        "SELECT id,name,email,role FROM users ORDER BY name COLLATE NOCASE,id LIMIT ?1 OFFSET ?2;",
        &statement);
    if (err == ESP_OK && (page > (size_t)INT64_MAX / page_size ||
        sqlite3_bind_int64(statement, 1, (sqlite3_int64)page_size) != SQLITE_OK ||
        sqlite3_bind_int64(statement, 2, (sqlite3_int64)(page * page_size)) != SQLITE_OK)) err = ESP_ERR_INVALID_ARG;
    int result = SQLITE_DONE;
    while (err == ESP_OK && (result = sqlite3_step(statement)) == SQLITE_ROW) {
        user_record_t *record = &records[*count];
        record->id = sqlite3_column_int64(statement, 0);
        copy_column(record->name, sizeof(record->name), statement, 1);
        copy_column(record->email, sizeof(record->email), statement, 2);
        copy_column(record->role, sizeof(record->role), statement, 3);
        ++*count;
    }
    if (err == ESP_OK && result != SQLITE_DONE) err = statement_result(result);
    sqlite3_finalize(statement);
    xSemaphoreGive(s_mutex);
    return err;
}

esp_err_t user_repository_get(int64_t id, user_record_t *record)
{
    if (!s_database || id <= 0 || !record) return ESP_ERR_INVALID_ARG;
    if (xSemaphoreTake(s_mutex, portMAX_DELAY) != pdTRUE) return ESP_ERR_TIMEOUT;
    sqlite3_stmt *statement = NULL;
    esp_err_t err = prepare("SELECT id,name,email,role FROM users WHERE id=?1;", &statement);
    if (err == ESP_OK && sqlite3_bind_int64(statement, 1, id) != SQLITE_OK) err = ESP_FAIL;
    int result = err == ESP_OK ? sqlite3_step(statement) : SQLITE_ERROR;
    if (result == SQLITE_ROW) {
        record->id = sqlite3_column_int64(statement, 0);
        copy_column(record->name, sizeof(record->name), statement, 1);
        copy_column(record->email, sizeof(record->email), statement, 2);
        copy_column(record->role, sizeof(record->role), statement, 3);
    } else if (err == ESP_OK) {
        err = result == SQLITE_DONE ? ESP_ERR_NOT_FOUND : statement_result(result);
    }
    sqlite3_finalize(statement);
    xSemaphoreGive(s_mutex);
    return err;
}

esp_err_t user_repository_create(const user_record_t *record)
{
    if (!s_database || !record) return ESP_ERR_INVALID_ARG;
    if (xSemaphoreTake(s_mutex, portMAX_DELAY) != pdTRUE) return ESP_ERR_TIMEOUT;
    sqlite3_stmt *statement = NULL;
    esp_err_t err = prepare("INSERT INTO users(name,email,role) VALUES(?1,?2,?3);", &statement);
    if (err == ESP_OK) err = bind_record(statement, record, false);
    if (err == ESP_OK) err = statement_result(sqlite3_step(statement));
    sqlite3_finalize(statement);
    xSemaphoreGive(s_mutex);
    return err;
}

esp_err_t user_repository_update(const user_record_t *record)
{
    if (!s_database || !record || record->id <= 0) return ESP_ERR_INVALID_ARG;
    if (xSemaphoreTake(s_mutex, portMAX_DELAY) != pdTRUE) return ESP_ERR_TIMEOUT;
    sqlite3_stmt *statement = NULL;
    esp_err_t err = prepare(
        "UPDATE users SET name=?1,email=?2,role=?3,updated_at=strftime('%Y-%m-%dT%H:%M:%fZ','now') WHERE id=?4;",
        &statement);
    if (err == ESP_OK) err = bind_record(statement, record, false);
    if (err == ESP_OK && sqlite3_bind_int64(statement, 4, record->id) != SQLITE_OK) err = ESP_FAIL;
    if (err == ESP_OK) err = statement_result(sqlite3_step(statement));
    if (err == ESP_OK && sqlite3_changes(s_sqlite) == 0) err = ESP_ERR_NOT_FOUND;
    sqlite3_finalize(statement);
    xSemaphoreGive(s_mutex);
    return err;
}

esp_err_t user_repository_delete(int64_t id)
{
    if (!s_database || id <= 0) return ESP_ERR_INVALID_ARG;
    if (xSemaphoreTake(s_mutex, portMAX_DELAY) != pdTRUE) return ESP_ERR_TIMEOUT;
    sqlite3_stmt *statement = NULL;
    esp_err_t err = prepare("DELETE FROM users WHERE id=?1;", &statement);
    if (err == ESP_OK && sqlite3_bind_int64(statement, 1, id) != SQLITE_OK) err = ESP_FAIL;
    if (err == ESP_OK) err = statement_result(sqlite3_step(statement));
    if (err == ESP_OK && sqlite3_changes(s_sqlite) == 0) err = ESP_ERR_NOT_FOUND;
    sqlite3_finalize(statement);
    xSemaphoreGive(s_mutex);
    return err;
}
