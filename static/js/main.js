import { showAdminLogin, loginAdmin, logoutAdmin, initAuth } from './auth.js';
import { loadShiftTypes, createShiftType, createShift, deleteShift, deleteShiftType } from './shifts.js';
import { setLanguage, t } from './i18n.js';
import './participants.js';  // Import for side effects (window assignments)
import { joinShift } from './participants.js';

// Make functions globally available
window.showAdminLogin = showAdminLogin;
window.loginAdmin = loginAdmin;
window.logoutAdmin = logoutAdmin;
window.createShiftType = createShiftType;
window.createShift = createShift;
window.deleteShift = deleteShift;
window.deleteShiftType = deleteShiftType;
window.setLanguage = setLanguage;

// Initialize the application
document.addEventListener('DOMContentLoaded', () => {
    // Get config from the window object (set in base.html)
    const config = window.serverConfig;
    if (config) {
        initAuth(config);
    }
    loadShiftTypes();
}); 