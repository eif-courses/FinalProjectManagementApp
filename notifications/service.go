// notifications/service.go
package notifications

import (
	"context"
	"fmt"
	"log"
	"os"
	"time" // Add this import

	msgraphsdk "github.com/microsoftgraph/msgraph-sdk-go"
	"github.com/microsoftgraph/msgraph-sdk-go/models"
	"github.com/microsoftgraph/msgraph-sdk-go/users"
)

type NotificationService struct {
	graphClient *msgraphsdk.GraphServiceClient
	systemEmail string // Your system notification email
}

func NewNotificationService(graphClient *msgraphsdk.GraphServiceClient) *NotificationService {
	systemEmail := os.Getenv("SYSTEM_NOTIFICATION_EMAIL") // e.g., "thesis-system@viko.lt"
	if systemEmail == "" {
		systemEmail = "thesis-notifications@baigiamieji.onmicrosoft.com" // fallback
	}

	return &NotificationService{
		graphClient: graphClient,
		systemEmail: systemEmail,
	}
}

// SendTestNotification sends a test email
func (n *NotificationService) SendTestNotification(ctx context.Context, toEmail string) error {
	if n.graphClient == nil {
		return fmt.Errorf("graph client is not initialized")
	}

	if n.systemEmail == "" {
		return fmt.Errorf("system email is not configured")
	}

	log.Printf("DEBUG: Attempting to send test notification from %s to %s", n.systemEmail, toEmail)

	subject := "Test Notification from Thesis Management System"

	body := fmt.Sprintf(`
<p>This is a test notification from the Thesis Management System.</p>
<p>If you received this email, the notification system is working correctly.</p>
<hr>
<p><strong>Debug Information:</strong></p>
<ul>
<li>Sent from: %s</li>
<li>Sent to: %s</li>
<li>Sent at: %s</li>
</ul>
<hr>
<p><em>Gerb. naudotojau, tai yra testas iš baigiamųjų darbų valdymo sistemos.</em></p>
`, n.systemEmail, toEmail, time.Now().Format("2006-01-02 15:04:05"))

	return n.sendNotification(ctx, toEmail, subject, body)
}

// IsEnabled returns whether the notification service is enabled
func (n *NotificationService) IsEnabled() bool {
	return n.graphClient != nil && n.systemEmail != ""
}

// TestConnection tests if the notification service can connect to Microsoft Graph
func (n *NotificationService) TestConnection(ctx context.Context) error {
	if !n.IsEnabled() {
		return fmt.Errorf("notification service is not properly configured")
	}

	// Try to get the system user to verify access
	_, err := n.graphClient.Users().ByUserId(n.systemEmail).Get(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to access system user %s: %w", n.systemEmail, err)
	}

	return nil
}

// GetSystemEmail returns the system email address being used

// Send thesis-related notifications
func (n *NotificationService) SendThesisDeadlineReminder(ctx context.Context, studentEmail, studentName, deadlineDate string) error {
	subject := "Thesis Deadline Reminder - Primenant apie baigiamojo darbo terminą"

	body := fmt.Sprintf(`
Dear %s / Gerb. %s,

This is a reminder that your thesis submission deadline is approaching: %s

Please ensure you submit all required documents before the deadline.

---

Tai priminimas, kad artėja jūsų baigiamojo darbo pateikimo terminas: %s

Prašome užtikrinti, kad visi reikalingi dokumentai būtų pateikti iki termino.

Best regards / Pagarbiai,
Thesis Management System
`, studentName, studentName, deadlineDate, deadlineDate)

	return n.sendNotification(ctx, studentEmail, subject, body)
}

func (n *NotificationService) SendTopicApprovalNotification(ctx context.Context, studentEmail, studentName, topicTitle string, approved bool) error {
	var subject, body string

	if approved {
		subject = "Thesis Topic Approved - Baigiamojo darbo tema patvirtinta"
		body = fmt.Sprintf(`
Dear %s / Gerb. %s,

Your thesis topic has been approved: "%s"

You can now proceed with your thesis work.

---

Jūsų baigiamojo darbo tema buvo patvirtinta: "%s"

Dabar galite tęsti darbą su baigiamuoju darbu.

Best regards / Pagarbiai,
Thesis Management System
`, studentName, studentName, topicTitle, topicTitle)
	} else {
		subject = "Thesis Topic Requires Revision - Baigiamojo darbo tema reikalauja pataisymų"
		body = fmt.Sprintf(`
Dear %s / Gerb. %s,

Your thesis topic "%s" requires revision before approval.

Please check your account for detailed feedback and resubmit.

---

Jūsų baigiamojo darbo tema "%s" reikalauja pataisymų prieš patvirtinimą.

Prašome patikrinti savo paskyrą dėl išsamaus atsiliepimo ir pateikti iš naujo.

Best regards / Pagarbiai,
Thesis Management System
`, studentName, studentName, topicTitle, topicTitle)
	}

	return n.sendNotification(ctx, studentEmail, subject, body)
}

func (n *NotificationService) SendReviewerAssignmentNotification(ctx context.Context, reviewerEmail, reviewerName, studentName, topicTitle string) error {
	subject := "New Thesis Review Assignment - Naujas baigiamojo darbo vertinimo paskyrimas"

	body := fmt.Sprintf(`
Dear %s / Gerb. %s,

You have been assigned as a reviewer for the following thesis:

Student: %s
Topic: %s

Please log into the system to access the thesis documents and provide your review.

---

Jums buvo paskirtas šio baigiamojo darbo vertinimas:

Studentas: %s
Tema: %s

Prašome prisijungti prie sistemos, kad galėtumėte peržiūrėti baigiamojo darbo dokumentus ir pateikti vertinimą.

Best regards / Pagarbiai,
Thesis Management System
`, reviewerName, reviewerName, studentName, topicTitle, studentName, topicTitle)

	return n.sendNotification(ctx, reviewerEmail, subject, body)
}

func (n *NotificationService) SendDefenseScheduleNotification(ctx context.Context, studentEmail, studentName, defenseDate, defenseTime, location string) error {
	subject := "Thesis Defense Scheduled - Baigiamojo darbo gynimas suplanuotas"

	body := fmt.Sprintf(`
Dear %s / Gerb. %s,

Your thesis defense has been scheduled:

Date: %s
Time: %s
Location: %s

Please confirm your attendance and prepare for the defense.

---

Jūsų baigiamojo darbo gynimas buvo suplanuotas:

Data: %s
Laikas: %s
Vieta: %s

Prašome patvirtinti dalyvavimą ir pasiruošti gynimui.

Best regards / Pagarbiai,
Thesis Management System
`, studentName, studentName, defenseDate, defenseTime, location, defenseDate, defenseTime, location)

	return n.sendNotification(ctx, studentEmail, subject, body)
}

// Add this method to your notifications/service.go
func (n *NotificationService) SendTestNotificationWithDebug(ctx context.Context, toEmail string) error {
	if n.graphClient == nil {
		return fmt.Errorf("graph client is nil")
	}

	if n.systemEmail == "" {
		return fmt.Errorf("system email is empty")
	}

	fmt.Printf("DEBUG: Sending from: %s to: %s\n", n.systemEmail, toEmail)

	// Test if we can access the system user first
	fmt.Printf("DEBUG: Testing access to system user: %s\n", n.systemEmail)
	user, err := n.graphClient.Users().ByUserId(n.systemEmail).Get(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to access system user %s: %w", n.systemEmail, err)
	}

	if user.GetDisplayName() != nil {
		fmt.Printf("DEBUG: System user found: %s\n", *user.GetDisplayName())
	}

	// Now try sending the email
	subject := "Test Notification from Thesis Management System"
	body := fmt.Sprintf(`
<p>This is a test notification sent at %s</p>
<p>System email: %s</p>
<p>Target email: %s</p>
`, time.Now().Format("2006-01-02 15:04:05"), n.systemEmail, toEmail)

	return n.sendNotification(ctx, toEmail, subject, body)
}

// Core notification sending method
// sendNotification sends an email notification with detailed error handling
func (n *NotificationService) sendNotification(ctx context.Context, toEmail, subject, body string) error {
	log.Printf("DEBUG: Creating email message")

	message := models.NewMessage()

	// Set recipient
	recipient := models.NewRecipient()
	emailAddress := models.NewEmailAddress()
	emailAddress.SetAddress(&toEmail)
	recipient.SetEmailAddress(emailAddress)
	message.SetToRecipients([]models.Recipientable{recipient})

	// Set subject and body
	message.SetSubject(&subject)
	messageBody := models.NewItemBody()
	contentType := models.HTML_BODYTYPE
	messageBody.SetContentType(&contentType)
	messageBody.SetContent(&body)
	message.SetBody(messageBody)

	// Send from system email
	sendMailRequest := users.NewItemSendMailPostRequestBody()
	sendMailRequest.SetMessage(message)

	log.Printf("DEBUG: Attempting to send email via Graph API using system email: %s", n.systemEmail)

	err := n.graphClient.Users().ByUserId(n.systemEmail).SendMail().Post(ctx, sendMailRequest, nil)
	if err != nil {
		log.Printf("ERROR: Graph API call failed: %v", err)
		return fmt.Errorf("graph API send mail failed: %w", err)
	}

	log.Printf("SUCCESS: Email sent successfully via Graph API")
	return nil
}
func (n *NotificationService) GetSystemEmail() string {
	return n.systemEmail
}
