package files // files/sharepoint_test.go
import (
	"context"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/joho/godotenv"
	msgraph "github.com/microsoftgraph/msgraph-sdk-go"
	"os"
	"testing"
)

func TestSharePointConnection(t *testing.T) {
	// Load environment variables
	if err := godotenv.Load("../.env"); err != nil {
		t.Skip("Skipping test: .env file not found")
	}

	// Check if required environment variables are set
	if os.Getenv("AZURE_CLIENT_ID") == "" {
		t.Skip("Skipping test: AZURE_CLIENT_ID not set")
	}

	// Create Graph client
	cred, err := azidentity.NewClientSecretCredential(
		os.Getenv("AZURE_TENANT_ID"),
		os.Getenv("AZURE_CLIENT_ID"),
		os.Getenv("AZURE_CLIENT_SECRET"),
		nil,
	)
	if err != nil {
		t.Fatalf("Failed to create credentials: %v", err)
	}

	graphClient, err := msgraph.NewGraphServiceClientWithCredentials(
		cred,
		[]string{"https://graph.microsoft.com/.default"},
	)
	if err != nil {
		t.Fatalf("Failed to create Graph client: %v", err)
	}

	ctx := context.Background()
	siteURL := os.Getenv("SHAREPOINT_SITE_URL")
	if siteURL == "" {
		siteURL = "https://vikolt.sharepoint.com/sites/thesis_management-O365G"
	}

	// Test getting site ID
	siteID, err := GetSiteID(ctx, graphClient, siteURL)
	if err != nil {
		t.Fatalf("Failed to get site ID: %v", err)
	}

	if siteID == "" {
		t.Fatal("Site ID is empty")
	}

	t.Logf("Site ID: %s", siteID)

	// Test getting drive ID
	driveID, err := GetDocumentLibraryDriveID(ctx, graphClient, siteID)
	if err != nil {
		t.Fatalf("Failed to get drive ID: %v", err)
	}

	if driveID == "" {
		t.Fatal("Drive ID is empty")
	}

	t.Logf("Drive ID: %s", driveID)
}

func TestFileUpload(t *testing.T) {
	// Load environment variables
	if err := godotenv.Load("../.env"); err != nil {
		t.Skip("Skipping test: .env file not found")
	}

	// Check if required environment variables are set
	if os.Getenv("AZURE_CLIENT_ID") == "" {
		t.Skip("Skipping test: Azure credentials not set")
	}

	// Create Graph client
	cred, err := azidentity.NewClientSecretCredential(
		os.Getenv("AZURE_TENANT_ID"),
		os.Getenv("AZURE_CLIENT_ID"),
		os.Getenv("AZURE_CLIENT_SECRET"),
		nil,
	)
	if err != nil {
		t.Fatalf("Failed to create credentials: %v", err)
	}

	graphClient, err := msgraph.NewGraphServiceClientWithCredentials(
		cred,
		[]string{"https://graph.microsoft.com/.default"},
	)
	if err != nil {
		t.Fatalf("Failed to create Graph client: %v", err)
	}

	ctx := context.Background()
	siteURL := os.Getenv("SHAREPOINT_SITE_URL")
	if siteURL == "" {
		siteURL = "https://vikolt.sharepoint.com/sites/thesis_management-O365G"
	}

	// Get site and drive IDs
	siteID, err := GetSiteID(ctx, graphClient, siteURL)
	if err != nil {
		t.Fatalf("Failed to get site ID: %v", err)
	}

	driveID, err := GetDocumentLibraryDriveID(ctx, graphClient, siteID)
	if err != nil {
		t.Fatalf("Failed to get drive ID: %v", err)
	}

	// Create SharePoint service
	spService := NewSharePointService(graphClient, siteID, driveID)

	// Create a test file
	testFile := "test_upload.txt"
	testContent := "This is a test file for SharePoint upload"

	err = os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Remove(testFile)

	// Upload the test file
	err = spService.UploadFile(ctx, testFile, "test-uploads")
	if err != nil {
		t.Fatalf("Failed to upload file: %v", err)
	}

	t.Log("File uploaded successfully!")
}
