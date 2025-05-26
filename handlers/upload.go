// handlers/upload.go
package handlers

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"FinalProjectManagementApp/auth"
	"FinalProjectManagementApp/files"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	msgraph "github.com/microsoftgraph/msgraph-sdk-go"
)

// UploadFileHandler handles file uploads to SharePoint with HTMX support
func UploadFileHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusMethodNotAllowed)
		w.Write([]byte(`<div class="alert alert-error">Method not allowed</div>`))
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
		renderError(w, "Failed to get file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Get authenticated user
	user := auth.GetUserFromContext(r.Context())
	if user == nil {
		renderError(w, "User not authenticated", http.StatusUnauthorized)
		return
	}

	// Render initial upload progress
	renderProgress(w, "Initializing upload...", 10)
	w.(http.Flusher).Flush()

	// Create Graph client
	graphClient, err := createGraphClient()
	if err != nil {
		renderError(w, fmt.Sprintf("Failed to create Graph client: %v", err), http.StatusInternalServerError)
		return
	}

	renderProgress(w, "Connecting to SharePoint...", 25)
	w.(http.Flusher).Flush()

	// SharePoint site configuration
	siteURL := os.Getenv("SHAREPOINT_SITE_URL")
	if siteURL == "" {
		siteURL = "https://vikolt.sharepoint.com/sites/thesis_management-O365G"
	}

	// Get SharePoint site ID
	siteID, err := files.GetSiteID(r.Context(), graphClient, siteURL)
	if err != nil {
		renderError(w, fmt.Sprintf("Failed to get site ID: %v", err), http.StatusInternalServerError)
		return
	}

	renderProgress(w, "Getting document library...", 40)
	w.(http.Flusher).Flush()

	// Get default drive ID (Documents library)
	driveID, err := files.GetDocumentLibraryDriveID(r.Context(), graphClient, siteID)
	if err != nil {
		renderError(w, fmt.Sprintf("Failed to get drive ID: %v", err), http.StatusInternalServerError)
		return
	}

	// Create SharePoint service
	spService := files.NewSharePointService(graphClient, siteID, driveID)

	// Create folder structure based on user
	department := user.Department
	if department == "" {
		department = "General"
	}
	targetFolder := fmt.Sprintf("uploads/%s/%s", department, user.Email)

	renderProgress(w, "Creating folder structure...", 60)
	w.(http.Flusher).Flush()

	// Create folder if it doesn't exist
	err = spService.CreateFolderPath(r.Context(), targetFolder)
	if err != nil {
		fmt.Printf("Note: Could not create folder (might already exist): %v\n", err)
	}

	renderProgress(w, "Preparing file for upload...", 75)
	w.(http.Flusher).Flush()

	// Save file temporarily
	tempFile := filepath.Join(os.TempDir(), fmt.Sprintf("%d_%s", time.Now().Unix(), header.Filename))
	defer os.Remove(tempFile)

	outFile, err := os.Create(tempFile)
	if err != nil {
		renderError(w, "Failed to create temp file", http.StatusInternalServerError)
		return
	}
	defer outFile.Close()

	_, err = io.Copy(outFile, file)
	if err != nil {
		renderError(w, "Failed to save file", http.StatusInternalServerError)
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
	renderSuccess(w, header.Filename, fmt.Sprintf("%s/%s", targetFolder, header.Filename))
}

// Helper functions for HTMX responses
func renderProgress(w http.ResponseWriter, message string, percentage int) {
	html := fmt.Sprintf(`
	<div id="upload-status" class="space-y-4">
		<div class="text-blue-600 text-sm">%s</div>
		<div class="w-full bg-gray-200 rounded-full h-2">
			<div class="bg-blue-600 h-2 rounded-full transition-all duration-300" style="width: %d%%"></div>
		</div>
		<div class="text-xs text-gray-500">%d%% complete</div>
	</div>`, message, percentage, percentage)

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

func renderError(w http.ResponseWriter, message string, statusCode int) {
	html := fmt.Sprintf(`
	<div id="upload-status" class="space-y-4">
		<div class="p-4 bg-red-50 border border-red-200 rounded-lg">
			<div class="text-red-600 text-sm">❌ %s</div>
		</div>
		<button type="button" 
				class="w-full bg-blue-600 text-white py-2 px-4 rounded-md hover:bg-blue-700"
				onclick="resetUploadForm()">
			Try Again
		</button>
	</div>`, message)

	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(statusCode)
	w.Write([]byte(html))
}

func renderSuccess(w http.ResponseWriter, filename, path string) {
	html := fmt.Sprintf(`
	<div id="upload-status" class="space-y-4">
		<div class="p-4 bg-green-50 border border-green-200 rounded-lg">
			<div class="text-green-600 text-sm">✅ File uploaded successfully!</div>
			<div class="text-xs text-green-600 mt-1">%s → %s</div>
		</div>
		<button type="button" 
				class="w-full bg-blue-600 text-white py-2 px-4 rounded-md hover:bg-blue-700"
				onclick="resetUploadForm()">
			Upload Another File
		</button>
	</div>`, filename, path)

	w.Header().Set("Content-Type", "text/html")
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
            transition: border-color 0.3s;
        }
        .upload-area:hover {
            border-color: #4299e1;
        }
        .upload-area.dragover {
            border-color: #4299e1;
            background-color: #ebf8ff;
        }
        .htmx-request .upload-area {
            opacity: 0.5;
            pointer-events: none;
        }
    </style>
</head>
<body class="bg-gray-100 min-h-screen">
    <div class="container mx-auto py-8 px-4">
        <div class="max-w-md mx-auto bg-white rounded-lg shadow-md p-6">
            <h1 class="text-2xl font-bold mb-4">Upload File to SharePoint</h1>
            
            <div class="mb-4 p-4 bg-blue-50 rounded-lg">
                <p class="text-sm text-blue-800"><strong>User:</strong> %s</p>
                <p class="text-sm text-blue-800"><strong>Department:</strong> %s</p>
                <p class="text-sm text-blue-800"><strong>Upload to:</strong> uploads/%s/%s/</p>
            </div>
            
            <form id="uploadForm" 
                  hx-post="/api/upload"
                  hx-target="#upload-status"
                  hx-swap="outerHTML"
                  hx-encoding="multipart/form-data"
                  hx-indicator="#loading"
                  class="space-y-4">
                
                <div class="upload-area" id="uploadArea">
                    <input type="file" id="file" name="file" required class="hidden">
                    <div id="uploadText">
                        <svg class="mx-auto h-12 w-12 text-gray-400 mb-4" stroke="currentColor" fill="none" viewBox="0 0 48 48">
                            <path d="M28 8H12a4 4 0 00-4 4v20m32-12v8m0 0v8a4 4 0 01-4 4H12a4 4 0 01-4-4v-4m32-4l-3.172-3.172a4 4 0 00-5.656 0L28 28M8 32l9.172-9.172a4 4 0 015.656 0L28 28m0 0l4 4m4-24h8m-4-4v8m-12 4h.02" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
                        </svg>
                        <p class="text-gray-600">Click to select a file or drag and drop</p>
                        <p class="text-sm text-gray-500">Maximum file size: 32MB</p>
                    </div>
                </div>
                
                <div id="fileInfo" class="hidden">
                    <p class="text-sm text-gray-600">Selected file: <span id="fileName"></span></p>
                    <p class="text-sm text-gray-600">Size: <span id="fileSize"></span></p>
                </div>
                
                <button type="submit" id="uploadBtn" 
                        class="w-full bg-blue-600 text-white py-2 px-4 rounded-md hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 disabled:bg-gray-400">
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
                <a href="/dashboard" class="text-blue-600 hover:text-blue-800">← Back to Dashboard</a>
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
                fileName.textContent = file.name;
                fileSize.textContent = formatFileSize(file.size);
                fileInfo.classList.remove('hidden');
                uploadBtn.disabled = false;
            }
        }

        function formatFileSize(bytes) {
            if (bytes === 0) return '0 Bytes';
            const k = 1024;
            const sizes = ['Bytes', 'KB', 'MB', 'GB'];
            const i = Math.floor(Math.log(bytes) / Math.log(k));
            return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
        }

        function resetUploadForm() {
            fileInput.value = '';
            fileInfo.classList.add('hidden');
            uploadBtn.disabled = false;
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

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

// createGraphClient creates Graph client with app credentials
func createGraphClient() (*msgraph.GraphServiceClient, error) {
	cred, err := azidentity.NewClientSecretCredential(
		os.Getenv("AZURE_TENANT_ID"),
		os.Getenv("AZURE_CLIENT_ID"),
		os.Getenv("AZURE_CLIENT_SECRET"),
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create credentials: %v", err)
	}

	graphClient, err := msgraph.NewGraphServiceClientWithCredentials(
		cred,
		[]string{"https://graph.microsoft.com/.default"},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create Graph client: %v", err)
	}

	return graphClient, nil
}
