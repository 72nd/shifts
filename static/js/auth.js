// Auth state
let isAdmin = false;
let adminPassword = '';

export function showAdminLogin() {
    new bootstrap.Modal(document.getElementById('adminLoginModal')).show();
}

export function loginAdmin() {
    const form = document.getElementById('adminLoginForm');
    const password = form.elements.password.value;

    fetch('/admin/verify', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify({ password }),
    })
    .then(response => {
        if (response.ok) {
            isAdmin = true;
            adminPassword = password;
            document.body.classList.add('admin-mode');
            document.getElementById('adminLoginBtn').style.display = 'none';
            document.getElementById('adminLogoutBtn').style.display = 'inline-block';
            bootstrap.Modal.getInstance(document.getElementById('adminLoginModal')).hide();
            form.reset();
        } else {
            alert('Falsches Passwort');
        }
    });
}

export function logoutAdmin() {
    isAdmin = false;
    adminPassword = '';
    document.body.classList.remove('admin-mode');
    document.getElementById('adminLoginBtn').style.display = 'inline-block';
    document.getElementById('adminLogoutBtn').style.display = 'none';
}

export function fetchWithAuth(url, options = {}) {
    if (isAdmin) {
        options.headers = {
            ...options.headers,
            'X-Admin-Token': adminPassword
        };
    }
    return fetch(url, options);
} 