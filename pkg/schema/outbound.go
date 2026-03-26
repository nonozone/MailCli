package schema

type DraftMessage struct {
	Account     string            `json:"account,omitempty" yaml:"account,omitempty"`
	From        *Address          `json:"from,omitempty" yaml:"from,omitempty"`
	To          []Address         `json:"to,omitempty" yaml:"to,omitempty"`
	Cc          []Address         `json:"cc,omitempty" yaml:"cc,omitempty"`
	Bcc         []Address         `json:"bcc,omitempty" yaml:"bcc,omitempty"`
	Subject     string            `json:"subject,omitempty" yaml:"subject,omitempty"`
	BodyMD      string            `json:"body_md,omitempty" yaml:"body_md,omitempty"`
	BodyText    string            `json:"body_text,omitempty" yaml:"body_text,omitempty"`
	Headers     map[string]string `json:"headers,omitempty" yaml:"headers,omitempty"`
	Attachments []Attachment      `json:"attachments,omitempty" yaml:"attachments,omitempty"`
}

type ReplyDraft struct {
	Account          string       `json:"account,omitempty" yaml:"account,omitempty"`
	From             *Address     `json:"from,omitempty" yaml:"from,omitempty"`
	To               []Address    `json:"to,omitempty" yaml:"to,omitempty"`
	Cc               []Address    `json:"cc,omitempty" yaml:"cc,omitempty"`
	Bcc              []Address    `json:"bcc,omitempty" yaml:"bcc,omitempty"`
	Subject          string       `json:"subject,omitempty" yaml:"subject,omitempty"`
	BodyMD           string       `json:"body_md,omitempty" yaml:"body_md,omitempty"`
	BodyText         string       `json:"body_text,omitempty" yaml:"body_text,omitempty"`
	ReplyToID        string       `json:"reply_to_id,omitempty" yaml:"reply_to_id,omitempty"`
	ReplyToMessageID string       `json:"reply_to_message_id,omitempty" yaml:"reply_to_message_id,omitempty"`
	References       []string     `json:"references,omitempty" yaml:"references,omitempty"`
	Attachments      []Attachment `json:"attachments,omitempty" yaml:"attachments,omitempty"`
}

type Attachment struct {
	Name        string `json:"name,omitempty" yaml:"name,omitempty"`
	ContentType string `json:"content_type,omitempty" yaml:"content_type,omitempty"`
	Path        string `json:"path,omitempty" yaml:"path,omitempty"`
}

type SendResult struct {
	OK        bool       `json:"ok" yaml:"ok"`
	MessageID string     `json:"message_id,omitempty" yaml:"message_id,omitempty"`
	ThreadID  string     `json:"thread_id,omitempty" yaml:"thread_id,omitempty"`
	Provider  string     `json:"provider,omitempty" yaml:"provider,omitempty"`
	Account   string     `json:"account,omitempty" yaml:"account,omitempty"`
	Error     *SendError `json:"error,omitempty" yaml:"error,omitempty"`
}

type SendError struct {
	Code    string `json:"code,omitempty" yaml:"code,omitempty"`
	Message string `json:"message,omitempty" yaml:"message,omitempty"`
}
