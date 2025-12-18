// pantry/email/template.go
// Template rendering for email content.
package email

import (
	"bytes"
	"context"
	"fmt"
	htmltemplate "html/template"
	"sync"
	texttemplate "text/template"
	"time"
)

// TemplateRenderer renders email content from templates.
type TemplateRenderer struct {
	mu        sync.RWMutex
	textTpls  map[string]*texttemplate.Template
	htmlTpls  map[string]*htmltemplate.Template
	basePath  string
	funcMap   map[string]any
}

// TemplateConfig configures the template renderer.
type TemplateConfig struct {
	// BasePath is the directory containing templates.
	BasePath string

	// FuncMap provides custom template functions.
	FuncMap map[string]any
}

// NewTemplateRenderer creates a new template renderer.
func NewTemplateRenderer(cfg TemplateConfig) *TemplateRenderer {
	return &TemplateRenderer{
		textTpls: make(map[string]*texttemplate.Template),
		htmlTpls: make(map[string]*htmltemplate.Template),
		basePath: cfg.BasePath,
		funcMap:  cfg.FuncMap,
	}
}

// LoadText loads a text template from a string.
func (r *TemplateRenderer) LoadText(name, content string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	tpl := texttemplate.New(name)
	if r.funcMap != nil {
		tpl = tpl.Funcs(texttemplate.FuncMap(r.funcMap))
	}

	parsed, err := tpl.Parse(content)
	if err != nil {
		return fmt.Errorf("email: failed to parse text template %s: %w", name, err)
	}

	r.textTpls[name] = parsed
	return nil
}

// LoadHTML loads an HTML template from a string.
func (r *TemplateRenderer) LoadHTML(name, content string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	tpl := htmltemplate.New(name)
	if r.funcMap != nil {
		tpl = tpl.Funcs(htmltemplate.FuncMap(r.funcMap))
	}

	parsed, err := tpl.Parse(content)
	if err != nil {
		return fmt.Errorf("email: failed to parse HTML template %s: %w", name, err)
	}

	r.htmlTpls[name] = parsed
	return nil
}

// LoadTextFile loads a text template from a file.
func (r *TemplateRenderer) LoadTextFile(name, path string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	tpl := texttemplate.New(name)
	if r.funcMap != nil {
		tpl = tpl.Funcs(texttemplate.FuncMap(r.funcMap))
	}

	fullPath := path
	if r.basePath != "" {
		fullPath = r.basePath + "/" + path
	}

	parsed, err := tpl.ParseFiles(fullPath)
	if err != nil {
		return fmt.Errorf("email: failed to load text template %s: %w", name, err)
	}

	r.textTpls[name] = parsed
	return nil
}

// LoadHTMLFile loads an HTML template from a file.
func (r *TemplateRenderer) LoadHTMLFile(name, path string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	tpl := htmltemplate.New(name)
	if r.funcMap != nil {
		tpl = tpl.Funcs(htmltemplate.FuncMap(r.funcMap))
	}

	fullPath := path
	if r.basePath != "" {
		fullPath = r.basePath + "/" + path
	}

	parsed, err := tpl.ParseFiles(fullPath)
	if err != nil {
		return fmt.Errorf("email: failed to load HTML template %s: %w", name, err)
	}

	r.htmlTpls[name] = parsed
	return nil
}

// RenderText renders a text template with the given data.
func (r *TemplateRenderer) RenderText(name string, data any) (string, error) {
	r.mu.RLock()
	tpl, exists := r.textTpls[name]
	r.mu.RUnlock()

	if !exists {
		return "", fmt.Errorf("email: text template %s not found", name)
	}

	var buf bytes.Buffer
	if err := tpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("email: failed to render text template %s: %w", name, err)
	}

	return buf.String(), nil
}

// RenderHTML renders an HTML template with the given data.
func (r *TemplateRenderer) RenderHTML(name string, data any) (string, error) {
	r.mu.RLock()
	tpl, exists := r.htmlTpls[name]
	r.mu.RUnlock()

	if !exists {
		return "", fmt.Errorf("email: HTML template %s not found", name)
	}

	var buf bytes.Buffer
	if err := tpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("email: failed to render HTML template %s: %w", name, err)
	}

	return buf.String(), nil
}

// RenderMessage renders both text and HTML templates into a message.
func (r *TemplateRenderer) RenderMessage(textName, htmlName string, data any) (textBody, htmlBody string, err error) {
	if textName != "" {
		textBody, err = r.RenderText(textName, data)
		if err != nil {
			return "", "", err
		}
	}

	if htmlName != "" {
		htmlBody, err = r.RenderHTML(htmlName, data)
		if err != nil {
			return "", "", err
		}
	}

	return textBody, htmlBody, nil
}

// EmailTemplate represents a paired text/HTML template for an email type.
type EmailTemplate struct {
	Name        string // Template name/identifier
	Subject     string // Subject line (can use Go template syntax)
	TextBody    string // Text template content
	HTMLBody    string // HTML template content
}

// TemplateStore manages email templates.
type TemplateStore struct {
	mu        sync.RWMutex
	templates map[string]*compiledTemplate
}

type compiledTemplate struct {
	subject  *texttemplate.Template
	textBody *texttemplate.Template
	htmlBody *htmltemplate.Template
}

// NewTemplateStore creates a new template store.
func NewTemplateStore() *TemplateStore {
	return &TemplateStore{
		templates: make(map[string]*compiledTemplate),
	}
}

// Register registers an email template.
func (s *TemplateStore) Register(tpl EmailTemplate) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	compiled := &compiledTemplate{}

	// Parse subject
	if tpl.Subject != "" {
		t, err := texttemplate.New(tpl.Name + "_subject").Parse(tpl.Subject)
		if err != nil {
			return fmt.Errorf("email: failed to parse subject template: %w", err)
		}
		compiled.subject = t
	}

	// Parse text body
	if tpl.TextBody != "" {
		t, err := texttemplate.New(tpl.Name + "_text").Parse(tpl.TextBody)
		if err != nil {
			return fmt.Errorf("email: failed to parse text template: %w", err)
		}
		compiled.textBody = t
	}

	// Parse HTML body
	if tpl.HTMLBody != "" {
		t, err := htmltemplate.New(tpl.Name + "_html").Parse(tpl.HTMLBody)
		if err != nil {
			return fmt.Errorf("email: failed to parse HTML template: %w", err)
		}
		compiled.htmlBody = t
	}

	s.templates[tpl.Name] = compiled
	return nil
}

// Render renders a template with the given data into a Message.
func (s *TemplateStore) Render(name string, data any) (*Message, error) {
	s.mu.RLock()
	tpl, exists := s.templates[name]
	s.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("email: template %s not found", name)
	}

	msg := &Message{}

	// Render subject
	if tpl.subject != nil {
		var buf bytes.Buffer
		if err := tpl.subject.Execute(&buf, data); err != nil {
			return nil, fmt.Errorf("email: failed to render subject: %w", err)
		}
		msg.Subject = buf.String()
	}

	// Render text body
	if tpl.textBody != nil {
		var buf bytes.Buffer
		if err := tpl.textBody.Execute(&buf, data); err != nil {
			return nil, fmt.Errorf("email: failed to render text body: %w", err)
		}
		msg.TextBody = buf.String()
	}

	// Render HTML body
	if tpl.htmlBody != nil {
		var buf bytes.Buffer
		if err := tpl.htmlBody.Execute(&buf, data); err != nil {
			return nil, fmt.Errorf("email: failed to render HTML body: %w", err)
		}
		msg.HTMLBody = buf.String()
	}

	return msg, nil
}

// Has returns true if a template exists.
func (s *TemplateStore) Has(name string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, exists := s.templates[name]
	return exists
}

// List returns the names of all registered templates.
func (s *TemplateStore) List() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	names := make([]string, 0, len(s.templates))
	for name := range s.templates {
		names = append(names, name)
	}
	return names
}

// TemplateSender wraps a Sender with template support.
type TemplateSender struct {
	sender *Sender
	store  *TemplateStore
}

// NewTemplateSender creates a new template-aware sender.
func NewTemplateSender(sender *Sender, store *TemplateStore) *TemplateSender {
	return &TemplateSender{
		sender: sender,
		store:  store,
	}
}

// Send sends an email using a template.
func (s *TemplateSender) Send(ctx context.Context, to []string, templateName string, data any) error {
	msg, err := s.store.Render(templateName, data)
	if err != nil {
		return err
	}

	msg.To = to
	return s.sender.Send(ctx, *msg)
}

// SendTo sends an email to a single recipient using a template.
func (s *TemplateSender) SendTo(ctx context.Context, to, templateName string, data any) error {
	return s.Send(ctx, []string{to}, templateName, data)
}

// TemplateQueue wraps a Queue with template support.
type TemplateQueue struct {
	queue *Queue
	store *TemplateStore
}

// NewTemplateQueue creates a new template-aware queue.
func NewTemplateQueue(queue *Queue, store *TemplateStore) *TemplateQueue {
	return &TemplateQueue{
		queue: queue,
		store: store,
	}
}

// Enqueue renders a template and queues the email.
func (q *TemplateQueue) Enqueue(ctx context.Context, to []string, templateName string, data any) (string, error) {
	msg, err := q.store.Render(templateName, data)
	if err != nil {
		return "", err
	}

	msg.To = to
	return q.queue.EnqueueMessage(ctx, *msg)
}

// EnqueueTo renders a template and queues the email to a single recipient.
func (q *TemplateQueue) EnqueueTo(ctx context.Context, to, templateName string, data any) (string, error) {
	return q.Enqueue(ctx, []string{to}, templateName, data)
}

// Schedule renders a template and schedules the email for later delivery.
func (q *TemplateQueue) Schedule(ctx context.Context, to []string, templateName string, data any, at time.Time) (string, error) {
	msg, err := q.store.Render(templateName, data)
	if err != nil {
		return "", err
	}

	msg.To = to
	email := &QueuedEmail{Message: *msg}
	if err := q.queue.Schedule(ctx, email, at); err != nil {
		return "", err
	}
	return email.ID, nil
}

// Queue returns the underlying queue.
func (q *TemplateQueue) Queue() *Queue {
	return q.queue
}

// Store returns the template store.
func (q *TemplateQueue) Store() *TemplateStore {
	return q.store
}

// Common email templates.
var (
	// WelcomeTemplate is a basic welcome email template.
	WelcomeTemplate = EmailTemplate{
		Name:    "welcome",
		Subject: "Welcome to {{.AppName}}!",
		TextBody: `Hi {{.Name}},

Welcome to {{.AppName}}! We're excited to have you on board.

{{if .VerifyURL}}Please verify your email by visiting:
{{.VerifyURL}}{{end}}

Best regards,
The {{.AppName}} Team`,
		HTMLBody: `<!DOCTYPE html>
<html>
<body>
<p>Hi {{.Name}},</p>
<p>Welcome to <strong>{{.AppName}}</strong>! We're excited to have you on board.</p>
{{if .VerifyURL}}<p><a href="{{.VerifyURL}}">Click here to verify your email</a></p>{{end}}
<p>Best regards,<br>The {{.AppName}} Team</p>
</body>
</html>`,
	}

	// PasswordResetTemplate is a basic password reset email template.
	PasswordResetTemplate = EmailTemplate{
		Name:    "password_reset",
		Subject: "Reset Your Password",
		TextBody: `Hi {{.Name}},

We received a request to reset your password. Click the link below to create a new password:

{{.ResetURL}}

This link will expire in {{.ExpiresIn}}.

If you didn't request this, please ignore this email.

Best regards,
The {{.AppName}} Team`,
		HTMLBody: `<!DOCTYPE html>
<html>
<body>
<p>Hi {{.Name}},</p>
<p>We received a request to reset your password. Click the button below to create a new password:</p>
<p><a href="{{.ResetURL}}" style="background-color: #007bff; color: white; padding: 10px 20px; text-decoration: none; border-radius: 5px;">Reset Password</a></p>
<p>This link will expire in {{.ExpiresIn}}.</p>
<p>If you didn't request this, please ignore this email.</p>
<p>Best regards,<br>The {{.AppName}} Team</p>
</body>
</html>`,
	}

	// EmailVerificationTemplate is a basic email verification template.
	EmailVerificationTemplate = EmailTemplate{
		Name:    "email_verification",
		Subject: "Verify Your Email Address",
		TextBody: `Hi {{.Name}},

Please verify your email address by clicking the link below:

{{.VerifyURL}}

This link will expire in {{.ExpiresIn}}.

Best regards,
The {{.AppName}} Team`,
		HTMLBody: `<!DOCTYPE html>
<html>
<body>
<p>Hi {{.Name}},</p>
<p>Please verify your email address by clicking the button below:</p>
<p><a href="{{.VerifyURL}}" style="background-color: #28a745; color: white; padding: 10px 20px; text-decoration: none; border-radius: 5px;">Verify Email</a></p>
<p>This link will expire in {{.ExpiresIn}}.</p>
<p>Best regards,<br>The {{.AppName}} Team</p>
</body>
</html>`,
	}
)

// RegisterCommonTemplates registers the common email templates.
func (s *TemplateStore) RegisterCommonTemplates() error {
	templates := []EmailTemplate{
		WelcomeTemplate,
		PasswordResetTemplate,
		EmailVerificationTemplate,
	}

	for _, tpl := range templates {
		if err := s.Register(tpl); err != nil {
			return err
		}
	}

	return nil
}
