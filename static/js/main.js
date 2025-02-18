import { showAdminLogin, loginAdmin, logoutAdmin, initAuth } from './auth.js';
import { loadShiftTypes, createShiftType, createShift, deleteShift, deleteShiftType } from './shifts.js';
import { setLanguage, t } from './i18n.js';
import './participants.js';  // Import for side effects (window assignments)
import { joinShift } from './participants.js';
import { createEvent, copyUrl } from './events.js';

let adminLoggedIn = false;

function updateAdminUI() {
    const loginBtn = document.getElementById('adminLoginBtn');
    const logoutBtn = document.getElementById('adminLogoutBtn');
    if (loginBtn && logoutBtn) {
        loginBtn.style.display = adminLoggedIn ? 'none' : 'block';
        logoutBtn.style.display = adminLoggedIn ? 'block' : 'none';
    }
}

function showError(message) {
    // TODO: Implement error display
    alert(message);
}

// Make functions globally available
window.showAdminLogin = showAdminLogin;
window.loginAdmin = loginAdmin;
window.logoutAdmin = logoutAdmin;
window.createShiftType = createShiftType;
window.createShift = createShift;
window.deleteShift = deleteShift;
window.deleteShiftType = deleteShiftType;
window.setLanguage = setLanguage;
window.createShiftPlan = () => createEvent({ showUrlModal: true, redirectToEvent: false });
window.copyUrl = copyUrl;

// Initialize the application
document.addEventListener('DOMContentLoaded', () => {
    // Get config from the window object (set in base.html)
    const config = window.serverConfig;
    
    // Check which page we're on
    const isEventPage = window.location.pathname.match(/\/events\/([^\/]+)/);
    const isHomePage = window.location.pathname === '/';
    
    // Initialize auth if we have config
    if (config) {
        initAuth(config);
    }
    
    // Load shift types only on event page
    if (isEventPage) {
        const eventId = isEventPage[1];
        loadShiftTypes(eventId);
    }
}); 