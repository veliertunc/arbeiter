#include "user_screen.h"

#include <stdint.h>
#include <stdbool.h>
#include <stdio.h>
#include <string.h>

#include "esp_check.h"
#include "esp_lvgl_port.h"
#include "lvgl.h"
#include "user_service.h"

#define SCREEN_WIDTH 240
#define COLOR_BG 0xF3F6FB
#define COLOR_INK 0x172033
#define COLOR_MUTED 0x667085
#define COLOR_ACCENT 0x2563EB
#define COLOR_CARD 0xFFFFFF

static lv_obj_t *s_home;
static lv_obj_t *s_list;
static lv_obj_t *s_count_label;
static lv_obj_t *s_page_label;
static lv_obj_t *s_notice_label;
static lv_obj_t *s_name_field;
static lv_obj_t *s_email_field;
static lv_obj_t *s_role_dropdown;
static lv_obj_t *s_keyboard;
static lv_obj_t *s_form_screen;
static lv_obj_t *s_form_notice;
static int64_t s_editing_id;
static int64_t s_pending_delete_id;
static size_t s_page;
static size_t s_total;

static lv_obj_t *make_button(lv_obj_t *parent, const char *text,
                             int x, int y, int width, int height,
                             lv_event_cb_t callback, void *user_data)
{
    lv_obj_t *button = lv_button_create(parent);
    lv_obj_set_pos(button, x, y);
    lv_obj_set_size(button, width, height);
    lv_obj_set_style_radius(button, 9, 0);
    lv_obj_set_style_bg_color(button, lv_color_hex(COLOR_ACCENT), 0);
    lv_obj_set_style_border_width(button, 0, 0);
    lv_obj_t *label = lv_label_create(button);
    lv_label_set_text(label, text);
    lv_obj_center(label);
    if (callback) lv_obj_add_event_cb(button, callback, LV_EVENT_CLICKED, user_data);
    return button;
}

static void set_notice(lv_obj_t *label, const char *message, bool error)
{
    lv_label_set_text(label, message);
    lv_obj_set_style_text_color(label, lv_color_hex(error ? 0xB42318 : 0x16803C), 0);
}

static void refresh_home(void);
static void open_form(const user_record_t *record);

static void start_add_user(lv_event_t *event)
{
    (void)event;
    open_form(NULL);
}

static void start_edit_user(lv_event_t *event)
{
    int64_t id = (int64_t)(intptr_t)lv_event_get_user_data(event);
    user_record_t record;
    esp_err_t err = user_service_get(id, &record);
    if (err != ESP_OK) {
        set_notice(s_notice_label, "Could not load that user", true);
        return;
    }
    open_form(&record);
}

static void confirm_delete(lv_event_t *event)
{
    lv_obj_t *dialog = lv_event_get_user_data(event);
    if (lv_event_get_code(event) == LV_EVENT_CLICKED) {
        esp_err_t err = user_service_delete(s_pending_delete_id);
        lv_obj_delete(dialog);
        if (err == ESP_OK) {
            if (s_total > 0U && s_page * USER_SERVICE_PAGE_SIZE >= s_total - 1U && s_page > 0U)
                --s_page;
            set_notice(s_notice_label, "User deleted", false);
            refresh_home();
        } else {
            set_notice(s_notice_label, "Delete failed", true);
        }
    }
}

static void cancel_delete(lv_event_t *event)
{
    lv_obj_delete(lv_event_get_user_data(event));
}

static void show_delete_confirmation(lv_event_t *event)
{
    s_pending_delete_id = (int64_t)(intptr_t)lv_event_get_user_data(event);
    lv_obj_t *dialog = lv_obj_create(s_home);
    lv_obj_set_size(dialog, 216, 112);
    lv_obj_align(dialog, LV_ALIGN_CENTER, 0, 0);
    lv_obj_set_style_bg_color(dialog, lv_color_hex(COLOR_CARD), 0);
    lv_obj_set_style_border_color(dialog, lv_color_hex(0xD0D5DD), 0);
    lv_obj_set_style_shadow_width(dialog, 18, 0);
    lv_obj_set_style_radius(dialog, 14, 0);
    lv_obj_clear_flag(dialog, LV_OBJ_FLAG_SCROLLABLE);

    lv_obj_t *label = lv_label_create(dialog);
    lv_label_set_text(label, "Delete this user?");
    lv_obj_align(label, LV_ALIGN_TOP_MID, 0, 8);
    make_button(dialog, "Cancel", 12, 62, 88, 34, cancel_delete, dialog);
    make_button(dialog, "Delete", 116, 62, 88, 34, confirm_delete, dialog);
}

static void change_page(lv_event_t *event)
{
    intptr_t delta = (intptr_t)lv_event_get_user_data(event);
    size_t page_count = (s_total + USER_SERVICE_PAGE_SIZE - 1U) / USER_SERVICE_PAGE_SIZE;
    if (delta < 0 && s_page > 0U) --s_page;
    else if (delta > 0 && s_page + 1U < page_count) ++s_page;
    refresh_home();
}

static void refresh_home(void)
{
    user_record_t users[USER_SERVICE_PAGE_SIZE];
    size_t count = 0U;
    size_t total = 0U;
    esp_err_t err = user_service_list(s_page, users, &count, &total);
    if (err != ESP_OK) {
        set_notice(s_notice_label, "Database read failed", true);
        return;
    }
    s_total = total;
    char count_text[40];
    (void)snprintf(count_text, sizeof(count_text), "%u users", (unsigned int)total);
    lv_label_set_text(s_count_label, count_text);

    size_t page_count = (total + USER_SERVICE_PAGE_SIZE - 1U) / USER_SERVICE_PAGE_SIZE;
    if (page_count == 0U) page_count = 1U;
    char page_text[32];
    (void)snprintf(page_text, sizeof(page_text), "%u / %u",
                   (unsigned int)(s_page + 1U), (unsigned int)page_count);
    lv_label_set_text(s_page_label, page_text);
    lv_obj_clean(s_list);

    if (count == 0U) {
        lv_obj_t *empty = lv_label_create(s_list);
        lv_label_set_text(empty, "No users yet.\nTap + Add to create one.");
        lv_obj_set_style_text_color(empty, lv_color_hex(COLOR_MUTED), 0);
        lv_obj_center(empty);
    }

    for (size_t i = 0U; i < count; ++i) {
        int y = (int)(i * 39U);
        lv_obj_t *row = lv_obj_create(s_list);
        lv_obj_set_pos(row, 2, y);
        lv_obj_set_size(row, 212, 37);
        lv_obj_set_style_bg_color(row, lv_color_hex(COLOR_CARD), 0);
        lv_obj_set_style_border_width(row, 0, 0);
        lv_obj_set_style_radius(row, 8, 0);
        lv_obj_set_style_pad_all(row, 2, 0);
        lv_obj_clear_flag(row, LV_OBJ_FLAG_SCROLLABLE);

        char summary[220];
        (void)snprintf(summary, sizeof(summary), "%s\n%s", users[i].name, users[i].email);
        lv_obj_t *label = lv_label_create(row);
        lv_label_set_text(label, summary);
        lv_obj_set_pos(label, 4, 1);
        lv_obj_set_size(label, 112, 34);
        lv_obj_set_style_text_color(label, lv_color_hex(COLOR_INK), 0);
        lv_obj_set_style_text_font(label, &lv_font_montserrat_12, 0);
        lv_label_set_long_mode(label, LV_LABEL_LONG_DOT);

        int64_t id = users[i].id;
        make_button(row, "Edit", 120, 2, 42, 31, start_edit_user, (void *)(intptr_t)id);
        lv_obj_t *delete_button = make_button(row, "Del", 166, 2, 42, 31,
                                              show_delete_confirmation, (void *)(intptr_t)id);
        lv_obj_set_style_bg_color(delete_button, lv_color_hex(0xD92D20), 0);
    }
}

static void keyboard_focus(lv_event_t *event)
{
    lv_obj_t *field = lv_event_get_target(event);
    lv_keyboard_set_textarea(s_keyboard, field);
    lv_obj_clear_flag(s_keyboard, LV_OBJ_FLAG_HIDDEN);
}

static void keyboard_ready(lv_event_t *event)
{
    (void)event;
    lv_obj_add_flag(s_keyboard, LV_OBJ_FLAG_HIDDEN);
}

static void form_save(lv_event_t *event)
{
    (void)event;
    const char *name = lv_textarea_get_text(s_name_field);
    const char *email = lv_textarea_get_text(s_email_field);
    const char *role = "member";
    switch (lv_dropdown_get_selected(s_role_dropdown)) {
    case 1: role = "admin"; break;
    case 2: role = "owner"; break;
    default: break;
    }

    esp_err_t err = s_editing_id == 0
        ? user_service_create(name, email, role)
        : user_service_update(s_editing_id, name, email, role);
    if (err == ESP_ERR_INVALID_STATE) {
        set_notice(s_form_notice, "That email is already in use", true);
        return;
    }
    if (err == ESP_ERR_INVALID_ARG) {
        set_notice(s_form_notice, "Check name, email, and role", true);
        return;
    }
    if (err != ESP_OK) {
        set_notice(s_form_notice, "Could not save user", true);
        return;
    }

    if (s_form_screen) {
        lv_obj_t *old_form = s_form_screen;
        s_form_screen = NULL;
        lv_screen_load(s_home);
        lv_obj_delete(old_form);
    }
    s_page = 0U;
    set_notice(s_notice_label, s_editing_id == 0 ? "User created" : "User updated", false);
    refresh_home();
}

static void form_cancel(lv_event_t *event)
{
    (void)event;
    lv_screen_load(s_home);
    if (s_form_screen) {
        lv_obj_delete(s_form_screen);
        s_form_screen = NULL;
    }
}

static void open_form(const user_record_t *record)
{
    s_editing_id = record ? record->id : 0;
    s_form_screen = lv_obj_create(NULL);
    lv_obj_set_style_bg_color(s_form_screen, lv_color_hex(COLOR_BG), 0);
    lv_obj_clear_flag(s_form_screen, LV_OBJ_FLAG_SCROLLABLE);

    lv_obj_t *back = make_button(s_form_screen, "Back", 8, 8, 56, 32, form_cancel, NULL);
    lv_obj_set_style_bg_color(back, lv_color_hex(0x475467), 0);
    lv_obj_t *title = lv_label_create(s_form_screen);
    lv_label_set_text(title, record ? "Edit user" : "New user");
    lv_obj_align(title, LV_ALIGN_TOP_MID, 0, 17);
    lv_obj_set_style_text_color(title, lv_color_hex(COLOR_INK), 0);
    lv_obj_set_style_text_font(title, &lv_font_montserrat_16, 0);

    s_name_field = lv_textarea_create(s_form_screen);
    lv_obj_set_pos(s_name_field, 8, 55);
    lv_obj_set_size(s_name_field, 224, 36);
    lv_textarea_set_one_line(s_name_field, true);
    lv_textarea_set_placeholder_text(s_name_field, "Full name");
    lv_textarea_set_max_length(s_name_field, 80);
    lv_obj_add_event_cb(s_name_field, keyboard_focus, LV_EVENT_FOCUSED, NULL);

    s_email_field = lv_textarea_create(s_form_screen);
    lv_obj_set_pos(s_email_field, 8, 99);
    lv_obj_set_size(s_email_field, 224, 36);
    lv_textarea_set_one_line(s_email_field, true);
    lv_textarea_set_placeholder_text(s_email_field, "Email address");
    lv_textarea_set_max_length(s_email_field, 254);
    lv_textarea_set_accepted_chars(s_email_field,
        "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789@._+-");
    lv_obj_add_event_cb(s_email_field, keyboard_focus, LV_EVENT_FOCUSED, NULL);

    s_role_dropdown = lv_dropdown_create(s_form_screen);
    lv_dropdown_set_options(s_role_dropdown, "Member\nAdministrator\nOwner");
    lv_obj_set_pos(s_role_dropdown, 8, 144);
    lv_obj_set_size(s_role_dropdown, 224, 36);
    lv_obj_t *save = make_button(s_form_screen, "Save user", 8, 188, 108, 36, form_save, NULL);
    lv_obj_t *cancel = make_button(s_form_screen, "Cancel", 124, 188, 108, 36, form_cancel, NULL);
    lv_obj_set_style_bg_color(cancel, lv_color_hex(0x475467), 0);
    (void)save;

    s_form_notice = lv_label_create(s_form_screen);
    lv_label_set_text(s_form_notice, "");
    lv_obj_set_pos(s_form_notice, 8, 230);
    lv_obj_set_width(s_form_notice, 224);
    lv_obj_set_style_text_font(s_form_notice, &lv_font_montserrat_12, 0);

    s_keyboard = lv_keyboard_create(s_form_screen);
    lv_obj_set_size(s_keyboard, SCREEN_WIDTH, 132);
    lv_obj_align(s_keyboard, LV_ALIGN_BOTTOM_MID, 0, 0);
    lv_obj_add_flag(s_keyboard, LV_OBJ_FLAG_HIDDEN);
    lv_obj_add_event_cb(s_keyboard, keyboard_ready, LV_EVENT_READY, NULL);

    if (record) {
        lv_textarea_set_text(s_name_field, record->name);
        lv_textarea_set_text(s_email_field, record->email);
        if (strcmp(record->role, "admin") == 0) lv_dropdown_set_selected(s_role_dropdown, 1U);
        else if (strcmp(record->role, "owner") == 0) lv_dropdown_set_selected(s_role_dropdown, 2U);
    }
    lv_screen_load(s_form_screen);
}

static esp_err_t create_home(void)
{
    s_home = lv_obj_create(NULL);
    if (!s_home) return ESP_ERR_NO_MEM;
    lv_obj_set_style_bg_color(s_home, lv_color_hex(COLOR_BG), 0);
    lv_obj_clear_flag(s_home, LV_OBJ_FLAG_SCROLLABLE);

    lv_obj_t *title = lv_label_create(s_home);
    lv_label_set_text(title, "User manager");
    lv_obj_set_pos(title, 8, 10);
    lv_obj_set_style_text_color(title, lv_color_hex(COLOR_INK), 0);
    lv_obj_set_style_text_font(title, &lv_font_montserrat_18, 0);

    lv_obj_t *add = make_button(s_home, "+ Add", 158, 6, 74, 34, start_add_user, NULL);
    lv_obj_set_style_bg_color(add, lv_color_hex(COLOR_ACCENT), 0);
    s_count_label = lv_label_create(s_home);
    lv_obj_set_pos(s_count_label, 8, 44);
    lv_obj_set_style_text_color(s_count_label, lv_color_hex(COLOR_MUTED), 0);
    s_notice_label = lv_label_create(s_home);
    lv_obj_set_pos(s_notice_label, 80, 44);
    lv_obj_set_width(s_notice_label, 152);
    lv_label_set_long_mode(s_notice_label, LV_LABEL_LONG_DOT);
    lv_obj_set_style_text_font(s_notice_label, &lv_font_montserrat_12, 0);

    s_list = lv_obj_create(s_home);
    lv_obj_set_pos(s_list, 8, 65);
    lv_obj_set_size(s_list, 224, 207);
    lv_obj_set_style_bg_opa(s_list, LV_OPA_TRANSP, 0);
    lv_obj_set_style_border_width(s_list, 0, 0);
    lv_obj_set_style_pad_all(s_list, 0, 0);
    lv_obj_set_style_pad_row(s_list, 2, 0);
    lv_obj_set_scrollbar_mode(s_list, LV_SCROLLBAR_MODE_ACTIVE);

    lv_obj_t *previous = make_button(s_home, "Prev", 8, 280, 58, 32,
                                     change_page, (void *)(intptr_t)-1);
    lv_obj_set_style_bg_color(previous, lv_color_hex(0x475467), 0);
    s_page_label = lv_label_create(s_home);
    lv_obj_align(s_page_label, LV_ALIGN_BOTTOM_MID, 0, -16);
    lv_obj_set_style_text_color(s_page_label, lv_color_hex(COLOR_MUTED), 0);
    lv_obj_t *next = make_button(s_home, "Next", 174, 280, 58, 32, change_page, (void *)1);
    lv_obj_set_style_bg_color(next, lv_color_hex(0x475467), 0);
    return ESP_OK;
}

esp_err_t user_screen_init(void)
{
    ESP_RETURN_ON_ERROR(create_home(), "user_screen", "Home screen allocation failed");
    refresh_home();
    lv_screen_load(s_home);
    return ESP_OK;
}
