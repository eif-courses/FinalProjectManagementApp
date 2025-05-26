// static/js/topic-form.js
document.addEventListener('DOMContentLoaded', function() {
    // Enhanced form validation
    const form = document.getElementById('topic-form');
    if (form) {
        form.addEventListener('submit', function(e) {
            const title = form.querySelector('[name="title"]').value;
            const titleEn = form.querySelector('[name="title_en"]').value;
            const problem = form.querySelector('[name="problem"]').value;
            const objective = form.querySelector('[name="objective"]').value;
            const tasks = form.querySelector('[name="tasks"]').value;

            if (title.length < 10) {
                e.preventDefault();
                showFormError('Title must be at least 10 characters long');
                return;
            }

            if (titleEn.length < 10) {
                e.preventDefault();
                showFormError('English title must be at least 10 characters long');
                return;
            }

            if (problem.length < 50) {
                e.preventDefault();
                showFormError('Problem description must be at least 50 characters long');
                return;
            }
        });
    }

    // Setup auto-save functionality
    setupAutoSave();
});

function showFormError(message) {
    const result = document.getElementById('form-result');
    if (result) {
        result.innerHTML = `
            <div class="rounded-md bg-destructive/15 p-3">
                <div class="flex">
                    <div class="ml-3">
                        <h3 class="text-sm font-medium text-destructive">
                            ${message}
                        </h3>
                    </div>
                </div>
            </div>
        `;
    }
}

// Auto-save draft functionality
let autoSaveTimeout;
function setupAutoSave() {
    const form = document.getElementById('topic-form');
    if (form) {
        const inputs = form.querySelectorAll('input, textarea');
        inputs.forEach(input => {
            input.addEventListener('input', function() {
                clearTimeout(autoSaveTimeout);
                autoSaveTimeout = setTimeout(() => {
                    saveDraft();
                }, 3000); // Save after 3 seconds of inactivity
            });
        });
    }
}

function saveDraft() {
    const form = document.getElementById('topic-form');
    if (form) {
        const formData = new FormData(form);
        formData.append('is_draft', 'true');

        fetch('/api/topic/save-draft', {
            method: 'POST',
            body: formData
        }).then(response => {
            if (response.ok) {
                // Show subtle indicator that draft was saved
                const indicator = document.createElement('div');
                indicator.className = 'fixed top-4 right-4 bg-green-100 text-green-800 px-3 py-1 rounded text-sm z-50';
                indicator.textContent = 'Draft saved';
                document.body.appendChild(indicator);

                setTimeout(() => {
                    indicator.remove();
                }, 2000);
            }
        }).catch(err => {
            console.log('Auto-save failed:', err);
        });
    }
}

// Character counter for textareas
document.addEventListener('DOMContentLoaded', function() {
    const textareas = document.querySelectorAll('textarea[data-min-length]');
    textareas.forEach(textarea => {
        const minLength = parseInt(textarea.dataset.minLength);
        const counter = document.createElement('div');
        counter.className = 'text-xs text-muted-foreground mt-1';
        textarea.parentNode.appendChild(counter);

        function updateCounter() {
            const current = textarea.value.length;
            counter.textContent = `${current}/${minLength} characters minimum`;

            if (current < minLength) {
                counter.className = 'text-xs text-red-500 mt-1';
            } else {
                counter.className = 'text-xs text-green-600 mt-1';
            }
        }

        textarea.addEventListener('input', updateCounter);
        updateCounter();
    });
});