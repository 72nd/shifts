export let currentLanguage = 'de';

// Get system language
function getSystemLanguage() {
    const lang = navigator.language.toLowerCase();
    return lang.startsWith('de') ? 'de' : 'en';
}

const translations = {
    de: {
        // Navigation & Headers
        'admin_login': 'Login',
        'admin_logout': 'Admin Logout',
        'shift_planning': 'Schichtplanung',

        // Table Columns
        'time': 'Zeit',
        'duration': 'Dauer',
        'occupancy': 'Besetzung',
        'actions': 'Akt.',

        // Filter Section
        'shift_type_filter': 'Schichttyp Filter',
        'all_shift_types': 'Alle Schichttypen',
        'day': 'Tag',
        'all_days': 'Alle Tage',
        'status': 'Status',
        'all': 'Alle',
        'completely_empty': 'Komplett leer',
        'critical': 'Kritisch (< 50% besetzt)',
        'spots_available': 'Noch Plätze frei',
        'fully_booked': 'Voll besetzt',
        'reset_filters': 'Filter zurücksetzen',
        'active_filters': 'Aktive Filter',

        // Shift Type Card
        'new_shift_type': 'Neuer Schichttyp',
        'new_shift': 'Neue Schicht',
        'edit_shift_type': 'Schichttyp bearbeiten',
        'edit_shift': 'Schicht bearbeiten',
        'location': 'Ort',
        'show_more': 'Mehr anzeigen',
        'show_less': 'Weniger anzeigen',
        'no_participants': 'Keine Helfenden',

        // Modals
        'password': 'Passwort',
        'close': 'Schließen',
        'cancel': 'Abbrechen',
        'create': 'Erstellen',
        'save': 'Speichern',
        'delete': 'Löschen',
        'edit': 'Bearbeiten',
        'title': 'Titel',
        'description': 'Beschreibung',
        'start_time': 'Startzeit',
        'duration': 'Dauer',
        'duration_minutes': 'Dauer (Minuten)',
        'required_people': 'Benötigte Personen',
        'consecutive_shifts': 'Anzahl aufeinanderfolgender Schichten',
        'creates_multiple_shifts': 'Erstellt mehrere Schichten, die direkt aufeinander folgen',
        
        // Participants
        'sign_up': 'Eintragen',
        'sign_up_for_shift': 'Für Schicht eintragen',
        'name': 'Name',
        'name_hint': 'Gerne auch Spitznamen verwenden. Bitte achte darauf, dass dein Name möglichst einzigartig ist (z.B. "Marie K." statt nur "Marie"). Keine Sonderzeichen.',
        'participants': 'Helfende',
        'occupancy': 'Besetzung',
        'remove_from_shift': 'Aus der Schicht austragen',
        
        // Confirmations
        'confirm_delete_shift': 'Möchten Sie diese Schicht wirklich löschen?',
        'confirm_delete_shift_type': 'Möchten Sie diesen Schichttyp und alle zugehörigen Schichten wirklich löschen?',
        'confirm_remove_participant': 'Möchten Sie {name} wirklich aus der Schicht austragen?',
        
        // Status Messages
        'empty_shifts': 'leer',
        'critical_shifts': 'kritisch',
        'needed_helpers': 'Benötigte Helfende',
        
        // Errors
        'error_creating_shift_type': 'Fehler beim Erstellen des Schichttyps',
        'error_creating_shift': 'Fehler beim Erstellen der Schicht',
        'error_updating_shift_type': 'Fehler beim Aktualisieren des Schichttyps',
        'error_updating_shift': 'Fehler beim Aktualisieren der Schicht',
        'error_deleting_shift': 'Fehler beim Löschen der Schicht',
        'error_deleting_shift_type': 'Fehler beim Löschen des Schichttyps',
        'error_joining_shift': 'Fehler beim Eintragen in die Schicht',
        'error_leaving_shift': 'Fehler beim Austragen aus der Schicht',
        'wrong_password': 'Falsches Passwort',

        // Participant Search
        'participant_search': 'Helfer:innensuche',
        'participant_search_placeholder': 'Namen eingeben...',

        // Validation error messages
        'error_title_length_de': 'Der deutsche Titel muss zwischen 2 und 100 Zeichen lang sein',
        'error_title_chars_de': 'Der deutsche Titel enthält unerlaubte Zeichen.',
        'error_title_length_en': 'Der englische Titel muss zwischen 2 und 100 Zeichen lang sein',
        'error_title_chars_en': 'Der englische Titel enthält unerlaubte Zeichen.',
        'error_location_length': 'Der Ort muss zwischen 2 und 100 Zeichen lang sein',
        'error_location_chars': 'Der Ort enthält unerlaubte Zeichen.',
        'error_start_time_required': 'Bitte geben Sie eine Startzeit an',
        'error_start_time_future': 'Die Startzeit muss in der Zukunft liegen',
        'error_duration_min': 'Die Dauer muss mindestens 1 Minute betragen',
        'error_required_min': 'Es muss mindestens 1 Person benötigt werden',
        'error_name_length': 'Der Name muss zwischen 2 und 50 Zeichen lang sein',
        'error_name_chars': 'Der Name enthält unerlaubte Zeichen.',
        'allowed_chars_title': 'Erlaubte Zeichen sind: {chars}',
        'allowed_chars_name': 'Erlaubte Zeichen sind: {chars}',

        // Shift Type Password
        'shift_type_password': 'Passwort',
        'shift_type_password_hint': 'Optional: Wenn gesetzt, müssen sich Helfende bei der zuständigen Crew melden, um das Passwort zu erfahren.',
        'shift_password': 'Passwort',
        'shift_password_hint': 'Diese Schicht ist passwortgeschützt, da eine Einweisung erforderlich ist oder die Crew dich vorab kennenlernen möchte. Die zuständige Crew wird dir das Passwort persönlich mitteilen.',
        'error_wrong_password': 'Falsches Passwort. Bitte melde dich bei der zuständigen Crew.',

        // Sort functionality
        'sort_shift_types': 'Schichttypen sortieren',
        'sort_shift_types_hint': 'Ziehen Sie die Einträge mit der Maus, um die Reihenfolge anzupassen.',
        'error_updating_sort_order': 'Fehler beim Speichern der Reihenfolge',

        // Day Delta Helper
        'set_date': 'Datum setzen',
        'set_day_relative_to_event': 'Tag relativ zum Event ({date}) setzen:',
        'day_delta_placeholder': 'z.B. -1 für einen Tag davor',

        // Event-related translations
        'events': "Events",
        'new_event': "Neues Event",
        'event_title': "Titel",
        'start_date': "Startdatum",
        'end_date': "Enddatum",
        'admin_password': "Admin-Passwort",
        'admin_password_hint': "Dieses Passwort wird benötigt, um Schichten und Schichttypen zu verwalten.",
        'day_zero': "Tag Null",
        'day_zero_hint': "Optional: Referenzdatum für relative Tagesangaben (z.B. \"Tag -1\" für einen Tag vor dem Event)",
        'debug_mode': "Debug-Modus",
        'debug_mode_hint': "Aktiviert zusätzliche Funktionen für Entwicklung und Testing.",
        'view_shifts': "Schichten anzeigen",
        'no_events': "Keine Events verfügbar.",
        'current_language': "Deutsch",
        'language_de': "Deutsch",
        'language_en': "English",
        'add_shift_type': "Schichttyp hinzufügen",
        'delete_shift_type': "Schichttyp löschen",
        'title_de': "Titel (Deutsch)",
        'title_en': "Titel (Englisch)",
        'description_de': "Beschreibung (Deutsch)",
        'description_en': "Beschreibung (Englisch)",
        'add_shift': "Schicht hinzufügen",
        'delete_shift': "Schicht löschen",
        'join_shift': "Schicht übernehmen",
        'leave_shift': "Schicht verlassen",
        'enter_name': "Name eingeben",
        'join': "Übernehmen",
        'leave': "Verlassen",
        'confirm_delete': "Wirklich löschen?",
        'error_loading_shift_types': "Fehler beim Laden der Schichttypen",
        'error_admin_login': "Falsches Admin-Passwort",
        'sort_shift_types': "Schichttypen sortieren",
        'error_updating_sort_order': "Fehler beim Aktualisieren der Sortierreihenfolge",
        'set_day_relative_to_event': "Tag relativ zum Event setzen",
        'day_delta_placeholder': "Tage (z.B. -1 für einen Tag vor dem Event)"
    },
    en: {
        // Navigation & Headers
        'admin_login': 'Login',
        'admin_logout': 'Admin Logout',
        'shift_planning': 'Shift Planning',

        // Table Columns
        'time': 'Time',
        'duration': 'Duration',
        'occupancy': 'Occupancy',
        'actions': 'Act.',

        // Filter Section
        'shift_type_filter': 'Shift Type Filter',
        'all_shift_types': 'All Shift Types',
        'day': 'Day',
        'all_days': 'All Days',
        'status': 'Status',
        'all': 'All',
        'completely_empty': 'Completely Empty',
        'critical': 'Critical (< 50% filled)',
        'spots_available': 'Spots Available',
        'fully_booked': 'Fully Booked',
        'reset_filters': 'Reset Filters',
        'active_filters': 'Active Filters',

        // Shift Type Card
        'new_shift_type': 'New Shift Type',
        'new_shift': 'New Shift',
        'edit_shift_type': 'Edit Shift Type',
        'edit_shift': 'Edit Shift',
        'location': 'Location',
        'show_more': 'Show More',
        'show_less': 'Show Less',
        'no_participants': 'No Participants',

        // Modals
        'password': 'Password',
        'close': 'Close',
        'cancel': 'Cancel',
        'create': 'Create',
        'save': 'Save',
        'delete': 'Delete',
        'edit': 'Edit',
        'title': 'Title',
        'description': 'Description',
        'start_time': 'Start Time',
        'duration': 'Duration',
        'duration_minutes': 'Duration (minutes)',
        'required_people': 'Required People',
        'consecutive_shifts': 'Number of Consecutive Shifts',
        'creates_multiple_shifts': 'Creates multiple shifts that follow each other',
        
        // Participants
        'sign_up': 'Sign Up',
        'sign_up_for_shift': 'Sign Up for Shift',
        'name': 'Name',
        'name_hint': 'Feel free to use nicknames. Please make sure your name is as unique as possible (e.g. "Marie K." instead of just "Marie"). No special characters.',
        'participants': 'Helpers',
        'occupancy': 'Occupancy',
        'remove_from_shift': 'Remove from Shift',
        'no_participants': 'No Helpers',
        
        // Confirmations
        'confirm_delete_shift': 'Do you really want to delete this shift?',
        'confirm_delete_shift_type': 'Do you really want to delete this shift type and all associated shifts?',
        'confirm_remove_participant': 'Do you really want to remove {name} from the shift?',
        
        // Status Messages
        'empty_shifts': 'empty',
        'critical_shifts': 'critical',
        'needed_helpers': 'Helpers needed',
        
        // Errors
        'error_creating_shift_type': 'Error creating shift type',
        'error_creating_shift': 'Error creating shift',
        'error_updating_shift_type': 'Error updating shift type',
        'error_updating_shift': 'Error updating shift',
        'error_deleting_shift': 'Error deleting shift',
        'error_deleting_shift_type': 'Error deleting shift type',
        'error_joining_shift': 'Error joining shift',
        'error_leaving_shift': 'Error leaving shift',
        'wrong_password': 'Wrong password',

        // Participant Search
        'participant_search': 'Helper Search',
        'participant_search_placeholder': 'Enter name...',

        // Validation error messages
        'error_title_length_de': 'The German title must be between 2 and 100 characters long',
        'error_title_chars_de': 'The German title contains invalid characters.',
        'error_title_length_en': 'The English title must be between 2 and 100 characters long',
        'error_title_chars_en': 'The English title contains invalid characters.',
        'error_location_length': 'The location must be between 2 and 100 characters long',
        'error_location_chars': 'The location contains invalid characters.',
        'error_start_time_required': 'Please enter a start time',
        'error_start_time_future': 'The start time must be in the future',
        'error_duration_min': 'The duration must be at least 1 minute',
        'error_required_min': 'At least 1 person is required',
        'error_name_length': 'The name must be between 2 and 50 characters long',
        'error_name_chars': 'The name contains invalid characters.',
        'allowed_chars_title': 'Allowed characters are: {chars}',
        'allowed_chars_name': 'Allowed characters are: {chars}',

        // Shift Type Password
        'shift_type_password': 'Password',
        'shift_type_password_hint': 'Optional: If set, helpers need to contact the responsible crew to get the password.',
        'shift_password': 'Password',
        'shift_password_hint': 'This shift is password protected because an introduction is required or the crew would like to get to know you beforehand. The responsible crew will provide you with the password in person.',
        'error_wrong_password': 'Wrong password. Please contact the responsible crew.',

        // Sort functionality
        'sort_shift_types': 'Sort Shift Types',
        'sort_shift_types_hint': 'Drag and drop items to adjust the order.',
        'error_updating_sort_order': 'Error saving the sort order',

        // Day Delta Helper
        'set_date': 'Set date',
        'set_day_relative_to_event': 'Set day relative to event ({date}):',
        'day_delta_placeholder': 'e.g. -1 for one day before',

        // Event-related translations
        'events': "Events",
        'new_event': "New Event",
        'event_title': "Title",
        'start_date': "Start Date",
        'end_date': "End Date",
        'admin_password': "Admin Password",
        'admin_password_hint': "This password is required to manage shifts and shift types.",
        'day_zero': "Day Zero",
        'day_zero_hint': "Optional: Reference date for relative day specifications (e.g. \"Day -1\" for one day before the event)",
        'debug_mode': "Debug Mode",
        'debug_mode_hint': "Enables additional features for development and testing.",
        'view_shifts': "View Shifts",
        'no_events': "No events available.",
        'current_language': "English",
        'language_de': "German",
        'language_en': "English",
        'add_shift_type': "Add Shift Type",
        'delete_shift_type': "Delete Shift Type",
        'title_de': "Title (German)",
        'title_en': "Title (English)",
        'description_de': "Description (German)",
        'description_en': "Description (English)",
        'add_shift': "Add Shift",
        'delete_shift': "Delete Shift",
        'join_shift': "Join Shift",
        'leave_shift': "Leave Shift",
        'enter_name': "Enter Name",
        'join': "Join",
        'leave': "Leave",
        'confirm_delete': "Really delete?",
        'error_loading_shift_types': "Error loading shift types",
        'error_admin_login': "Wrong admin password",
        'sort_shift_types': "Sort Shift Types",
        'error_updating_sort_order': "Error updating sort order",
        'set_day_relative_to_event': "Set day relative to event",
        'day_delta_placeholder': "Days (e.g. -1 for one day before the event)"
    }
};

export function setLanguage(lang) {
    if (translations[lang]) {
        currentLanguage = lang;
        document.documentElement.lang = lang;
        updatePageTranslations();
        
        // Update active state of language buttons
        document.querySelectorAll('.language-btn').forEach(btn => {
            if (btn.getAttribute('data-language') === lang) {
                btn.classList.add('active');
            } else {
                btn.classList.remove('active');
            }
        });

        // Update visibility of language-specific content
        document.querySelectorAll('[data-language]').forEach(el => {
            const isButton = el.classList.contains('language-btn') || el.classList.contains('description-toggle');
            if (!isButton) {
                if (el.getAttribute('data-language') === lang) {
                    el.style.display = 'block';
                } else {
                    el.style.display = 'none';
                }
            }
            
            // Handle description toggle buttons separately
            if (el.classList.contains('description-toggle')) {
                if (el.getAttribute('data-language') === lang) {
                    el.style.display = el.closest('.description-container').querySelector('.description.truncated') ? 'inline-flex' : 'none';
                } else {
                    el.style.display = 'none';
                }
            }
        });

        // Dispatch a custom event for language change
        document.dispatchEvent(new CustomEvent('languageChanged', { detail: { language: lang } }));
    }
}

export function t(key, params = {}) {
    const translation = translations[currentLanguage][key] || translations['en'][key] || key;
    return translation.replace(/\{(\w+)\}/g, (match, key) => params[key] || match);
}

function updatePageTranslations() {
    // Update all elements with data-i18n attribute
    document.querySelectorAll('[data-i18n]').forEach(element => {
        const key = element.getAttribute('data-i18n');
        if (key) {
            if (element.tagName === 'INPUT' || element.tagName === 'TEXTAREA') {
                if (element.getAttribute('placeholder')) {
                    element.placeholder = t(key);
                }
            } else {
                element.textContent = t(key);
            }
        }
    });
}

// Initialize with system language
document.addEventListener('DOMContentLoaded', () => {
    setLanguage(getSystemLanguage());
}); 