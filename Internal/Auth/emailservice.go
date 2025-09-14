package Auth

import (
	"context"
	"crypto/tls"
	"fmt"
	"html/template"
	configs "app/configs"
	"net/smtp"

	"strings"
	"time"

	utils "app/pkg/utils"

	"github.com/badoux/checkmail"
)

// EmailService handles sending emails via SMTP SSL with connection pooling
type EmailService struct {
	SMTPHost   string
	SMTPPort   int
	Username   string
	Password   string
	Timeout    time.Duration
	MaxRetries int
	RetryDelay time.Duration
	templates  *emailTemplates
}

type emailTemplates struct {
	welcome      *template.Template
	verification *template.Template
}

// EmailTemplate data structures
type WelcomeEmailData struct {
	Username string
	AppName  string
	Year     int
}

type VerificationEmailData struct {
	Code    string
	AppName string
	Minutes int
}

// NewEmailService creates an EmailService with connection pooling
func NewEmailService(cfg *configs.Config) (*EmailService, error) {
	if cfg.Email.SMTPHost == "" || cfg.Email.Username == "" || cfg.Email.Password == "" {
		return nil, fmt.Errorf("email configuration is incomplete")
	}

	es := &EmailService{
		SMTPHost:   cfg.Email.SMTPHost,
		SMTPPort:   cfg.Email.SMTPPort,
		Username:   cfg.Email.Username,
		Password:   cfg.Email.Password,
		Timeout:    time.Duration(cfg.Email.Timeout) * time.Second,
		MaxRetries: 3,
		RetryDelay: 2 * time.Second,
	}

	// Set default timeout if not configured
	if es.Timeout == 0 {
		es.Timeout = 30 * time.Second
	}

	// Initialize email templates
	if err := es.initTemplates(); err != nil {
		return nil, fmt.Errorf("failed to initialize email templates: %v", err)
	}

	return es, nil
}

// initTemplates initializes HTML email templates
func (es *EmailService) initTemplates() error {
	// Welcome email template
	welcomeHTML := `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Welcome to {{.AppName}}</title>
</head>
<body style="font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif; background: #f5f5f5; margin: 0; padding: 20px;">
    <div style="max-width: 600px; margin: 0 auto; background: #ffffff; border-radius: 12px; box-shadow: 0 4px 12px rgba(0,0,0,0.1); overflow: hidden;">
        <div style="background: linear-gradient(135deg, #4CAF50, #45a049); padding: 40px; text-align: center;">
            <h1 style="color: #ffffff; margin: 0; font-size: 28px; font-weight: 600;">Welcome to {{.AppName}}!</h1>
        </div>
        <div style="padding: 40px;">
            <h2 style="color: #333; margin: 0 0 20px 0; font-size: 24px;">Hi {{.Username}},</h2>
            <p style="color: #555; line-height: 1.6; margin: 0 0 20px 0; font-size: 16px;">
                Welcome to {{.AppName}}! We're excited to have you join our streaming community.
            </p>
            <div style="margin: 30px 0; padding: 25px; background: #f8f9fa; border-radius: 8px; border-left: 4px solid #4CAF50;">
                <h3 style="color: #333; margin: 0 0 15px 0; font-size: 18px;">Getting Started:</h3>
                <ul style="color: #666; margin: 0; padding-left: 20px; line-height: 1.8;">
                    <li>Complete your profile setup</li>
                    <li>Explore the streaming community</li>
                    <li>Start your first live stream</li>
                    <li>Connect with other streamers</li>
                </ul>
            </div>
            <p style="color: #888; font-size: 14px; margin: 30px 0 0 0; text-align: center;">
                If you have any questions, feel free to contact our support team. We're here to help!<br>
                <strong>Team {{.AppName}}</strong><br>
                <small>© {{.Year}} {{.AppName}}. All rights reserved.</small>
            </p>
        </div>
    </div>
</body>
</html>`

	// Verification email template
	verificationHTML := `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Verification Code - {{.AppName}}</title>
</head>
<body style="font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif; background: #f5f5f5; margin: 0; padding: 20px;">
    <div style="max-width: 600px; margin: 0 auto; background: #ffffff; border-radius: 12px; box-shadow: 0 4px 12px rgba(0,0,0,0.1); overflow: hidden;">
        <div style="background: linear-gradient(135deg, #FF9800, #F57C00); padding: 40px; text-align: center;">
            <h1 style="color: #ffffff; margin: 0; font-size: 28px; font-weight: 600;">Security Verification</h1>
        </div>
        <div style="padding: 40px; text-align: center;">
            <h2 style="color: #333; margin: 0 0 20px 0;">Your verification code is:</h2>
            <div style="background: #f8f9fa; padding: 30px; border-radius: 12px; margin: 30px 0; border: 2px dashed #FF9800;">
                <span style="font-size: 36px; font-weight: bold; color: #FF9800; letter-spacing: 8px; font-family: 'Courier New', monospace;">{{.Code}}</span>
            </div>
            <p style="color: #666; font-size: 16px; margin: 20px 0;">
                This code will expire in <strong>{{.Minutes}} minutes</strong>.
            </p>
            <div style="background: #fff3cd; border: 1px solid #ffeaa7; border-radius: 8px; padding: 15px; margin: 20px 0;">
                <p style="color: #856404; margin: 0; font-size: 14px;">
                    ⚠️ If you didn't request this code, please ignore this email and secure your account.
                </p>
            </div>
            <p style="color: #888; font-size: 14px; margin: 30px 0 0 0;">
                <strong>Team {{.AppName}} Security</strong>
            </p>
        </div>
    </div>
</body>
</html>`

	var err error
	es.templates = &emailTemplates{}

	es.templates.welcome, err = template.New("welcome").Parse(welcomeHTML)
	if err != nil {
		return err
	}

	es.templates.verification, err = template.New("verification").Parse(verificationHTML)
	if err != nil {
		return err
	}

	return nil
}

// SendWelcomeEmail sends a welcome email using template
func (es *EmailService) SendWelcomeEmail(ctx context.Context, to, username string) error {
	if !isValidEmail(to) {
		return fmt.Errorf("invalid email address: %s", to)
	}

	data := WelcomeEmailData{
		Username: username,
		AppName:  "Kitch",
		Year:     time.Now().Year(),
	}

	var bodyBuffer strings.Builder
	if err := es.templates.welcome.Execute(&bodyBuffer, data); err != nil {
		return fmt.Errorf("failed to execute welcome email template: %v", err)
	}

	subject := "Welcome - Your Digital Journey Begins!"
	return es.sendMailWithRetry(ctx, to, subject, bodyBuffer.String(), true)
}

// Send2StepVerificationEmail sends a 2FA verification email
func (es *EmailService) Send2StepVerificationEmail(ctx context.Context, to, token string) error {
	if !isValidEmail(to) {
		return fmt.Errorf("invalid email address: %s", to)
	}

	if len(token) != 6 {
		return fmt.Errorf("verification token must be 6 digits")
	}

	data := VerificationEmailData{
		Code:    token,
		AppName: "Kitch",
		Minutes: 10,
	}

	var bodyBuffer strings.Builder
	if err := es.templates.verification.Execute(&bodyBuffer, data); err != nil {
		return fmt.Errorf("failed to execute verification email template: %v", err)
	}

	subject := "Your  Verification Code"
	return es.sendMailWithRetry(ctx, to, subject, bodyBuffer.String(), true)
}

// SendConfirmationEmail is an alias for SendWelcomeEmail for backward compatibility
func (es *EmailService) SendConfirmationEmail(ctx context.Context, to, username string) error {
	return es.SendWelcomeEmail(ctx, to, username)
}

// sendMailWithRetry implements retry logic for email sending
func (es *EmailService) sendMailWithRetry(ctx context.Context, to, subject, body string, isHTML bool) error {
	var lastErr error

	for attempt := 0; attempt <= es.MaxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(es.RetryDelay * time.Duration(attempt)):
				// Exponential backoff
			}
		}

		err := es.sendMail(ctx, to, subject, body, isHTML)
		if err == nil {
			return nil
		}

		lastErr = err
		if attempt < es.MaxRetries {
			// Log retry attempt with proper structured logging
			utils.Logger.Warnf("Email send attempt %d failed, retrying: %v", attempt+1, err)
		}
	}

	return fmt.Errorf("failed to send email after %d attempts: %v", es.MaxRetries+1, lastErr)
}

// sendMail sends email using SMTP SSL with improved error handling
func (es *EmailService) sendMail(ctx context.Context, to, subject, body string, isHTML bool) error {
	// Create context with timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, es.Timeout)
	defer cancel()

	// Validate inputs
	if err := es.validateEmailInputs(to, subject, body); err != nil {
		return err
	}

	// Build message
	msg, err := es.buildMessage(to, subject, body, isHTML)
	if err != nil {
		return err
	}

	// Send email with goroutine and proper error handling
	done := make(chan error, 1)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				done <- fmt.Errorf("panic in email sending: %v", r)
			}
		}()

		done <- es.sendSMTP(to, msg)
	}()

	// Wait for completion or timeout
	select {
	case <-timeoutCtx.Done():
		return fmt.Errorf("email sending timed out after %v: %v", es.Timeout, timeoutCtx.Err())
	case err := <-done:
		return err
	}
}

// validateEmailInputs validates email sending parameters
func (es *EmailService) validateEmailInputs(to, subject, body string) error {
	if to == "" {
		return fmt.Errorf("recipient email is required")
	}
	if subject == "" {
		return fmt.Errorf("subject is required")
	}
	if body == "" {
		return fmt.Errorf("email body is required")
	}
	if !isValidEmail(to) {
		return fmt.Errorf("invalid recipient email format: %s", to)
	}
	return nil
}

// buildMessage constructs the email message with proper headers
func (es *EmailService) buildMessage(to, subject, body string, isHTML bool) (string, error) {
	headers := make(map[string]string)
	headers["From"] = es.Username
	headers["To"] = to
	headers["Subject"] = subject
	headers["Date"] = time.Now().Format(time.RFC1123Z)
	headers["Message-ID"] = fmt.Sprintf("<%d.%s@%s>",
		time.Now().Unix(),
		strings.ReplaceAll(strings.Split(to, "@")[0], ".", "-"),
		es.SMTPHost)

	if isHTML {
		headers["MIME-Version"] = "1.0"
		headers["Content-Type"] = "text/html; charset=UTF-8"
		headers["Content-Transfer-Encoding"] = "quoted-printable"
	} else {
		headers["Content-Type"] = "text/plain; charset=UTF-8"
	}

	var msg strings.Builder
	for k, v := range headers {
		msg.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
	}
	msg.WriteString("\r\n" + body)

	return msg.String(), nil
}

// sendSMTP handles the actual SMTP communication
func (es *EmailService) sendSMTP(to, message string) error {
	addr := fmt.Sprintf("%s:%d", es.SMTPHost, es.SMTPPort)

	if es.SMTPPort == 465 {
		// Implicit TLS
		tlsConfig := &tls.Config{
			ServerName:         es.SMTPHost,
			InsecureSkipVerify: false,
			MinVersion:         tls.VersionTLS12,
		}
		tlsConn, err := tls.Dial("tcp", addr, tlsConfig)
		if err != nil {
			return fmt.Errorf("failed to establish TLS connection to %s: %v", addr, err)
		}
		defer tlsConn.Close()

		client, err := smtp.NewClient(tlsConn, es.SMTPHost)
		if err != nil {
			return fmt.Errorf("failed to create SMTP client: %v", err)
		}
		defer client.Quit()

		auth := smtp.PlainAuth("", es.Username, es.Password, es.SMTPHost)
		if err = client.Auth(auth); err != nil {
			return fmt.Errorf("SMTP authentication failed: %v", err)
		}
		if err = client.Mail(es.Username); err != nil {
			return fmt.Errorf("failed to set sender: %v", err)
		}
		if err = client.Rcpt(to); err != nil {
			return fmt.Errorf("failed to set recipient: %v", err)
		}
		writer, err := client.Data()
		if err != nil {
			return fmt.Errorf("failed to get data writer: %v", err)
		}
		defer writer.Close()
		if _, err = writer.Write([]byte(message)); err != nil {
			return fmt.Errorf("failed to write message: %v", err)
		}
		utils.Logger.Infof("SMTP: Email sent successfully to %s via %s", to, addr)
		return nil
	} else {
		// STARTTLS (e.g., port 587)
		client, err := smtp.Dial(addr)
		if err != nil {
			return fmt.Errorf("failed to dial SMTP server: %v", err)
		}
		defer client.Quit()

		// Upgrade to TLS
		tlsConfig := &tls.Config{
			ServerName:         es.SMTPHost,
			InsecureSkipVerify: false,
			MinVersion:         tls.VersionTLS12,
		}
		if err = client.StartTLS(tlsConfig); err != nil {
			return fmt.Errorf("failed to start TLS: %v", err)
		}

		auth := smtp.PlainAuth("", es.Username, es.Password, es.SMTPHost)
		if err = client.Auth(auth); err != nil {
			return fmt.Errorf("SMTP authentication failed: %v", err)
		}
		if err = client.Mail(es.Username); err != nil {
			return fmt.Errorf("failed to set sender: %v", err)
		}
		if err = client.Rcpt(to); err != nil {
			return fmt.Errorf("failed to set recipient: %v", err)
		}
		writer, err := client.Data()
		if err != nil {
			return fmt.Errorf("failed to get data writer: %v", err)
		}
		defer writer.Close()
		if _, err = writer.Write([]byte(message)); err != nil {
			return fmt.Errorf("failed to write message: %v", err)
		}
		utils.Logger.Infof("SMTP: Email sent successfully to %s via %s", to, addr)
		return nil
	}
}

// isValidEmail validates email format using proper email validation library
func isValidEmail(email string) bool {
	if len(email) == 0 || len(email) > 254 {
		return false
	}

	// Use proper email validation library
	err := checkmail.ValidateFormat(email)
	return err == nil
}
