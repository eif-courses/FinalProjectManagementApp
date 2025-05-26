// handlers/upload.go - Fixed file upload handler
package handlers

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"FinalProjectManagementApp/auth"
	"FinalProjectManagementApp/files"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	msgraph "github.com/microsoftgraph/msgraph-sdk-go"
)

// UploadFileHandler handles file uploads to SharePoint with enhanced error handling
func UploadFileHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		renderError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse multipart form (32MB max)
	err := r.ParseMultipartForm(32 << 20)
	if err != nil {
		renderError(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		renderError(w, "Failed to get file from form", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Validate file
	if header.Size == 0 {
		renderError(w, "File is empty", http.StatusBadRequest)
		return
	}

	if header.Size > 250*1024*1024 { // 250MB limit
		renderError(w, "File too large (max 250MB)", http.StatusBadRequest)
		return
	}

	// Get authenticated user
	user := auth.GetUserFromContext(r.Context())
	if user == nil {
		renderError(w, "User not authenticated", http.StatusUnauthorized)
		return
	}

	// Validate file type (basic security)
	if !isAllowedFileType(header.Filename) {
		renderError(w, "File type not allowed", http.StatusBadRequest)
		return
	}

	// Render initial upload progress
	renderProgress(w, "Initializing upload...", 10)
	w.(http.Flusher).Flush()

	// Create Graph client with enhanced error handling
	graphClient, err := createGraphClient()
	if err != nil {
		renderError(w, fmt.Sprintf("Failed to create Graph client: %v", err), http.StatusInternalServerError)
		return
	}

	renderProgress(w, "Connecting to SharePoint...", 25)
	w.(http.Flusher).Flush()

	// SharePoint site configuration - make these configurable
	siteURL := os.Getenv("SHAREPOINT_SITE_URL")
	if siteURL == "" {
		siteURL = "https://vikolt.sharepoint.com/sites/thesis_management-O365G"
	}

	// Get SharePoint site ID
	siteID, err := files.GetSiteID(r.Context(), graphClient, siteURL)
	if err != nil {
		renderError(w, fmt.Sprintf("Failed to get SharePoint site: %v", err), http.StatusInternalServerError)
		return
	}

	renderProgress(w, "Getting document library...", 40)
	w.(http.Flusher).Flush()

	// Get default drive ID (Documents library)
	driveID, err := files.GetDocumentLibraryDriveID(r.Context(), graphClient, siteID)
	if err != nil {
		renderError(w, fmt.Sprintf("Failed to get document library: %v", err), http.StatusInternalServerError)
		return
	}

	// Create SharePoint service
	spService := files.NewSharePointService(graphClient, siteID, driveID)

	// Create safe folder structure based on user
	department := sanitizeFolderName(user.Department)
	if department == "" {
		department = "General"
	}
	userEmail := sanitizeFolderName(user.Email)
	targetFolder := fmt.Sprintf("uploads/%s/%s", department, userEmail)

	renderProgress(w, "Creating folder structure...", 60)
	w.(http.Flusher).Flush()

	// Create folder if it doesn't exist
	err = spService.CreateFolderPath(r.Context(), targetFolder)
	if err != nil {
		fmt.Printf("Warning: Could not create folder structure '%s': %v\n", targetFolder, err)
		// Continue anyway - folder might already exist
	}

	renderProgress(w, "Preparing file for upload...", 75)
	w.(http.Flusher).Flush()

	// Create temporary file with safe filename
	safeFilename := sanitizeFilename(header.Filename)
	tempFile := filepath.Join(os.TempDir(), fmt.Sprintf("%d_%s", time.Now().Unix(), safeFilename))
	defer func() {
		if err := os.Remove(tempFile); err != nil {
			fmt.Printf("Warning: Failed to remove temp file %s: %v\n", tempFile, err)
		}
	}()

	// Save uploaded file to temp location
	outFile, err := os.Create(tempFile)
	if err != nil {
		renderError(w, "Failed to create temporary file", http.StatusInternalServerError)
		return
	}
	defer outFile.Close()

	_, err = io.Copy(outFile, file)
	if err != nil {
		renderError(w, "Failed to save uploaded file", http.StatusInternalServerError)
		return
	}

	renderProgress(w, "Uploading to SharePoint...", 90)
	w.(http.Flusher).Flush()

	// Upload to SharePoint
	err = spService.UploadFile(r.Context(), tempFile, targetFolder)
	if err != nil {
		renderError(w, fmt.Sprintf("Failed to upload to SharePoint: %v", err), http.StatusInternalServerError)
		return
	}

	// Success response
	sharePointPath := fmt.Sprintf("%s/%s", targetFolder, safeFilename)
	renderSuccess(w, safeFilename, sharePointPath)
}

// Helper functions for HTMX responses
func renderProgress(w http.ResponseWriter, message string, percentage int) {
	html := fmt.Sprintf(`
	<div id="upload-status" class="space-y-4">
		<div class="text-blue-600 text-sm font-medium">%s</div>
		<div class="w-full bg-gray-200 rounded-full h-3">
			<div class="bg-blue-600 h-3 rounded-full transition-all duration-500 ease-out" style="width: %d%%"></div>
		</div>
		<div class="text-xs text-gray-500">%d%% complete</div>
	</div>`, message, percentage, percentage)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(html))
}

func renderError(w http.ResponseWriter, message string, statusCode int) {
	html := fmt.Sprintf(`
	<div id="upload-status" class="space-y-4">
		<div class="p-4 bg-red-50 border border-red-200 rounded-lg">
			<div class="flex items-center">
				<svg class="h-5 w-5 text-red-400 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"></path>
				</svg>
				<div class="text-red-600 text-sm font-medium">Upload Failed</div>
			</div>
			<div class="text-red-600 text-sm mt-1">%s</div>
		</div>
		<button type="button" 
				class="w-full bg-blue-600 text-white py-2 px-4 rounded-md hover:bg-blue-700 transition-colors"
				onclick="resetUploadForm()">
			Try Again
		</button>
	</div>`, message)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(statusCode)
	w.Write([]byte(html))
}

func renderSuccess(w http.ResponseWriter, filename, path string) {
	html := fmt.Sprintf(`
	<div id="upload-status" class="space-y-4">
		<div class="p-4 bg-green-50 border border-green-200 rounded-lg">
			<div class="flex items-center">
				<svg class="h-5 w-5 text-green-400 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"></path>
				</svg>
				<div class="text-green-600 text-sm font-medium">Upload Successful!</div>
			</div>
			<div class="text-xs text-green-600 mt-1">
				<div><strong>File:</strong> %s</div>
				<div><strong>Location:</strong> %s</div>
			</div>
		</div>
		<button type="button" 
				class="w-full bg-blue-600 text-white py-2 px-4 rounded-md hover:bg-blue-700 transition-colors"
				onclick="resetUploadForm()">
			Upload Another File
		</button>
	</div>`, filename, path)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(html))
}

// ShowUploadPage displays the upload form with HTMX
func ShowUploadPage(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/auth/login", http.StatusFound)
		return
	}

	department := user.Department
	if department == "" {
		department = "General"
	}

	html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <title>Upload File to SharePoint</title>
    <link rel="stylesheet" href="/assets/css/output.css">
    <script src="https://unpkg.com/htmx.org@1.9.10"></script>
    <style>
        .upload-area {
            border: 2px dashed #cbd5e0;
            border-radius: 8px;
            padding: 40px;
            text-align: center;
            transition: all 0.3s ease;
            cursor: pointer;
        }
        .upload-area:hover {
            border-color: #4299e1;
            background-color: #f7fafc;
        }
        .upload-area.dragover {
            border-color: #4299e1;
            background-color: #ebf8ff;
            transform: scale(1.02);
        }
        .htmx-request .upload-area {
            opacity: 0.5;
            pointer-events: none;
        }
        .file-info {
            background: #f8f9fa;
            border: 1px solid #e9ecef;
            border-radius: 6px;
            padding: 12px;
        }
    </style>
</head>
<body class="bg-gray-100 min-h-screen">
    <div class="container mx-auto py-8 px-4">
        <div class="max-w-md mx-auto bg-white rounded-lg shadow-md p-6">
            <h1 class="text-2xl font-bold mb-4 text-gray-800">Upload File to SharePoint</h1>
            
            <div class="mb-4 p-4 bg-blue-50 rounded-lg">
                <p class="text-sm text-blue-800"><strong>User:</strong> %s</p>
                <p class="text-sm text-blue-800"><strong>Department:</strong> %s</p>
                <p class="text-sm text-blue-800"><strong>Upload to:</strong> uploads/%s/%s/</p>
            </div>

            <div class="mb-4 p-3 bg-yellow-50 border border-yellow-200 rounded-lg">
                <p class="text-sm text-yellow-800">
                    <strong>Allowed files:</strong> PDF, DOC, DOCX, XLS, XLSX, PPT, PPTX, TXT, ZIP, RAR, JPG, PNG
                </p>
                <p class="text-sm text-yellow-800">
                    <strong>Maximum size:</strong> 250MB
                </p>
            </div>
            
            <form id="uploadForm" 
                  hx-post="/api/upload"
                  hx-target="#upload-status"
                  hx-swap="outerHTML"
                  hx-encoding="multipart/form-data"
                  hx-indicator="#loading"
                  class="space-y-4">
                
                <div class="upload-area" id="uploadArea">
                    <input type="file" id="file" name="file" required class="hidden" accept=".pdf,.doc,.docx,.xls,.xlsx,.ppt,.pptx,.txt,.zip,.rar,.jpg,.jpeg,.png">
                    <div id="uploadText">
                        <svg class="mx-auto h-12 w-12 text-gray-400 mb-4" stroke="currentColor" fill="none" viewBox="0 0 48 48">
                            <path d="M28 8H12a4 4 0 00-4 4v20m32-12v8m0 0v8a4 4 0 01-4 4H12a4 4 0 01-4-4v-4m32-4l-3.172-3.172a4 4 0 00-5.656 0L28 28M8 32l9.172-9.172a4 4 0 015.656 0L28 28m0 0l4 4m4-24h8m-4-4v8m-12 4h.02" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
                        </svg>
                        <p class="text-gray-600 font-medium">Click to select a file or drag and drop</p>
                        <p class="text-sm text-gray-500 mt-1">PDF, DOC, XLS, PPT, Images, Archives</p>
                        <p class="text-sm text-gray-500">Maximum file size: 250MB</p>
                    </div>
                </div>
                
                <div id="fileInfo" class="hidden file-info">
                    <div class="flex items-center justify-between">
                        <div class="flex-1">
                            <p class="text-sm font-medium text-gray-700">Selected file:</p>
                            <p class="text-sm text-gray-600" id="fileName"></p>
                            <p class="text-xs text-gray-500">Size: <span id="fileSize"></span></p>
                        </div>
                        <button type="button" onclick="clearFileSelection()" class="text-red-500 hover:text-red-700">
                            <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"></path>
                            </svg>
                        </button>
                    </div>
                </div>
                
                <button type="submit" id="uploadBtn" 
                        class="w-full bg-blue-600 text-white py-3 px-4 rounded-md hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 disabled:bg-gray-400 disabled:cursor-not-allowed font-medium transition-colors"
                        disabled>
                    Upload to SharePoint
                </button>
            </form>
            
            <div id="upload-status" class="mt-4"></div>
            
            <div id="loading" class="htmx-indicator mt-4">
                <div class="flex items-center justify-center">
                    <div class="animate-spin rounded-full h-6 w-6 border-b-2 border-blue-600"></div>
                    <span class="ml-2 text-sm text-gray-600">Processing...</span>
                </div>
            </div>
            
            <div class="mt-6">
                <a href="/dashboard" class="text-blue-600 hover:text-blue-800 text-sm font-medium">← Back to Dashboard</a>
            </div>
        </div>
    </div>

    <script>
        const uploadArea = document.getElementById('uploadArea');
        const fileInput = document.getElementById('file');
        const fileInfo = document.getElementById('fileInfo');
        const fileName = document.getElementById('fileName');
        const fileSize = document.getElementById('fileSize');
        const uploadBtn = document.getElementById('uploadBtn');

        // Click to select file
        uploadArea.addEventListener('click', () => fileInput.click());

        // File input change
        fileInput.addEventListener('change', handleFileSelect);

        // Drag and drop
        uploadArea.addEventListener('dragover', (e) => {
            e.preventDefault();
            uploadArea.classList.add('dragover');
        });

        uploadArea.addEventListener('dragleave', () => {
            uploadArea.classList.remove('dragover');
        });

        uploadArea.addEventListener('drop', (e) => {
            e.preventDefault();
            uploadArea.classList.remove('dragover');
            const files = e.dataTransfer.files;
            if (files.length > 0) {
                fileInput.files = files;
                handleFileSelect();
            }
        });

        function handleFileSelect() {
            const file = fileInput.files[0];
            if (file) {
                // Validate file size
                if (file.size > 250 * 1024 * 1024) {
                    alert('File is too large. Maximum size is 250MB.');
                    clearFileSelection();
                    return;
                }

                fileName.textContent = file.name;
                fileSize.textContent = formatFileSize(file.size);
                fileInfo.classList.remove('hidden');
                uploadBtn.disabled = false;
            }
        }

        function clearFileSelection() {
            fileInput.value = '';
            fileInfo.classList.add('hidden');
            uploadBtn.disabled = true;
        }

        function formatFileSize(bytes) {
            if (bytes === 0) return '0 Bytes';
            const k = 1024;
            const sizes = ['Bytes', 'KB', 'MB', 'GB'];
            const i = Math.floor(Math.log(bytes) / Math.log(k));
            return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
        }

        function resetUploadForm() {
            clearFileSelection();
            document.getElementById('upload-status').innerHTML = '';
        }

        // HTMX event listeners
        document.addEventListener('htmx:beforeRequest', function(e) {
            if (!fileInput.files[0]) {
                e.preventDefault();
                document.getElementById('upload-status').innerHTML = 
                    '<div class="text-red-600 text-sm">Please select a file first</div>';
                return;
            }
            uploadBtn.disabled = true;
            uploadBtn.textContent = 'Uploading...';
        });

        document.addEventListener('htmx:afterRequest', function(e) {
            uploadBtn.disabled = false;
            uploadBtn.textContent = 'Upload to SharePoint';
        });
    </script>
</body>
</html>`, user.Email, department, department, user.Email)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(html))
}

// createGraphClient creates Graph client with app credentials and required scopes
func createGraphClient() (*msgraph.GraphServiceClient, error) {
	cred, err := azidentity.NewClientSecretCredential(
		os.Getenv("AZURE_TENANT_ID"),
		os.Getenv("AZURE_CLIENT_ID"),
		os.Getenv("AZURE_CLIENT_SECRET"),
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create Azure credentials: %v", err)
	}

	// Create Graph client with required scopes for SharePoint
	graphClient, err := msgraph.NewGraphServiceClientWithCredentials(
		cred,
		[]string{"https://graph.microsoft.com/.default"},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create Graph client: %v", err)
	}

	return graphClient, nil
}

// Security helper functions
func isAllowedFileType(filename string) bool {
	allowedExtensions := []string{
		".pdf", ".doc", ".docx", ".xls", ".xlsx", ".ppt", ".pptx",
		".txt", ".zip", ".rar", ".7z",
		".jpg", ".jpeg", ".png", ".gif", ".bmp",
		".mp4", ".avi", ".mov", ".wmv", // video files
	}

	ext := strings.ToLower(filepath.Ext(filename))
	for _, allowed := range allowedExtensions {
		if ext == allowed {
			return true
		}
	}
	return false
}

func sanitizeFilename(filename string) string {
	// Remove or replace problematic characters
	filename = strings.ReplaceAll(filename, " ", "_")
	filename = strings.ReplaceAll(filename, "(", "")
	filename = strings.ReplaceAll(filename, ")", "")
	filename = strings.ReplaceAll(filename, "[", "")
	filename = strings.ReplaceAll(filename, "]", "")
	filename = strings.ReplaceAll(filename, "&", "and")
	return filename
}

func sanitizeFolderName(name string) string {
	// Remove email domain and clean up folder names
	if strings.Contains(name, "@") {
		name = strings.Split(name, "@")[0]
	}
	name = strings.ReplaceAll(name, " ", "_")
	name = strings.ReplaceAll(name, ".", "_")
	return name
}
