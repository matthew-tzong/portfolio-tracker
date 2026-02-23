package email

import (
	"context"
	"os"

	"github.com/resend/resend-go/v3"
)

// Sends an email to the given address with a CSV attachment.
func SendCSV(ctx context.Context, to, subject, filename string, csvContent []byte) error {
	secretKey := os.Getenv("RESEND_API_KEY")
	fromEmail := os.Getenv("RESEND_FROM")
	client := resend.NewClient(secretKey)
	// Creates a new email request.
	params := &resend.SendEmailRequest{
		From:    fromEmail,
		To:      []string{to},
		Subject: subject,
		Text:    "CSV export attached (data pruned by retention policy).",
		Attachments: []*resend.Attachment{
			{
				Content:     csvContent,
				Filename:    filename,
				ContentType: "text/csv",
			},
		},
	}
	// Sends the email.
	_, err := client.Emails.SendWithContext(ctx, params)
	return err
}
