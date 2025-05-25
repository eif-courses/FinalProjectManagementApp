// Mobile menu toggle
function toggleMobileMenu() {
    const mobileMenu = document.getElementById('mobile-menu');
    const menuIcon = document.getElementById('menu-icon');
    const closeIcon = document.getElementById('close-icon');

    if (mobileMenu.classList.contains('hidden')) {
        mobileMenu.classList.remove('hidden');
        menuIcon.classList.add('hidden');
        closeIcon.classList.remove('hidden');
    } else {
        mobileMenu.classList.add('hidden');
        menuIcon.classList.remove('hidden');
        closeIcon.classList.add('hidden');
    }
}

// Dropdown toggle
function toggleDropdown(dropdownId) {
    const dropdown = document.getElementById(dropdownId);
    if (dropdown) {
        dropdown.classList.toggle('hidden');
    }
}

// Close dropdowns when clicking outside
document.addEventListener('click', function(event) {
    const dropdowns = ['language-dropdown', 'user-dropdown'];

    dropdowns.forEach(function(dropdownId) {
        const dropdown = document.getElementById(dropdownId);
        const button = dropdown?.previousElementSibling;

        if (dropdown && !dropdown.contains(event.target) && !button?.contains(event.target)) {
            dropdown.classList.add('hidden');
        }
    });
});

// Close mobile menu when clicking outside
document.addEventListener('click', function(event) {
    const mobileMenu = document.getElementById('mobile-menu');
    const menuButton = document.querySelector('[onclick="toggleMobileMenu()"]');

    if (mobileMenu && !mobileMenu.contains(event.target) && !menuButton?.contains(event.target)) {
        mobileMenu.classList.add('hidden');
        document.getElementById('menu-icon').classList.remove('hidden');
        document.getElementById('close-icon').classList.add('hidden');
    }
});