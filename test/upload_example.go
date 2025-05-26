package test

import (
	"context"
	"fmt"
	"log"
	"os"

	"FinalProjectManagementApp/files"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/joho/godotenv"
	msgraph "github.com/microsoftgraph/msgraph-sdk-go"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found: %v", err)
	}

	// Validate required environment variables
	requiredEnvVars := []string{
		"AZURE_TENANT_ID",
		"AZURE_CLIENT_ID",
		"AZURE_CLIENT_SECRET",
	}

	for _, envVar := range requiredEnvVars {
		if os.Getenv(envVar) == "" {
			log.Fatalf("Required environment variable %s is not set", envVar)
		}
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

	fmt.Printf("Testing SharePoint upload with site: %s\n", siteURL)
	fmt.Println("==========================================")

	fmt.Println("Step 1: Getting SharePoint site ID...")
	siteID, err := files.GetSiteID(ctx, graphClient, siteURL)
	if err != nil {
		log.Fatal("Failed to get site ID:", err)
	}
	fmt.Printf("✓ Site ID: %s\n\n", siteID)

	fmt.Println("Step 2: Getting document library...")
	driveID, err := files.GetDocumentLibraryDriveID(ctx, graphClient, siteID)
	if err != nil {
		log.Fatal("Failed to get drive ID:", err)
	}
	fmt.Printf("✓ Drive ID: %s\n\n", driveID)

	// Create SharePoint service
	spService := files.NewSharePointService(graphClient, siteID, driveID)

	fmt.Println("Step 3: Creating test folder structure...")
	err = spService.CreateFolderPath(ctx, "test-uploads/go-sdk-test")
	if err != nil {
		fmt.Printf("Warning: Could not create folders: %v\n", err)
	} else {
		fmt.Println("✓ Folder structure created")
	}

	// Create a small test file
	fmt.Println("\nStep 4: Creating and uploading small test file...")
	smallTestFile := "small_test.txt"
	smallContent := fmt.Sprintf("Hello SharePoint from Go SDK!\nTimestamp: %s\nThis is a small file test.",
		fmt.Sprintf("%d", os.Getpid()))

	err = os.WriteFile(smallTestFile, []byte(smallContent), 0644)
	if err != nil {
		log.Fatal("Failed to create small test file:", err)
	}
	defer os.Remove(smallTestFile)

	// Upload the small test file
	err = spService.UploadFile(ctx, smallTestFile, "test-uploads/go-sdk-test")
	if err != nil {
		log.Fatal("Failed to upload small file:", err)
	}
	fmt.Println("✓ Small file uploaded successfully!")

	// Create a larger test file (> 4MB) to test resumable upload
	fmt.Println("\nStep 5: Creating and uploading large test file...")
	largeTestFile := "large_test.txt"

	// Create a file larger than 4MB
	largeContent := make([]byte, 5*1024*1024) // 5MB
	for i := range largeContent {
		largeContent[i] = byte('A' + (i % 26)) // Fill with A-Z pattern
	}

	err = os.WriteFile(largeTestFile, largeContent, 0644)
	if err != nil {
		log.Fatal("Failed to create large test file:", err)
	}
	defer os.Remove(largeTestFile)

	// Upload the large test file
	err = spService.UploadFile(ctx, largeTestFile, "test-uploads/go-sdk-test")
	if err != nil {
		log.Fatal("Failed to upload large file:", err)
	}
	fmt.Println("✓ Large file uploaded successfully!")

	fmt.Println("\n==========================================")
	fmt.Println("🎉 All SharePoint upload tests completed successfully!")
	fmt.Println("==========================================")

	fmt.Println("\nFiles uploaded to:")
	fmt.Printf("- %s/test-uploads/go-sdk-test/small_test.txt\n", siteURL)
	fmt.Printf("- %s/test-uploads/go-sdk-test/large_test.txt\n", siteURL)
}
