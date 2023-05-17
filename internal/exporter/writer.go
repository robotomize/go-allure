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

func WriteToFile(pth string) WriterOption {
	return func(w *writer) {
		w.pth = pth
	}
}

func WriteReportTo(writers ...io.Writer) WriterOption {
	return func(w *writer) {
		w.reportWriters = append(w.reportWriters, writers...)
	}
}

func NewWriter(opts ...WriterOption) Writer {
	w := writer{reportWriters: []io.Writer{io.Discard}}
	for _, o := range opts {
		o(&w)
	}

	return &w
}

type writer struct {
	pth           string
	reportWriters []io.Writer
}

// WriteReport - writeReport allure report to the given path.
func (o *writer) WriteReport(ctx context.Context, tests []allure.Test) error {
	// Check if the context is done to return early.
	if err := ctx.Err(); err != nil {
		return err
	}

	// Create the necessary directories in the file system if they don't exist.
	if len(o.pth) > 0 {
		if err := mkdir(o.pth); err != nil {
			return err
		}
	}

	// Loop through the Test objects and writeReport each one to a separate text file.
	for _, tc := range tests {
		if err := o.writeReport(tc); err != nil {
			return fmt.Errorf("writeReport test: %w", err)
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
	if err := mkdir(o.pth); err != nil {
		return fmt.Errorf("mkdir: %w", err)
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

// writeReport writes the test result to the specified path, if provided.
func (o *writer) writeReport(tc allure.Test) (err error) {
	// Use the console output and file output writers, if path is given.
	writers := make([]io.Writer, len(o.reportWriters))
	copy(writers, o.reportWriters)

	if o.pth != "" {
		// Open file for writeReport and 0644 permissions
		pth := filepath.Join(o.pth, fmt.Sprintf("%s-result.json", tc.UUID))
		file, openErr := os.OpenFile(pth, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
		if openErr != nil {
			return fmt.Errorf("os.OpenFile: %w", openErr)
		}

		defer func() {
			if syncErr := file.Sync(); syncErr != nil {
				err = fmt.Errorf("file Sync: %w", syncErr)
			}

			_ = file.Close()
		}()

		writers = append(writers, file)
	}

	w := io.MultiWriter(writers...)

	// Encode the test result in JSON format and writeReport it to the console and file output.
	if encErr := json.NewEncoder(w).Encode(tc); encErr != nil {
		return fmt.Errorf("json.NewEncoder.Encode: %w", encErr)
	}

	return nil
}
