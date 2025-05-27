// static/js/supervisor-report.js
function updateTotalSimilarity() {
    const otherMatch = parseFloat(document.querySelector('input[name="other_match"]').value) || 0;
    const oneMatch = parseFloat(document.querySelector('input[name="one_match"]').value) || 0;
    const ownMatch = parseFloat(document.querySelector('input[name="own_match"]').value) || 0;
    const joinMatch = parseFloat(document.querySelector('input[name="join_match"]').value) || 0;

    const total = otherMatch + oneMatch + ownMatch + joinMatch;

    // Update total display
    const totalElement = document.getElementById('total-similarity');
    if (totalElement) {
        totalElement.textContent = total.toFixed(1) + '%';
        totalElement.setAttribute('data-similarity', total.toFixed(1));
    }

    // Update status assessment
    const statusElement = document.getElementById('similarity-status');
    if (statusElement) {
        let status, colorClass;
        if (total <= 15) {
            status = 'Low similarity';
            colorClass = 'text-green-600';
        } else if (total <= 25) {
            status = 'Moderate similarity';
            colorClass = 'text-yellow-600';
        } else {
            status = 'High similarity';
            colorClass = 'text-red-600';
        }

        statusElement.textContent = status;
        statusElement.className = 'text-sm font-medium ' + colorClass;
    }
}

// Character counter for comments
document.addEventListener('DOMContentLoaded', function() {
    const commentTextarea = document.querySelector('textarea[name="supervisor_comments"]');
    const counter = document.getElementById('comment-counter');

    if (commentTextarea && counter) {
        commentTextarea.addEventListener('input', function() {
            counter.textContent = this.value.length;
        });
    }
});

// Form validation before submit
document.addEventListener('htmx:beforeRequest', function(event) {
    if (event.detail.elt.tagName === 'FORM') {
        const form = event.detail.elt;
        const comments = form.querySelector('textarea[name="supervisor_comments"]').value;
        const workplace = form.querySelector('input[name="supervisor_workplace"]').value;
        const position = form.querySelector('input[name="supervisor_position"]').value;

        if (comments.length < 50) {
            alert('Supervisor comments must be at least 50 characters long.');
            event.preventDefault();
            return false;
        }

        if (!workplace.trim()) {
            alert('Workplace is required.');
            event.preventDefault();
            return false;
        }

        if (!position.trim()) {
            alert('Position is required.');
            event.preventDefault();
            return false;
        }
    }
});