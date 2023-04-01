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

func (o *writer) WriteReport(ctx context.Context, tests []allure.Test) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if o.pth != "" {
		if err := o.mkdir(); err != nil {
			return err
		}
	}

	for _, tc := range tests {
		if err := o.write(tc); err != nil {
			return fmt.Errorf("write test: %w", err)
		}
	}

	return nil
}

func (o *writer) WriteAttachments(ctx context.Context, attachments []Attachment) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if o.pth != "" {
		if err := o.mkdir(); err != nil {
			return err
		}

		for _, attachment := range attachments {
			if err := o.writeAttachmentFile(attachment); err != nil {
				return err
			}
		}
	}

	return nil
}

func (o *writer) writeAttachmentFile(attachment Attachment) error {
	pth := filepath.Join(o.pth, attachment.Source)
	file, err := os.OpenFile(pth, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return fmt.Errorf("os.OpenFile: %w", err)
	}

	defer file.Close()

	if _, err = file.Write(attachment.Body); err != nil {
		return fmt.Errorf("os.OpenFile Write: %w", err)
	}

	if err = file.Sync(); err != nil {
		return fmt.Errorf("os.OpenFile Sync: %w", err)
	}

	return nil
}

func (o *writer) write(tc allure.Test) error {
	writers := []io.Writer{o.w}

	if o.pth != "" {
		pth := filepath.Join(o.pth, fmt.Sprintf("%s-result.json", tc.UUID))

		file, err := os.OpenFile(pth, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
		if err != nil {
			return fmt.Errorf("os.OpenFile: %w", err)
		}

		defer file.Close()

		writers = append(writers, file)
	}

	w := io.MultiWriter(writers...)

	if err := json.NewEncoder(w).Encode(tc); err != nil {
		return fmt.Errorf("json.NewEncoder.Encode: %w", err)
	}

	return nil
}

func (o *writer) mkdir() error {
	if _, err := os.Stat(o.pth); os.IsNotExist(err) {
		if err = os.MkdirAll(o.pth, os.ModePerm); err != nil {
			return fmt.Errorf("os.MkdirAll: %w", err)
		}
	}

	return nil
}
