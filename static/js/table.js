function toggleDropdown(id) {
    const dropdown = document.getElementById(id + '-dropdown');
    const allDropdowns = document.querySelectorAll('[id$="-dropdown"]');

    allDropdowns.forEach(dd => {
        if (dd.id !== id + '-dropdown') {
            dd.classList.add('hidden');
        }
    });

    dropdown.classList.toggle('hidden');
}

function openModal(modalId) {
    document.getElementById(modalId).classList.remove('hidden');
    document.body.style.overflow = 'hidden';
}

function closeModal(modalId) {
    document.getElementById(modalId).classList.add('hidden');
    document.body.style.overflow = 'auto';
}

// Close dropdowns when clicking outside
document.addEventListener('click', function(e) {
    if (!e.target.closest('.relative')) {
        document.querySelectorAll('[id$="-dropdown"]').forEach(dd => {
            dd.classList.add('hidden');
        });
    }
});