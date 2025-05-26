// files/sharepoint.go - Fixed SharePoint file upload service
package files

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	msgraphsdk "github.com/microsoftgraph/msgraph-sdk-go"
	"github.com/microsoftgraph/msgraph-sdk-go/drives"
	"github.com/microsoftgraph/msgraph-sdk-go/models"
)

type SharePointService struct {
	graphClient *msgraphsdk.GraphServiceClient
	siteID      string
	driveID     string
}

// NewSharePointService creates a new SharePoint service
func NewSharePointService(graphClient *msgraphsdk.GraphServiceClient, siteID, driveID string) *SharePointService {
	return &SharePointService{
		graphClient: graphClient,
		siteID:      siteID,
		driveID:     driveID,
	}
}

// UploadFile uploads a file to SharePoint library
func (s *SharePointService) UploadFile(ctx context.Context, filePath, targetFolder string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to get file info: %v", err)
	}

	fileName := filepath.Base(filePath)

	// Create target path in SharePoint
	var targetPath string
	if targetFolder != "" {
		targetPath = fmt.Sprintf("%s/%s", strings.Trim(targetFolder, "/"), fileName)
	} else {
		targetPath = fileName
	}

	// For small files (< 4MB), use simple upload
	const maxSimpleUploadSize = 4 * 1024 * 1024 // 4MB
	if fileInfo.Size() < maxSimpleUploadSize {
		return s.simpleUpload(ctx, file, targetPath)
	} else {
		return s.resumableUpload(ctx, file, targetPath, fileInfo.Size())
	}
}

// simpleUpload for small files - FIXED to use correct SDK methods
func (s *SharePointService) simpleUpload(ctx context.Context, file *os.File, targetPath string) error {
	// Read file content
	fileContent, err := io.ReadAll(file)
	if err != nil {
		return fmt.Errorf("failed to read file: %v", err)
	}

	// Upload using the correct method signature - no custom headers needed
	driveItem, err := s.graphClient.Drives().ByDriveId(s.driveID).
		Items().ByDriveItemId("root:/"+targetPath+":").
		Content().Put(ctx, fileContent, nil)

	if err != nil {
		return fmt.Errorf("failed to upload file to SharePoint: %v", err)
	}

	if driveItem != nil && driveItem.GetName() != nil {
		fmt.Printf("File uploaded successfully to SharePoint: %s (ID: %s)\n",
			*driveItem.GetName(), *driveItem.GetId())
	}

	return nil
}

// resumableUpload for large files - FIXED with proper session handling
func (s *SharePointService) resumableUpload(ctx context.Context, file *os.File, targetPath string, fileSize int64) error {
	// Create drive item uploadable properties
	driveItemProps := models.NewDriveItemUploadableProperties()
	fileName := filepath.Base(targetPath)
	driveItemProps.SetName(&fileName)

	// Set conflict behavior
	additionalData := map[string]interface{}{
		"@microsoft.graph.conflictBehavior": "replace",
	}
	driveItemProps.SetAdditionalData(additionalData)

	// Create upload session request body
	uploadSessionRequest := drives.NewItemItemsItemCreateUploadSessionPostRequestBody()
	uploadSessionRequest.SetItem(driveItemProps)

	// Create upload session
	uploadSession, err := s.graphClient.Drives().ByDriveId(s.driveID).
		Items().ByDriveItemId("root:/"+targetPath+":").
		CreateUploadSession().
		Post(ctx, uploadSessionRequest, nil)

	if err != nil {
		return fmt.Errorf("failed to create SharePoint upload session: %v", err)
	}

	if uploadSession.GetUploadUrl() == nil {
		return fmt.Errorf("upload session did not return an upload URL")
	}

	// Upload in chunks
	const chunkSize = 320 * 1024 // 320KB chunks (required multiple of 320KB)
	buffer := make([]byte, chunkSize)
	offset := int64(0)

	for offset < fileSize {
		// Read chunk
		n, err := file.Read(buffer)
		if err != nil && err != io.EOF {
			return fmt.Errorf("failed to read file chunk: %v", err)
		}

		if n == 0 {
			break
		}

		// Upload chunk
		err = s.uploadChunk(*uploadSession.GetUploadUrl(), buffer[:n], offset, fileSize)
		if err != nil {
			return fmt.Errorf("failed to upload chunk at offset %d: %v", offset, err)
		}

		offset += int64(n)
		fmt.Printf("Uploaded %d/%d bytes to SharePoint (%.1f%%)\n",
			offset, fileSize, float64(offset)/float64(fileSize)*100)
	}

	fmt.Printf("Large file upload completed successfully: %s\n", targetPath)
	return nil
}

// uploadChunk uploads a single chunk to the upload session URL
func (s *SharePointService) uploadChunk(uploadUrl string, chunk []byte, offset, totalSize int64) error {
	req, err := http.NewRequest("PUT", uploadUrl, bytes.NewReader(chunk))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	// Set required headers
	contentRange := fmt.Sprintf("bytes %d-%d/%d", offset, offset+int64(len(chunk))-1, totalSize)
	req.Header.Set("Content-Range", contentRange)
	req.Header.Set("Content-Length", fmt.Sprintf("%d", len(chunk)))
	req.Header.Set("Content-Type", "application/octet-stream")

	// Execute request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute upload request: %v", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("upload failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}

// CreateFolder creates a folder in SharePoint library - FIXED
func (s *SharePointService) CreateFolder(ctx context.Context, folderPath string) error {
	// Create DriveItem
	driveItem := models.NewDriveItem()
	driveItem.SetName(&folderPath)

	// Create folder facet
	folder := models.NewFolder()
	driveItem.SetFolder(folder)

	// Create the folder
	_, err := s.graphClient.Drives().ByDriveId(s.driveID).
		Items().ByDriveItemId("root").
		Children().Post(ctx, driveItem, nil)

	if err != nil {
		return fmt.Errorf("failed to create folder '%s': %v", folderPath, err)
	}

	fmt.Printf("Folder created successfully: %s\n", folderPath)
	return nil
}

// CreateFolderPath creates nested folders
func (s *SharePointService) CreateFolderPath(ctx context.Context, folderPath string) error {
	if folderPath == "" {
		return nil
	}

	// Split the path and create folders one by one
	parts := strings.Split(strings.Trim(folderPath, "/"), "/")
	currentPath := ""

	for _, part := range parts {
		if part == "" {
			continue
		}

		if currentPath == "" {
			currentPath = part
		} else {
			currentPath = currentPath + "/" + part
		}

		// Try to create each folder in the path
		err := s.createSingleFolder(ctx, currentPath, part)
		if err != nil {
			// Check if error is because folder already exists
			if strings.Contains(err.Error(), "already exists") ||
				strings.Contains(err.Error(), "nameAlreadyExists") {
				fmt.Printf("Folder already exists: %s\n", currentPath)
				continue
			}
			return fmt.Errorf("failed to create folder path '%s': %v", currentPath, err)
		} else {
			fmt.Printf("Created folder: %s\n", currentPath)
		}
	}

	return nil
}

// createSingleFolder creates a single folder
func (s *SharePointService) createSingleFolder(ctx context.Context, fullPath, folderName string) error {
	driveItem := models.NewDriveItem()
	driveItem.SetName(&folderName)

	folder := models.NewFolder()
	driveItem.SetFolder(folder)

	// Determine parent path
	parentPath := strings.TrimSuffix(fullPath, "/"+folderName)
	var parentID string

	if parentPath == "" {
		parentID = "root"
	} else {
		parentID = "root:/" + parentPath + ":"
	}

	_, err := s.graphClient.Drives().ByDriveId(s.driveID).
		Items().ByDriveItemId(parentID).
		Children().Post(ctx, driveItem, nil)

	return err
}

// GetSiteID gets SharePoint site ID by site URL - IMPROVED error handling
func GetSiteID(ctx context.Context, graphClient *msgraphsdk.GraphServiceClient, siteURL string) (string, error) {
	// Extract hostname and site path from URL
	// Example: https://vikolt.sharepoint.com/sites/your-site
	siteURL = strings.TrimPrefix(siteURL, "https://")
	siteURL = strings.TrimPrefix(siteURL, "http://")

	parts := strings.Split(siteURL, "/")
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid SharePoint site URL format. Expected: https://tenant.sharepoint.com/sites/sitename")
	}

	hostname := parts[0]
	sitePath := strings.Join(parts[1:], "/")

	// Use the sites API to get site info
	siteIdentifier := fmt.Sprintf("%s:/%s", hostname, sitePath)

	site, err := graphClient.Sites().BySiteId(siteIdentifier).Get(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("failed to get site '%s': %v", siteIdentifier, err)
	}

	if site.GetId() == nil {
		return "", fmt.Errorf("site ID not found for site '%s'", siteIdentifier)
	}

	fmt.Printf("Found SharePoint site: %s (ID: %s)\n",
		getStringValue(site.GetDisplayName()), *site.GetId())

	return *site.GetId(), nil
}

// GetDocumentLibraryDriveID gets the default document library drive ID
func GetDocumentLibraryDriveID(ctx context.Context, graphClient *msgraphsdk.GraphServiceClient, siteID string) (string, error) {
	drive, err := graphClient.Sites().BySiteId(siteID).Drive().Get(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("failed to get default drive for site '%s': %v", siteID, err)
	}

	if drive.GetId() == nil {
		return "", fmt.Errorf("drive ID not found for site '%s'", siteID)
	}

	fmt.Printf("Found document library: %s (ID: %s)\n",
		getStringValue(drive.GetName()), *drive.GetId())

	return *drive.GetId(), nil
}

// GetDriveIDByName gets drive ID by library name
func GetDriveIDByName(ctx context.Context, graphClient *msgraphsdk.GraphServiceClient, siteID, libraryName string) (string, error) {
	drives, err := graphClient.Sites().BySiteId(siteID).Drives().Get(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("failed to get drives for site '%s': %v", siteID, err)
	}

	if drives.GetValue() == nil {
		return "", fmt.Errorf("no drives found for site '%s'", siteID)
	}

	for _, drive := range drives.GetValue() {
		if drive.GetName() != nil && *drive.GetName() == libraryName {
			if drive.GetId() == nil {
				continue
			}
			fmt.Printf("Found library '%s': %s (ID: %s)\n",
				libraryName, *drive.GetName(), *drive.GetId())
			return *drive.GetId(), nil
		}
	}

	return "", fmt.Errorf("document library '%s' not found in site '%s'", libraryName, siteID)
}

// Helper function to safely get string values
func getStringValue(s *string) string {
	if s == nil {
		return "Unknown"
	}
	return *s
}
