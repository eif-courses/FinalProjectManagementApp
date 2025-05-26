package test

import (
	"context"
	"fmt"
	"github.com/joho/godotenv"
	"log"
	"os"

	"FinalProjectManagementApp/files"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	msgraph "github.com/microsoftgraph/msgraph-sdk-go"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found: %v", err)
	}

	// Create Graph client
	cred, err := azidentity.NewClientSecretCredential(
		os.Getenv("AZURE_TENANT_ID"),
		os.Getenv("AZURE_CLIENT_ID"),
		os.Getenv("AZURE_CLIENT_SECRET"),
		nil,
	)
	if err != nil {
		log.Fatal("Failed to create credentials:", err)
	}

	graphClient, err := msgraph.NewGraphServiceClientWithCredentials(
		cred,
		[]string{"https://graph.microsoft.com/.default"},
	)
	if err != nil {
		log.Fatal("Failed to create Graph client:", err)
	}

	ctx := context.Background()

	// Test SharePoint connection
	siteURL := os.Getenv("SHAREPOINT_SITE_URL")
	if siteURL == "" {
		siteURL = "https://vikolt.sharepoint.com/sites/thesis_management-O365G"
	}

	fmt.Println("Getting SharePoint site ID...")
	siteID, err := files.GetSiteID(ctx, graphClient, siteURL)
	if err != nil {
		log.Fatal("Failed to get site ID:", err)
	}

	fmt.Println("Getting document library...")
	driveID, err := files.GetDocumentLibraryDriveID(ctx, graphClient, siteID)
	if err != nil {
		log.Fatal("Failed to get drive ID:", err)
	}

	// Create SharePoint service
	spService := files.NewSharePointService(graphClient, siteID, driveID)

	// Create a test file
	testFile := "test.txt"
	err = os.WriteFile(testFile, []byte("Hello SharePoint from Go!"), 0644)
	if err != nil {
		log.Fatal("Failed to create test file:", err)
	}
	defer os.Remove(testFile)

	// Upload the test file
	fmt.Println("Uploading test file...")
	err = spService.UploadFile(ctx, testFile, "test-uploads")
	if err != nil {
		log.Fatal("Failed to upload file:", err)
	}

	fmt.Println("Upload test completed successfully!")
}
