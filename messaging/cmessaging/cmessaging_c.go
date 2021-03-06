/*
 Copyright 2014 Canonical Ltd.

 This program is free software: you can redistribute it and/or modify it
 under the terms of the GNU General Public License version 3, as published
 by the Free Software Foundation.

 This program is distributed in the hope that it will be useful, but
 WITHOUT ANY WARRANTY; without even the implied warranties of
 MERCHANTABILITY, SATISFACTORY QUALITY, or FITNESS FOR A PARTICULAR
 PURPOSE.  See the GNU General Public License for more details.

 You should have received a copy of the GNU General Public License along
 with this program.  If not, see <http://www.gnu.org/licenses/>.
*/
package cmessaging

/*
#include <glib.h>
#include <messaging-menu/messaging-menu.h>

// this is a .go file instead of a .c file because of dh-golang limitations

void handleActivate(gchar* c_action, const gchar * c_notification , gpointer obj);

static void activate_cb(MessagingMenuMessage* msg, gchar* action, GVariant* parameter, gpointer obj) {
    handleActivate(action, messaging_menu_message_get_id(msg), obj);
}

static GHashTable* map = NULL;

void add_notification (const gchar* desktop_id, const gchar* notification_id,
          const gchar* icon_path, const gchar* summary, const gchar* body,
          gint64 timestamp, const gchar** actions, gpointer obj) {
    if (map == NULL) {
        map = g_hash_table_new_full (g_str_hash, g_str_equal, g_free, g_object_unref);
    }

    MessagingMenuApp* app = g_hash_table_lookup (map, desktop_id);
    if (app == NULL) {
        GIcon* app_icon = g_icon_new_for_string (desktop_id, NULL);
        app = messaging_menu_app_new (desktop_id);
	messaging_menu_app_register (app);
	messaging_menu_app_append_source (app, "postal", app_icon, "Postal");
        g_hash_table_insert (map, g_strdup (desktop_id), app);
        g_object_unref (app_icon);
        // app only has a single refcount, and it's stored in the map. No need to g_object_unref it!
    }

    GIcon* icon = g_icon_new_for_string(icon_path, NULL);
    MessagingMenuMessage* msg = messaging_menu_message_new(notification_id, icon, summary,
                                                           "", body,
                                                           timestamp);
    // unity8 support for actions in the messaging menu is strange. Not doing that for now.
    messaging_menu_app_append_message(app, msg, "postal", TRUE);

    g_signal_connect(msg, "activate", G_CALLBACK(activate_cb), obj);
    g_object_unref(msg);
}

void remove_notification (const gchar* desktop_id, const gchar* notification_id) {
    if (map == NULL) {
        return;
    }
    MessagingMenuApp* app = g_hash_table_lookup (map, desktop_id);
    if (app == NULL) {
        // no app in the hash table, bailout
        return;
    }
    messaging_menu_app_remove_message_by_id (app, notification_id);
}

gboolean notification_exists (const gchar* desktop_id, const gchar* notification_id) {
    if (map == NULL) {
        return FALSE;
    }
    MessagingMenuApp* app = g_hash_table_lookup (map, desktop_id);
    if (app == NULL) {
        // no app in the hash table, bailout
        return FALSE;
    }
    MessagingMenuMessage* msg = messaging_menu_app_get_message(app, notification_id);
    if (msg != NULL) {
        // the notification is still there
        return TRUE;
    }
    return FALSE;
}
*/
import "C"
