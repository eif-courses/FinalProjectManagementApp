// Repository Preview JavaScript
document.addEventListener('DOMContentLoaded', function() {
    const repoData = document.getElementById('repo-data');
    if (repoData) {
        const studentId = repoData.getAttribute('data-student-id');
        if (studentId) {
            loadRepositoryPreview(studentId);
        }
    }

    // Setup upload form if it exists
    const uploadForm = document.getElementById('source-upload-form');
    if (uploadForm) {
        setupUploadForm();
    }
});

async function loadRepositoryPreview(studentId) {
    try {
        const response = await fetch(`/api/repository/student/${studentId}`);
        if (!response.ok) {
            throw new Error('Repository not found');
        }

        const data = await response.json();
        displayRepositoryPreview(data);
    } catch (error) {
        console.error('Error loading repository:', error);
        displayRepositoryError(error.message);
    }
}

function displayRepositoryPreview(data) {
    const container = document.getElementById('repo-tree-preview');
    const fileCountEl = document.getElementById('file-count');
    const totalFilesEl = document.getElementById('total-files');
    const totalSizeEl = document.getElementById('total-size');
    const commitCountEl = document.getElementById('commit-count');
    const languageTagsEl = document.getElementById('language-tags');

    if (data.error) {
        displayRepositoryError(data.error);
        return;
    }

    const files = data.contents?.files || [];
    const stats = data.contents?.stats || {};

    // Update file count
    fileCountEl.textContent = `(${files.length} items)`;

    // Update stats
    totalFilesEl.textContent = stats.total_files || files.length;
    totalSizeEl.textContent = formatFileSize(stats.total_size || 0);
    commitCountEl.textContent = stats.commit_count || 0;

    // Create file tree
    const treeHtml = createFileTree(files.slice(0, 10)); // Show first 10 files
    container.innerHTML = treeHtml;

    // Add language tags
    if (stats.languages) {
        const languageHtml = Object.entries(stats.languages)
            .slice(0, 5) // Show top 5 languages
            .map(([lang, count]) =>
                `<span class="inline-flex items-center px-2 py-1 rounded text-xs font-medium bg-blue-100 text-blue-800">
                    ${lang} (${count})
                </span>`
            ).join('');
        languageTagsEl.innerHTML = languageHtml;
    }

    // Add "show more" if there are more files
    if (files.length > 10) {
        const showMoreEl = document.createElement('div');
        showMoreEl.className = 'px-4 py-2 border-t bg-gray-50 text-center';
        showMoreEl.innerHTML = `
            <button onclick="viewFullRepository(${data.repository?.student_record_id || studentId})" 
                    class="text-sm text-blue-600 hover:text-blue-800">
                View all ${files.length} files →
            </button>
        `;
        container.appendChild(showMoreEl);
    }
}

function createFileTree(files) {
    if (!files || files.length === 0) {
        return `
            <div class="p-4 text-center text-gray-500">
                <div class="text-sm">No files found</div>
            </div>
        `;
    }

    const fileHtml = files.map(file => {
        const icon = getFileIcon(file.name, file.type);
        const sizeText = file.type === 'file' ? formatFileSize(file.size) : '';

        return `
            <div class="flex items-center justify-between px-4 py-2 hover:bg-gray-50 border-b border-gray-100 last:border-b-0">
                <div class="flex items-center space-x-2 flex-1 min-w-0">
                    <span class="text-base">${icon}</span>
                    <span class="text-sm font-medium text-gray-900 truncate">${file.name}</span>
                    ${file.language ? `<span class="text-xs text-gray-500">${file.language}</span>` : ''}
                </div>
                <div class="text-xs text-gray-500 ml-2">
                    ${sizeText}
                </div>
            </div>
        `;
    }).join('');

    return fileHtml;
}

function displayRepositoryError(errorMessage) {
    const container = document.getElementById('repo-tree-preview');
    container.innerHTML = `
        <div class="p-4 text-center text-red-500">
            <div class="text-sm">${errorMessage}</div>
            <div class="text-xs text-gray-500 mt-1">Repository may not be accessible</div>
        </div>
    `;

    // Update stats to show error state
    document.getElementById('file-count').textContent = '(Error)';
    document.getElementById('total-files').textContent = '-';
    document.getElementById('total-size').textContent = '-';
    document.getElementById('commit-count').textContent = '-';
}

function getFileIcon(filename, type) {
    if (type === 'dir') return '📁';

    const ext = filename.split('.').pop()?.toLowerCase();
    const icons = {
        'js': '💛', 'ts': '💙', 'jsx': '⚛️', 'tsx': '⚛️',
        'py': '🐍', 'go': '🐹', 'java': '☕', 'cpp': '⚙️', 'c': '⚙️',
        'html': '🌐', 'css': '🎨', 'scss': '🎨', 'sass': '🎨',
        'json': '📋', 'xml': '📄', 'yaml': '📄', 'yml': '📄',
        'md': '📝', 'txt': '📄', 'pdf': '📕', 'zip': '📦',
        'png': '🖼️', 'jpg': '🖼️', 'jpeg': '🖼️', 'gif': '🖼️', 'svg': '🎨',
        'sql': '🗄️', 'dockerfile': '🐳', 'gitignore': '🙈'
    };

    if (filename.toLowerCase().includes('readme')) return '📖';
    if (filename.toLowerCase().includes('license')) return '📜';

    return icons[ext] || '📄';
}

function formatFileSize(bytes) {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
}

function viewFullRepository(studentId) {
    if (studentId && studentId !== 'null') {
        window.open(`/repository/student/${studentId}`, '_blank');
    }
}

function uploadNewVersion() {
    if (confirm('Upload a new version of your source code? This will create a new commit in your repository.')) {
        // Redirect to upload page
        window.location.href = '/test-upload';
    }
}

function setupUploadForm() {
    const uploadForm = document.getElementById('source-upload-form');
    if (!uploadForm) return;

    uploadForm.addEventListener('submit', async function(e) {
        e.preventDefault();

        const formData = new FormData(this);
        const uploadButton = document.getElementById('upload-button');
        const progressContainer = document.getElementById('upload-progress');
        const progressBar = document.getElementById('progress-bar');
        const statusText = document.getElementById('upload-status');

        // Show progress, hide form
        uploadButton.disabled = true;
        uploadButton.textContent = 'Uploading...';
        progressContainer.classList.remove('hidden');

        try {
            const response = await fetch('/api/source-code/upload', {
                method: 'POST',
                body: formData
            });

            const result = await response.json();

            if (result.success) {
                // Simulate progress for better UX
                let progress = 0;
                const interval = setInterval(() => {
                    progress += 10;
                    progressBar.style.width = progress + '%';

                    if (progress >= 100) {
                        clearInterval(interval);
                        statusText.textContent = 'Upload completed successfully!';
                        statusText.className = 'text-sm text-green-600';
                        setTimeout(() => {
                            window.location.reload();
                        }, 1500);
                    }
                }, 200);
            } else {
                throw new Error(result.error || 'Upload failed');
            }
        } catch (error) {
            console.error('Upload error:', error);
            statusText.textContent = 'Upload failed: ' + error.message;
            statusText.className = 'text-sm text-red-600';
            uploadButton.disabled = false;
            uploadButton.textContent = 'Upload Source Code';
            progressContainer.classList.add('hidden');
        }
    });
}