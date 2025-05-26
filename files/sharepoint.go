// files/sharepoint.go - SharePoint file upload service (FIXED for your SDK version)
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
	if fileInfo.Size() < 4*1024*1024 {
		return s.simpleUpload(ctx, file, targetPath)
	} else {
		return s.resumableUpload(ctx, file, targetPath, fileInfo.Size())
	}
}

// simpleUpload for small files - FIXED
func (s *SharePointService) simpleUpload(ctx context.Context, file *os.File, targetPath string) error {
	fileContent, err := io.ReadAll(file)
	if err != nil {
		return fmt.Errorf("failed to read file: %v", err)
	}

	// Upload using byte slice instead of Reader
	_, err = s.graphClient.Drives().ByDriveId(s.driveID).
		Items().ByDriveItemId("root:/"+targetPath+":").
		Content().Put(ctx, fileContent, nil)

	if err != nil {
		return fmt.Errorf("failed to upload file to SharePoint: %v", err)
	}

	fmt.Printf("File uploaded successfully to SharePoint: %s\n", targetPath)
	return nil
}

// resumableUpload for large files - FIXED
func (s *SharePointService) resumableUpload(ctx context.Context, file *os.File, targetPath string, fileSize int64) error {
	// Create drive item uploadable properties
	driveItemProps := models.NewDriveItemUploadableProperties()
	fileName := filepath.Base(targetPath)
	driveItemProps.SetName(&fileName)

	// Use the correct type for the request body
	uploadSessionRequest := drives.NewItemItemsItemCreateUploadSessionPostRequestBody()
	uploadSessionRequest.SetItem(driveItemProps)

	uploadSession, err := s.graphClient.Drives().ByDriveId(s.driveID).
		Items().ByDriveItemId("root:/"+targetPath+":").
		CreateUploadSession().
		Post(ctx, uploadSessionRequest, nil)

	if err != nil {
		return fmt.Errorf("failed to create SharePoint upload session: %v", err)
	}

	// Upload in chunks
	chunkSize := int64(320 * 1024) // 320KB chunks
	buffer := make([]byte, chunkSize)

	for offset := int64(0); offset < fileSize; {
		n, err := file.Read(buffer)
		if err != nil && err != io.EOF {
			return fmt.Errorf("failed to read file chunk: %v", err)
		}

		if n == 0 {
			break
		}

		// Upload chunk
		err = s.uploadChunk(uploadSession.GetUploadUrl(), buffer[:n], offset, fileSize)
		if err != nil {
			return fmt.Errorf("failed to upload chunk: %v", err)
		}

		offset += int64(n)
		fmt.Printf("Uploaded %d/%d bytes to SharePoint\n", offset, fileSize)
	}

	return nil
}

func (s *SharePointService) uploadChunk(uploadUrl *string, chunk []byte, offset, totalSize int64) error {
	req, err := http.NewRequest("PUT", *uploadUrl, bytes.NewReader(chunk))
	if err != nil {
		return err
	}

	contentRange := fmt.Sprintf("bytes %d-%d/%d", offset, offset+int64(len(chunk))-1, totalSize)
	req.Header.Set("Content-Range", contentRange)
	req.Header.Set("Content-Length", fmt.Sprintf("%d", len(chunk)))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("upload failed with status: %d", resp.StatusCode)
	}

	return nil
}

// CreateFolder creates a folder in SharePoint library - FIXED
func (s *SharePointService) CreateFolder(ctx context.Context, folderPath string) error {
	// Create DriveItem using the New constructor
	driveItem := models.NewDriveItem()
	driveItem.SetName(&folderPath)

	// Create folder object
	folder := models.NewFolder()
	driveItem.SetFolder(folder)

	_, err := s.graphClient.Drives().ByDriveId(s.driveID).
		Items().ByDriveItemId("root").
		Children().Post(ctx, driveItem, nil)

	if err != nil {
		return fmt.Errorf("failed to create folder: %v", err)
	}

	return nil
}

// CreateFolderPath creates nested folders
func (s *SharePointService) CreateFolderPath(ctx context.Context, folderPath string) error {
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
			// Folder might already exist, continue
			fmt.Printf("Note: Could not create folder %s: %v\n", currentPath, err)
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

// GetSiteID gets SharePoint site ID by site URL
func GetSiteID(ctx context.Context, graphClient *msgraphsdk.GraphServiceClient, siteURL string) (string, error) {
	// Extract hostname and site path from URL
	// Example: https://vikolt.sharepoint.com/sites/your-site
	parts := strings.Split(strings.TrimPrefix(siteURL, "https://"), "/")
	if len(parts) < 3 {
		return "", fmt.Errorf("invalid SharePoint site URL")
	}

	hostname := parts[0]
	sitePath := strings.Join(parts[1:], "/")

	site, err := graphClient.Sites().BySiteId(fmt.Sprintf("%s:/%s", hostname, sitePath)).Get(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("failed to get site: %v", err)
	}

	return *site.GetId(), nil
}

// GetDocumentLibraryDriveID gets the default document library drive ID
func GetDocumentLibraryDriveID(ctx context.Context, graphClient *msgraphsdk.GraphServiceClient, siteID string) (string, error) {
	drive, err := graphClient.Sites().BySiteId(siteID).Drive().Get(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("failed to get default drive: %v", err)
	}

	return *drive.GetId(), nil
}

// GetDriveIDByName gets drive ID by library name
func GetDriveIDByName(ctx context.Context, graphClient *msgraphsdk.GraphServiceClient, siteID, libraryName string) (string, error) {
	drives, err := graphClient.Sites().BySiteId(siteID).Drives().Get(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("failed to get drives: %v", err)
	}

	for _, drive := range drives.GetValue() {
		if drive.GetName() != nil && *drive.GetName() == libraryName {
			return *drive.GetId(), nil
		}
	}

	return "", fmt.Errorf("document library '%s' not found", libraryName)
}
