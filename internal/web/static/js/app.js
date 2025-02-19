// Theme handling
document.addEventListener('DOMContentLoaded', () => {
    const themeController = document.querySelector('.theme-controller');
    
    // Load theme preference
    const savedTheme = localStorage.getItem('theme') || 'light';
    document.documentElement.setAttribute('data-theme', savedTheme);
    themeController.checked = savedTheme === 'dark';

    // Handle theme changes
    themeController.addEventListener('change', (e) => {
        const theme = e.target.checked ? 'dark' : 'light';
        document.documentElement.setAttribute('data-theme', theme);
        localStorage.setItem('theme', theme);
    });
}); 