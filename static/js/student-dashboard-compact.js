document.addEventListener('DOMContentLoaded', function() {
    // Compact source code upload
    const sourceForm = document.getElementById('compact-source-form');
    if (sourceForm) {
        sourceForm.addEventListener('submit', async function(e) {
            e.preventDefault();

            const formData = new FormData(this);
            const uploadBtn = document.getElementById('compact-upload-btn');
            const progress = document.getElementById('compact-progress');
            const progressBar = document.getElementById('compact-progress-bar');
            const status = document.getElementById('compact-status');

            uploadBtn.disabled = true;
            uploadBtn.textContent = 'Uploading...';
            progress.classList.remove('hidden');

            try {
                const response = await fetch('/api/source-code/upload', {
                    method: 'POST',
                    body: formData
                });

                const result = await response.json();

                if (result.success) {
                    status.textContent = 'Upload successful! Refreshing...';
                    progressBar.style.width = '100%';
                    setTimeout(() => window.location.reload(), 1500);
                } else {
                    throw new Error(result.error || 'Upload failed');
                }
            } catch (error) {
                alert('Upload failed: ' + error.message);
                uploadBtn.disabled = false;
                uploadBtn.textContent = 'Upload';
                progress.classList.add('hidden');
            }
        });
    }
});

function uploadNewVersion() {
    if (confirm('Upload a new version of your source code?')) {
        window.location.reload();
    }
}

function uploadRecommendation() {
    alert('Recommendation upload feature coming soon!');
}

function uploadVideo() {
    alert('Video upload feature coming soon!');
}

function viewReport(type, id) {
    window.open(`/api/reports/${type}/${id}`, '_blank');
}

function playVideo() {
    alert('Video player coming soon!');
}