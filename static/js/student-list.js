// Student List Management JavaScript - Fixed Version

/**
 * Student List Management System
 * Handles search, filtering, document management, and various student operations
 */
class StudentListManager {
    constructor() {
        this.isLoading = false;
        this.currentStudentId = null;
        this.searchTimeout = null;
        this.abortController = null;

        // Bind methods to preserve context
        this.handleSearch = this.handleSearch.bind(this);
        this.handleClickOutside = this.handleClickOutside.bind(this);
        this.handleKeyboardShortcuts = this.handleKeyboardShortcuts.bind(this);

        this.init();
    }

    /**
     * Initialize the student list functionality
     */
    init() {
        if (document.readyState === 'loading') {
            document.addEventListener('DOMContentLoaded', () => this.setupEventListeners());
        } else {
            this.setupEventListeners();
        }
    }

    /**
     * Setup all event listeners
     */
    setupEventListeners() {
        this.setupSearchFunctionality();
        this.setupFilterFunctionality();
        this.setupClickOutsideHandlers();
        this.setupKeyboardShortcuts();
        this.loadAllDocuments();
    }

    /**
     * Enhanced search functionality with better error handling
     */
    setupSearchFunctionality() {
        const searchInput = document.getElementById('search');
        if (!searchInput) return;

        // Clear any existing listeners to prevent duplicates
        searchInput.removeEventListener('input', this.handleSearch);
        searchInput.removeEventListener('keydown', this.handleKeyPress);

        // Prevent form submission on Enter
        searchInput.addEventListener('keydown', (e) => {
            if (e.key === 'Enter') {
                e.preventDefault();
                this.performSearch(searchInput.value.trim());
            }
        });

        // Handle input with debouncing
        searchInput.addEventListener('input', (e) => {
            if (this.searchTimeout) {
                clearTimeout(this.searchTimeout);
            }

            this.searchTimeout = setTimeout(() => {
                // Only auto-search if user typed more than 2 characters or cleared the field
                const value = e.target.value.trim();
                if (value.length > 2 || value.length === 0) {
                    this.performSearch(value);
                }
            }, 500);
        });

        // Prevent losing focus
        searchInput.addEventListener('blur', (e) => {
            // Prevent blur if clicking on search-related elements
            const relatedTarget = e.relatedTarget;
            if (relatedTarget && relatedTarget.closest('.search-container')) {
                setTimeout(() => searchInput.focus(), 10);
            }
        });

        // Set initial value from URL if present
        const urlParams = new URLSearchParams(window.location.search);
        const searchTerm = urlParams.get('search');
        if (searchTerm) {
            searchInput.value = searchTerm;
        }
    }

    /**
     * Handle search input with debouncing (for backward compatibility)
     */
    handleSearch(event) {
        const searchValue = event.target.value.trim();

        // Clear existing timeout
        if (this.searchTimeout) {
            clearTimeout(this.searchTimeout);
        }

        // Debounce search
        this.searchTimeout = setTimeout(() => {
            this.performSearch(searchValue);
        }, 300);
    }

    /**
     * Handle Enter key press in search
     */
    handleKeyPress(event) {
        if (event.key === 'Enter') {
            event.preventDefault();
            if (this.searchTimeout) {
                clearTimeout(this.searchTimeout);
            }
            this.performSearch(event.target.value.trim());
        }
    }

    /**
     * Perform search operation with URL update
     */
    performSearch(searchTerm) {
        try {
            const url = new URL(window.location);

            if (searchTerm) {
                url.searchParams.set('search', searchTerm);
            } else {
                url.searchParams.delete('search');
            }

            url.searchParams.set('page', '1'); // Reset to first page

            // Update URL and reload
            window.location.href = url.toString();
        } catch (error) {
            console.error('Error performing search:', error);
            this.showErrorMessage('Search failed. Please try again.');
        }
    }

    /**
     * Setup filter functionality
     */
    setupFilterFunctionality() {
        const filterForm = document.getElementById('filterForm');
        if (!filterForm) return;

        filterForm.addEventListener('submit', (e) => {
            e.preventDefault();
            this.submitForm();
        });

        // Setup individual filter changes
        const filterSelects = filterForm.querySelectorAll('select');
        filterSelects.forEach(select => {
            select.addEventListener('change', () => this.submitForm());
        });
    }

    /**
     * Submit filter form
     */
    submitForm() {
        const filterForm = document.getElementById('filterForm');
        if (!filterForm) return;

        try {
            filterForm.submit();
        } catch (error) {
            console.error('Error submitting form:', error);
            this.showErrorMessage('Filter update failed.');
        }
    }

    /**
     * Clear all filters and search
     */
    clearFilters() {
        try {
            const filterForm = document.getElementById('filterForm');
            const searchInput = document.getElementById('search');

            if (filterForm) filterForm.reset();
            if (searchInput) searchInput.value = '';

            const url = new URL(window.location);
            url.search = '';
            window.location.href = url.toString();
        } catch (error) {
            console.error('Error clearing filters:', error);
            this.showErrorMessage('Failed to clear filters.');
        }
    }

    /**
     * Load documents for all students
     */
    async loadAllDocuments() {
        const docCells = document.querySelectorAll('[id^="docs-"]');
        const userRole = this.getCurrentUserRole();

        // Load documents concurrently with error handling
        const loadPromises = Array.from(docCells).map(cell => {
            const studentId = parseInt(cell.id.replace('docs-', ''));
            return this.loadDocuments(studentId, userRole).catch(error => {
                console.error(`Failed to load documents for student ${studentId}:`, error);
                return null; // Continue with other requests
            });
        });

        try {
            await Promise.allSettled(loadPromises);
        } catch (error) {
            console.error('Error loading documents:', error);
        }
    }

    /**
     * Load documents for a specific student with enhanced error handling
     */
    async loadDocuments(studentId, userRole) {
        const container = document.getElementById(`docs-${studentId}`);
        if (!container) return;

        // Show loading state
        container.innerHTML = '<div class="text-xs text-muted-foreground italic">Kraunama...</div>';

        // Abort previous request if exists
        if (this.abortController) {
            this.abortController.abort();
        }
        this.abortController = new AbortController();

        try {
            const response = await fetch(`/api/students/${studentId}/documents`, {
                method: 'GET',
                headers: {
                    'Content-Type': 'application/json',
                },
                credentials: 'same-origin',
                signal: this.abortController.signal
            });

            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }

            const data = await response.json();
            this.displayDocuments(container, data.documents || [], userRole);
        } catch (error) {
            if (error.name === 'AbortError') {
                return; // Request was cancelled
            }

            console.error('Error loading documents:', error);
            container.innerHTML = '<div class="text-xs text-red-500">Klaida kraunant dokumentus</div>';
        }
    }

    /**
     * Display documents with improved structure
     */
    displayDocuments(container, documents, userRole) {
        if (!documents || documents.length === 0) {
            container.innerHTML = '<div class="text-xs text-muted-foreground">Nėra dokumentų</div>';
            return;
        }

        // Document type mapping with better organization
        const documentTypes = {
            'VIDEO': { icon: '🎥', label: 'Video', priority: 1 },
            'RECOMMENDATION.PDF': { icon: '📋', label: 'Rekomendacija', priority: 2 },
            'FINAL_THESIS.PDF': { icon: '📄', label: 'Darbas', priority: 3 },
            'SOURCE_CODE': { icon: '💻', label: 'Kodas', priority: 4 }
        };

        // Sort documents by priority
        const sortedDocuments = documents.sort((a, b) => {
            const aPriority = documentTypes[a.type]?.priority || 999;
            const bPriority = documentTypes[b.type]?.priority || 999;
            return aPriority - bPriority;
        });

        const documentsHtml = sortedDocuments.map(doc => {
            const docType = documentTypes[doc.type] || { icon: '📎', label: doc.type };
            const statusClass = this.getDocumentStatusClass(doc.status);
            const statusText = this.getDocumentStatusText(doc.status);

            return `
                <div class="flex items-center justify-between text-xs border rounded p-1 hover:bg-muted/50 transition-colors">
                    <div class="flex items-center gap-1 flex-1 min-w-0">
                        <span class="text-sm" aria-label="${docType.label}">${docType.icon}</span>
                        <span class="truncate text-xs font-medium" title="${doc.name || docType.label}">
                            ${docType.label}
                        </span>
                    </div>
                    <div class="flex items-center gap-1">
                        <span class="inline-flex items-center rounded px-1 py-0.5 text-xs font-medium ${statusClass}">
                            ${statusText}
                        </span>
                        ${this.canViewDocument(userRole, doc) ? `
                            <button onclick="studentListManager.viewDocument(${doc.id})" 
                                    class="text-xs text-blue-600 hover:text-blue-800 p-1 rounded hover:bg-blue-50 transition-colors"
                                    title="Žiūrėti dokumentą">
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

    /**
     * New function for viewing topic registration form
     */
    viewTopicRegistrationForm(studentId) {
        if (this.isLoading) return;
        window.open(`/students/${studentId}/topic-registration-form`, '_blank');
    }

    /**
     * Enhanced API request handling with better error management
     */
    async performAction(url, method, data = null) {
        if (this.isLoading) {
            this.showErrorMessage('Palaukite, kol baigsis ankstesnis veiksmas.');
            return Promise.reject(new Error('Request in progress'));
        }

        this.isLoading = true;

        const options = {
            method: method,
            headers: {
                'Content-Type': 'application/json',
                'X-Requested-With': 'XMLHttpRequest' // CSRF protection
            },
            credentials: 'same-origin'
        };

        if (data && (method === 'POST' || method === 'PUT' || method === 'PATCH')) {
            options.body = JSON.stringify(data);
        }

        try {
            const response = await fetch(url, options);

            if (!response.ok) {
                let errorMessage = `HTTP error! status: ${response.status}`;
                try {
                    const errorData = await response.json();
                    errorMessage = errorData.message || errorMessage;
                } catch (e) {
                    // Response is not JSON, use status text
                    errorMessage = response.statusText || errorMessage;
                }
                throw new Error(errorMessage);
            }

            const result = await response.json();
            return result;
        } catch (error) {
            console.error('API request failed:', error);
            throw error;
        } finally {
            this.isLoading = false;
        }
    }

    /**
     * Enhanced workflow actions with better UX
     */
    async requestChanges(studentId) {
        if (this.isLoading) return;

        const reason = prompt('Įveskite pakeitimų priežastį:');
        if (!reason?.trim()) return;

        try {
            await this.performAction(`/api/students/${studentId}/request-changes`, 'POST', {
                reason: reason.trim()
            });

            this.showSuccessMessage('Pakeitimai sėkmingai pateikti');
            setTimeout(() => window.location.reload(), 1500);
        } catch (error) {
            this.showErrorMessage('Klaida prašant pakeitimų: ' + error.message);
        }
    }

    async approveRegistration(studentId) {
        if (this.isLoading) return;

        if (!confirm('Ar tikrai norite patvirtinti šią temą?')) return;

        try {
            await this.performAction(`/api/students/${studentId}/topic/approve`, 'POST');
            this.showSuccessMessage('Tema sėkmingai patvirtinta');
            setTimeout(() => window.location.reload(), 1500);
        } catch (error) {
            this.showErrorMessage('Klaida tvirtinant temą: ' + error.message);
        }
    }

    async rejectRegistration(studentId) {
        if (this.isLoading) return;

        const reason = prompt('Įveskite atmetimo priežastį:');
        if (!reason?.trim()) return;

        try {
            await this.performAction(`/api/students/${studentId}/topic/reject`, 'POST', {
                reason: reason.trim()
            });

            this.showSuccessMessage('Tema atmesta');
            setTimeout(() => window.location.reload(), 1500);
        } catch (error) {
            this.showErrorMessage('Klaida atmetant temą: ' + error.message);
        }
    }

    /**
     * Navigation methods with loading states
     */
    viewRegistration(studentId) {
        if (this.isLoading) return;
        this.navigateWithLoading(`/students/${studentId}/topic/view`);
    }

    editRegistration(studentId) {
        if (this.isLoading) return;
        this.navigateWithLoading(`/students/${studentId}/topic/edit`);
    }

    uploadDocument(studentId) {
        if (this.isLoading) return;
        this.navigateWithLoading(`/students/${studentId}/documents/upload`);
    }

    viewDocument(documentId) {
        if (this.isLoading) return;
        window.open(`/api/documents/${documentId}/view`, '_blank');
    }

    viewReview(studentId) {
        if (this.isLoading) return;
        this.navigateWithLoading(`/students/${studentId}/review/view`);
    }

    editReview(studentId) {
        if (this.isLoading) return;
        this.navigateWithLoading(`/students/${studentId}/review/edit`);
    }

    assignReviewer(studentId) {
        if (this.isLoading) return;
        this.navigateWithLoading(`/students/${studentId}/assign-reviewer`);
    }

    viewReviewerReport(studentId) {
        if (this.isLoading) return;
        window.open(`/students/${studentId}/reviewer-report`, '_blank');
    }

    viewSupervisorReport(studentId) {
        if (this.isLoading) return;
        window.open(`/students/${studentId}/supervisor-report`, '_blank');
    }

    editSupervisorReport(studentId) {
        if (this.isLoading) return;
        this.navigateWithLoading(`/students/${studentId}/supervisor-report/edit`);
    }

    editStudent(studentId) {
        if (this.isLoading) return;
        this.navigateWithLoading(`/students/edit/${studentId}`);
    }

    async deleteStudent(studentId) {
        if (this.isLoading) return;

        if (!confirm('Ar tikrai norite ištrinti šį studentą? Šis veiksmas negrįžtamas.')) return;

        try {
            await this.performAction(`/api/students/${studentId}`, 'DELETE');
            this.showSuccessMessage('Studentas sėkmingai ištrintas');
            setTimeout(() => window.location.reload(), 1500);
        } catch (error) {
            this.showErrorMessage('Klaida trinant studentą: ' + error.message);
        }
    }

    navigateWithLoading(url) {
        this.isLoading = true;
        window.location.href = url;
    }

    /**
     * Enhanced click outside handling
     */
    setupClickOutsideHandlers() {
        document.addEventListener('click', this.handleClickOutside);
    }

    handleClickOutside(event) {
        // Close action menus when clicking outside
        if (!event.target.closest('[id^="actions-"]') &&
            !event.target.closest('button[onclick^="toggleActions"]')) {
            this.closeAllActionMenus();
        }
    }

    closeAllActionMenus() {
        const allActionMenus = document.querySelectorAll('[id^="actions-"]');
        allActionMenus.forEach(menu => {
            menu.classList.add('hidden');
        });
    }

    toggleActions(studentId) {
        this.closeAllActionMenus();
        const menu = document.getElementById(`actions-${studentId}`);
        if (menu) {
            menu.classList.toggle('hidden');
        }
    }

    /**
     * Keyboard shortcuts
     */
    setupKeyboardShortcuts() {
        document.addEventListener('keydown', this.handleKeyboardShortcuts);
    }

    handleKeyboardShortcuts(event) {
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
            this.closeAllActionMenus();
        }
    }

    /**
     * Utility methods
     */
    getCurrentUserRole() {
        // Try multiple sources for user role
        const userRoleElement = document.querySelector('[data-user-role]');
        if (userRoleElement) {
            return userRoleElement.getAttribute('data-user-role');
        }

        if (typeof window.currentUserRole !== 'undefined') {
            return window.currentUserRole;
        }

        return 'student'; // Default fallback
    }

    getDocumentStatusClass(status) {
        const statusClasses = {
            'approved': 'bg-green-100 text-green-800 ring-1 ring-green-600/20',
            'pending': 'bg-yellow-100 text-yellow-800 ring-1 ring-yellow-600/20',
            'rejected': 'bg-red-100 text-red-800 ring-1 ring-red-600/20',
            'draft': 'bg-gray-100 text-gray-800 ring-1 ring-gray-600/20'
        };
        return statusClasses[status] || statusClasses.draft;
    }

    getDocumentStatusText(status) {
        const statusTexts = {
            'approved': 'Patvirtinta',
            'pending': 'Laukia',
            'rejected': 'Atmesta',
            'draft': 'Juodraštis'
        };
        return statusTexts[status] || statusTexts.draft;
    }

    canViewDocument(userRole, document) {
        const allowedRoles = ['admin', 'department_head', 'supervisor'];
        return allowedRoles.includes(userRole) || document.status === 'approved';
    }

    /**
     * Enhanced notification system
     */
    showSuccessMessage(message) {
        this.showNotification(message, 'success');
    }

    showErrorMessage(message) {
        this.showNotification(message, 'error');
    }

    showNotification(message, type = 'info') {
        // Remove existing notifications of the same type
        const existingNotifications = document.querySelectorAll(`.notification-${type}`);
        existingNotifications.forEach(notification => notification.remove());

        const notification = document.createElement('div');
        notification.className = `notification-${type} fixed top-4 right-4 z-50 max-w-sm p-4 rounded-lg shadow-lg transition-all duration-300 transform translate-x-full`;

        const typeClasses = {
            'success': 'bg-green-500 text-white',
            'error': 'bg-red-500 text-white',
            'info': 'bg-blue-500 text-white',
            'warning': 'bg-yellow-500 text-black'
        };

        notification.classList.add(...typeClasses[type].split(' '));
        notification.setAttribute('role', 'alert');
        notification.setAttribute('aria-live', 'polite');

        notification.innerHTML = `
            <div class="flex items-center justify-between">
                <span class="text-sm font-medium">${message}</span>
                <button onclick="this.parentElement.parentElement.remove()" 
                        class="ml-2 ${type === 'warning' ? 'text-black hover:text-gray-700' : 'text-white hover:text-gray-200'}"
                        aria-label="Uždaryti pranešimą">
                    <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/>
                    </svg>
                </button>
            </div>
        `;

        document.body.appendChild(notification);

        // Animate in
        requestAnimationFrame(() => {
            notification.classList.remove('translate-x-full');
        });

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

    /**
     * Export functionality
     */
    async exportData() {
        if (this.isLoading) return;

        try {
            this.isLoading = true;
            const url = new URL('/api/students/export', window.location.origin);

            // Add current filters to export
            const urlParams = new URLSearchParams(window.location.search);
            for (const [key, value] of urlParams) {
                if (key !== 'page') {
                    url.searchParams.set(key, value);
                }
            }

            const response = await fetch(url.toString(), {
                method: 'GET',
                credentials: 'same-origin'
            });

            if (!response.ok) {
                throw new Error('Export failed');
            }

            // Create download
            const blob = await response.blob();
            const downloadUrl = window.URL.createObjectURL(blob);
            const link = document.createElement('a');
            link.href = downloadUrl;
            link.download = `students_export_${new Date().toISOString().split('T')[0]}.csv`;
            document.body.appendChild(link);
            link.click();
            document.body.removeChild(link);
            window.URL.revokeObjectURL(downloadUrl);

            this.showSuccessMessage('Duomenys sėkmingai eksportuoti');
        } catch (error) {
            console.error('Export failed:', error);
            this.showErrorMessage('Eksportavimo klaida: ' + error.message);
        } finally {
            this.isLoading = false;
        }
    }

    /**
     * Cleanup method
     */
    destroy() {
        // Remove event listeners
        document.removeEventListener('click', this.handleClickOutside);
        document.removeEventListener('keydown', this.handleKeyboardShortcuts);

        // Clear timeouts
        if (this.searchTimeout) {
            clearTimeout(this.searchTimeout);
        }

        // Abort any pending requests
        if (this.abortController) {
            this.abortController.abort();
        }
    }
}

// Create global instance
const studentListManager = new StudentListManager();

// Global functions for backward compatibility (called from HTML onclick attributes)
function toggleFilters() {
    // This function was missing - filters are now always visible
    console.log('Filters are always visible now');
}

function clearFilters() {
    studentListManager.clearFilters();
}

function submitForm() {
    studentListManager.submitForm();
}

function handleSearch(event) {
    studentListManager.handleSearch(event);
}

function requestChanges(studentId) {
    studentListManager.requestChanges(studentId);
}

function viewTopicRegistrationForm(studentId) {
    studentListManager.viewTopicRegistrationForm(studentId);
}

function viewReviewerReport(studentId) {
    studentListManager.viewReviewerReport(studentId);
}

function viewSupervisorReport(studentId) {
    studentListManager.viewSupervisorReport(studentId);
}

function loadDocuments(studentId, userRole) {
    studentListManager.loadDocuments(studentId, userRole);
}

function viewRegistration(studentId) {
    studentListManager.viewRegistration(studentId);
}

function editRegistration(studentId) {
    studentListManager.editRegistration(studentId);
}

function approveRegistration(studentId) {
    studentListManager.approveRegistration(studentId);
}

function rejectRegistration(studentId) {
    studentListManager.rejectRegistration(studentId);
}

function uploadDocument(studentId) {
    studentListManager.uploadDocument(studentId);
}

function viewDocument(documentId) {
    studentListManager.viewDocument(documentId);
}

function viewReview(studentId) {
    studentListManager.viewReview(studentId);
}

function editReview(studentId) {
    studentListManager.editReview(studentId);
}

function assignReviewer(studentId) {
    studentListManager.assignReviewer(studentId);
}

function editSupervisorReport(studentId) {
    studentListManager.editSupervisorReport(studentId);
}

function toggleActions(studentId) {
    studentListManager.toggleActions(studentId);
}

function editStudent(studentId) {
    studentListManager.editStudent(studentId);
}

function deleteStudent(studentId) {
    studentListManager.deleteStudent(studentId);
}

function exportData() {
    studentListManager.exportData();
}

// Cleanup on page unload
window.addEventListener('beforeunload', () => {
    if (studentListManager) {
        studentListManager.destroy();
    }
});


// In your static/js/student-list.js or layout
document.addEventListener('DOMContentLoaded', function() {
    // Load documents for all cells that need it
    document.querySelectorAll('[data-load-documents="true"]').forEach(function(element) {
        const studentId = parseInt(element.dataset.studentId);
        const userRole = element.dataset.userRole;

        if (typeof loadDocuments === 'function') {
            loadDocuments(studentId, userRole);
        }
    });
});

// Also handle HTMX updates
document.body.addEventListener('htmx:afterSwap', function() {
    document.querySelectorAll('[data-load-documents="true"]').forEach(function(element) {
        const studentId = parseInt(element.dataset.studentId);
        const userRole = element.dataset.userRole;

        if (typeof loadDocuments === 'function') {
            loadDocuments(studentId, userRole);
        }
    });
});