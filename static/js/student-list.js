// Student List Management JavaScript

// Global variables
let currentStudentId = null;
let isLoading = false;

// Initialize when DOM is loaded
document.addEventListener('DOMContentLoaded', function() {
    initializeStudentList();
});

// Initialize student list functionality
function initializeStudentList() {
    setupSearchFunctionality();
    setupFilterFunctionality();
    setupClickOutsideHandlers();
    loadAllDocuments();
}

// Search functionality
function setupSearchFunctionality() {
    const searchInput = document.getElementById('search');
    if (searchInput) {
        let searchTimeout;
        searchInput.addEventListener('input', function() {
            clearTimeout(searchTimeout);
            searchTimeout = setTimeout(() => {
                performSearch(this.value);
            }, 300); // Debounce search
        });

        // Handle Enter key
        searchInput.addEventListener('keypress', function(e) {
            if (e.key === 'Enter') {
                e.preventDefault();
                performSearch(this.value);
            }
        });
    }
}

// Perform search operation
function performSearch(searchTerm) {
    const url = new URL(window.location);
    if (searchTerm.trim()) {
        url.searchParams.set('search', searchTerm.trim());
    } else {
        url.searchParams.delete('search');
    }
    url.searchParams.set('page', '1'); // Reset to first page
    window.location.href = url.toString();
}

// Filter functionality
function setupFilterFunctionality() {
    const filterForm = document.getElementById('filterForm');
    if (filterForm) {
        // Handle form submission
        filterForm.addEventListener('submit', function(e) {
            e.preventDefault();
            submitForm();
        });
    }
}

// Toggle filters visibility
function toggleFilters() {
    const filters = document.getElementById('filters');
    if (filters) {
        filters.classList.toggle('hidden');

        // Update button state
        const filterButton = document.querySelector('[onclick="toggleFilters()"]');
        if (filterButton) {
            const isVisible = !filters.classList.contains('hidden');
            filterButton.setAttribute('aria-expanded', isVisible);
        }
    }
}

// Clear all filters
function clearFilters() {
    const form = document.getElementById('filterForm');
    if (form) {
        // Reset all form elements
        const inputs = form.querySelectorAll('input, select');
        inputs.forEach(input => {
            if (input.type === 'checkbox' || input.type === 'radio') {
                input.checked = false;
            } else {
                input.value = '';
            }
        });

        // Submit the cleared form
        submitForm();
    }
}

// Submit filter form
function submitForm() {
    const form = document.getElementById('filterForm');
    if (form) {
        const formData = new FormData(form);
        const url = new URL(window.location);

        // Clear existing search params (except essential ones)
        const paramsToKeep = ['page'];
        for (const [key] of url.searchParams) {
            if (!paramsToKeep.includes(key)) {
                url.searchParams.delete(key);
            }
        }

        // Add form data to URL
        for (const [key, value] of formData) {
            if (value && value.trim()) {
                url.searchParams.set(key, value.trim());
            }
        }

        // Reset to first page when filtering
        url.searchParams.set('page', '1');

        window.location.href = url.toString();
    }
}

// Document management
function loadAllDocuments() {
    const docCells = document.querySelectorAll('[id^="docs-"]');
    docCells.forEach(cell => {
        const studentId = cell.id.replace('docs-', '');
        const userRole = getCurrentUserRole();
        loadDocuments(parseInt(studentId), userRole);
    });
}

// Load documents for a specific student
function loadDocuments(studentId, userRole) {
    const container = document.getElementById(`docs-${studentId}`);
    if (!container) return;

    // Show loading state
    container.innerHTML = '<div class="text-xs text-muted-foreground italic">Kraunama...</div>';

    // Make API call to get documents
    fetch(`/api/students/${studentId}/documents`, {
        method: 'GET',
        headers: {
            'Content-Type': 'application/json',
        },
        credentials: 'same-origin'
    })
        .then(response => {
            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }
            return response.json();
        })
        .then(data => {
            displayDocuments(container, data.documents || [], userRole);
        })
        .catch(error => {
            console.error('Error loading documents:', error);
            container.innerHTML = '<div class="text-xs text-red-500">Klaida kraunant dokumentus</div>';
        });
}

// Display documents in the container
// Updated displayDocuments function to handle multiple document types
function displayDocuments(container, documents, userRole) {
    if (!documents || documents.length === 0) {
        container.innerHTML = '<div class="text-xs text-muted-foreground">Nėra dokumentų</div>';
        return;
    }

    // Document type mapping with icons
    const documentTypes = {
        'VIDEO': { icon: '🎥', label: 'Video' },
        'RECOMMENDATION.PDF': { icon: '📋', label: 'Rekomendacija' },
        'FINAL_THESIS.PDF': { icon: '📄', label: 'Darbas' },
        'SOURCE_CODE': { icon: '💻', label: 'Kodas' }
    };

    const documentsHtml = documents.map(doc => {
        const docType = documentTypes[doc.type] || { icon: '📎', label: doc.type };
        const statusClass = getDocumentStatusClass(doc.status);
        const statusText = getDocumentStatusText(doc.status);

        return `
            <div class="flex items-center justify-between text-xs border rounded p-1">
                <div class="flex items-center gap-1 flex-1 min-w-0">
                    <span class="text-sm">${docType.icon}</span>
                    <span class="truncate text-xs" title="${doc.name}">
                        ${docType.label}
                    </span>
                </div>
                <div class="flex items-center gap-1">
                    <span class="inline-flex items-center rounded px-1 py-0.5 text-xs font-medium ${statusClass}">
                        ${statusText}
                    </span>
                    ${canViewDocument(userRole, doc) ? `
                        <button onclick="viewDocument(${doc.id})" 
                                class="text-xs text-blue-600 hover:text-blue-800 p-1">
                            <svg class="h-3 w-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"/>
                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z"/>
                            </svg>
                        </button>
                    ` : ''}
                </div>
            </div>
        `;
    }).join('');

    container.innerHTML = documentsHtml;
}

// Get CSS class for document status
function getDocumentStatusClass(status) {
    switch (status) {
        case 'approved':
            return 'bg-green-100 text-green-800';
        case 'pending':
            return 'bg-yellow-100 text-yellow-800';
        case 'rejected':
            return 'bg-red-100 text-red-800';
        default:
            return 'bg-gray-100 text-gray-800';
    }
}

// Get text for document status
function getDocumentStatusText(status) {
    switch (status) {
        case 'approved':
            return 'Patvirtinta';
        case 'pending':
            return 'Laukia';
        case 'rejected':
            return 'Atmesta';
        default:
            return 'Juodraštis';
    }
}

// Check if user can view document
function canViewDocument(userRole, document) {
    // Add your business logic here
    return ['admin', 'department_head', 'supervisor'].includes(userRole) ||
        document.status === 'approved';
}

// Registration management
function viewRegistration(studentId) {
    if (isLoading) return;

    window.location.href = `/students/${studentId}/topic/view`;
}

function editRegistration(studentId) {
    if (isLoading) return;

    window.location.href = `/students/${studentId}/topic/edit`;
}

function approveRegistration(studentId) {
    if (isLoading) return;

    if (confirm('Ar tikrai norite patvirtinti šią temą?')) {
        performAction(`/api/students/${studentId}/topic/approve`, 'POST')
            .then(() => {
                showSuccessMessage('Tema sėkmingai patvirtinta');
                setTimeout(() => window.location.reload(), 1000);
            })
            .catch(error => {
                showErrorMessage('Klaida tvirtinant temą: ' + error.message);
            });
    }
}

function rejectRegistration(studentId) {
    if (isLoading) return;

    const reason = prompt('Įveskite atmetimo priežastį:');
    if (reason && reason.trim()) {
        performAction(`/api/students/${studentId}/topic/reject`, 'POST', { reason: reason.trim() })
            .then(() => {
                showSuccessMessage('Tema atmesta');
                setTimeout(() => window.location.reload(), 1000);
            })
            .catch(error => {
                showErrorMessage('Klaida atmetant temą: ' + error.message);
            });
    }
}

// Document management
function uploadDocument(studentId) {
    if (isLoading) return;

    window.location.href = `/students/${studentId}/documents/upload`;
}

function viewDocument(documentId) {
    if (isLoading) return;

    window.open(`/api/documents/${documentId}/view`, '_blank');
}

// Review management
function viewReview(studentId) {
    if (isLoading) return;

    window.location.href = `/students/${studentId}/review/view`;
}

function editReview(studentId) {
    if (isLoading) return;

    window.location.href = `/students/${studentId}/review/edit`;
}

function assignReviewer(studentId) {
    if (isLoading) return;

    window.location.href = `/students/${studentId}/assign-reviewer`;
}

// Supervisor report management
function viewSupervisorReport(studentId) {
    if (isLoading) return;

    window.location.href = `/students/${studentId}/supervisor-report/view`;
}

function editSupervisorReport(studentId) {
    if (isLoading) return;

    window.location.href = `/students/${studentId}/supervisor-report/edit`;
}

// Actions menu management
function toggleActions(studentId) {
    // Close other open action menus
    const allActionMenus = document.querySelectorAll('[id^="actions-"]');
    allActionMenus.forEach(menu => {
        if (menu.id !== `actions-${studentId}`) {
            menu.classList.add('hidden');
        }
    });

    // Toggle current menu
    const menu = document.getElementById(`actions-${studentId}`);
    if (menu) {
        menu.classList.toggle('hidden');
    }
}

// Student management
function editStudent(studentId) {
    if (isLoading) return;

    window.location.href = `/students/edit/${studentId}`;
}

function deleteStudent(studentId) {
    if (isLoading) return;

    if (confirm('Ar tikrai norite ištrinti šį studentą? Šis veiksmas negrįžtamas.')) {
        performAction(`/api/students/${studentId}`, 'DELETE')
            .then(() => {
                showSuccessMessage('Studentas sėkmingai ištrintas');
                setTimeout(() => window.location.reload(), 1000);
            })
            .catch(error => {
                showErrorMessage('Klaida trinant studentą: ' + error.message);
            });
    }
}

// Utility functions
function performAction(url, method, data = null) {
    isLoading = true;

    const options = {
        method: method,
        headers: {
            'Content-Type': 'application/json',
        },
        credentials: 'same-origin'
    };

    if (data && (method === 'POST' || method === 'PUT')) {
        options.body = JSON.stringify(data);
    }

    return fetch(url, options)
        .then(response => {
            if (!response.ok) {
                return response.json().then(err => {
                    throw new Error(err.message || `HTTP error! status: ${response.status}`);
                });
            }
            return response.json();
        })
        .finally(() => {
            isLoading = false;
        });
}

function getCurrentUserRole() {
    // Try to get user role from a data attribute or global variable
    const userRoleElement = document.querySelector('[data-user-role]');
    if (userRoleElement) {
        return userRoleElement.getAttribute('data-user-role');
    }

    // Fallback: try to get from a global variable if set
    if (typeof window.currentUserRole !== 'undefined') {
        return window.currentUserRole;
    }

    // Default fallback
    return 'student';
}

// Click outside handlers
function setupClickOutsideHandlers() {
    document.addEventListener('click', function(event) {
        // Close action menus when clicking outside
        if (!event.target.closest('[id^="actions-"]') &&
            !event.target.closest('button[onclick^="toggleActions"]')) {
            const allActionMenus = document.querySelectorAll('[id^="actions-"]');
            allActionMenus.forEach(menu => {
                menu.classList.add('hidden');
            });
        }

        // Close filters when clicking outside
        const filters = document.getElementById('filters');
        const filterButton = document.querySelector('[onclick="toggleFilters()"]');
        if (filters && !filters.contains(event.target) &&
            !filterButton?.contains(event.target)) {
            // Don't auto-close filters as they might be intentionally opened
        }
    });
}

// Notification functions
function showSuccessMessage(message) {
    showNotification(message, 'success');
}

function showErrorMessage(message) {
    showNotification(message, 'error');
}

function showNotification(message, type = 'info') {
    // Create notification element
    const notification = document.createElement('div');
    notification.className = `fixed top-4 right-4 z-50 max-w-sm p-4 rounded-lg shadow-lg transition-all duration-300 transform translate-x-full`;

    // Set colors based on type
    switch (type) {
        case 'success':
            notification.classList.add('bg-green-500', 'text-white');
            break;
        case 'error':
            notification.classList.add('bg-red-500', 'text-white');
            break;
        default:
            notification.classList.add('bg-blue-500', 'text-white');
    }

    notification.innerHTML = `
        <div class="flex items-center justify-between">
            <span class="text-sm font-medium">${message}</span>
            <button onclick="this.parentElement.parentElement.remove()" 
                    class="ml-2 text-white hover:text-gray-200">
                <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/>
                </svg>
            </button>
        </div>
    `;

    // Add to DOM
    document.body.appendChild(notification);

    // Animate in
    setTimeout(() => {
        notification.classList.remove('translate-x-full');
    }, 100);

    // Auto remove after 5 seconds
    setTimeout(() => {
        notification.classList.add('translate-x-full');
        setTimeout(() => {
            if (notification.parentElement) {
                notification.remove();
            }
        }, 300);
    }, 5000);
}

// Export functionality (if needed)
function exportData() {
    if (isLoading) return;

    const url = new URL('/api/students/export', window.location.origin);

    // Add current filters to export
    const urlParams = new URLSearchParams(window.location.search);
    for (const [key, value] of urlParams) {
        if (key !== 'page') {
            url.searchParams.set(key, value);
        }
    }

    // Create a temporary link and click it
    const link = document.createElement('a');
    link.href = url.toString();
    link.download = `students_export_${new Date().toISOString().split('T')[0]}.csv`;
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
}

// Keyboard shortcuts
document.addEventListener('keydown', function(event) {
    // Ctrl/Cmd + F to focus search
    if ((event.ctrlKey || event.metaKey) && event.key === 'f') {
        event.preventDefault();
        const searchInput = document.getElementById('search');
        if (searchInput) {
            searchInput.focus();
            searchInput.select();
        }
    }

    // Escape to close menus
    if (event.key === 'Escape') {
        // Close action menus
        const allActionMenus = document.querySelectorAll('[id^="actions-"]');
        allActionMenus.forEach(menu => {
            menu.classList.add('hidden');
        });
    }
});

// Utility functions for pagination
function minInt(a, b) {
    return Math.min(a, b);
}

function maxInt(a, b) {
    return Math.max(a, b);
}

// Initialize tooltips (if you want to add them)
function initializeTooltips() {
    const elementsWithTooltips = document.querySelectorAll('[title]');
    elementsWithTooltips.forEach(element => {
        // You can add tooltip library initialization here
        // For example, if using a tooltip library
    });
}

// Call initialization
initializeTooltips();