package exporter

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/robotomize/go-allure/internal/allure"
)

type Writer interface {
	WriteReport(ctx context.Context, tests []allure.Test) error
	WriteAttachments(ctx context.Context, attachments []Attachment) error
}

type WriterOption func(*writer)

func WithOutputPth(pth string) WriterOption {
	return func(w *writer) {
		w.pth = pth
	}
}

func NewWriter(w1 io.Writer, opts ...WriterOption) Writer {
	w := writer{w: w1}
	for _, o := range opts {
		o(&w)
	}

	return &w
}

type writer struct {
	pth string
	w   io.Writer
}

// WriteReport - write allure report to the given path.
func (o *writer) WriteReport(ctx context.Context, tests []allure.Test) error {
	// Check if the context is done to return early.
	if err := ctx.Err(); err != nil {
		return err
	}

	if o.pth == "" {
		return nil
	}

	// Create the necessary directories in the file system if they don't exist.
	if err := o.mkdir(); err != nil {
		return err
	}

	// Loop through the Test objects and write each one to a separate text file.
	for _, tc := range tests {
		if err := o.write(tc); err != nil {
			return fmt.Errorf("write test: %w", err)
		}
	}

	return nil
}

// WriteAttachments writes the attachments to the given path.
func (o *writer) WriteAttachments(ctx context.Context, attachments []Attachment) error {
	// Return an error if the context is canceled.
	if err := ctx.Err(); err != nil {
		return err
	}

	if o.pth == "" {
		return nil
	}

	// Create the directory if it does not exist.
	if err := o.mkdir(); err != nil {
		return err
	}

	// Write each attachment file to disk.
	for _, attachment := range attachments {
		if err := o.writeAttachmentFile(attachment); err != nil {
			return err
		}
	}

	return nil
}

// writeAttachmentFile writes the attachment file to the specified path.
func (o *writer) writeAttachmentFile(attachment Attachment) error {
	// Get the file path.
	pth := filepath.Join(o.pth, attachment.Source)

	// Open the file for writing.
	file, err := os.OpenFile(pth, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return fmt.Errorf("os.OpenFile: %w", err)
	}

	// Close the file after the function is done executing.
	defer file.Close()

	// Write the attachment body to the file.
	if _, err = file.Write(attachment.Body); err != nil {
		return fmt.Errorf("os.OpenFile Write: %w", err)
	}

	// Sync the file to disk to ensure the data is actually written.
	if err = file.Sync(); err != nil {
		return fmt.Errorf("os.OpenFile Sync: %w", err)
	}

	return nil
}

// write writes the test result to the specified path, if provided.
func (o *writer) write(tc allure.Test) error {
	// Use the console output and file output writers, if path is given.
	writers := []io.Writer{o.w}
	if o.pth != "" {
		// Open file for write and 0644 permissions
		pth := filepath.Join(o.pth, fmt.Sprintf("%s-result.json", tc.UUID))
		file, err := os.OpenFile(pth, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
		if err != nil {
			return fmt.Errorf("os.OpenFile: %w", err)
		}
		defer file.Close()
		writers = append(writers, file)
	}

	w := io.MultiWriter(writers...)

	// Encode the test result in JSON format and write it to the console and file output.
	if err := json.NewEncoder(w).Encode(tc); err != nil {
		return fmt.Errorf("json.NewEncoder.Encode: %w", err)
	}

	return nil
}

// mkdir checks if the provided path exists and creates it if it does not.
func (o *writer) mkdir() error {
	// Check if the directory already exists.
	if _, err := os.Stat(o.pth); os.IsNotExist(err) {
		// Create the directory if it does not exist.
		if err = os.MkdirAll(o.pth, os.ModePerm); err != nil {
			return fmt.Errorf("os.MkdirAll: %w", err)
		}
	}

	return nil
}
