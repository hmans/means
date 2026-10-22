package collection

import (
	"fmt"
	"os"
)

// DeleteResult identifies the document that was permanently removed.
type DeleteResult struct {
	ID string `json:"id"`
}

// Delete removes a document only when its stored bytes match hash. Like Update,
// this is optimistic: a concurrent write can race the final removal.
// Application data need not satisfy the current schema, but the envelope must
// be readable and the ID must be unique within the selected collection.
func (c *Collection) Delete(id, hash string) (*DeleteResult, error) {
	if err := validateDocumentID(id); err != nil {
		return nil, err
	}
	if !validHash.MatchString(hash) {
		return nil, fmt.Errorf("hash is required and must be 64 lowercase hexadecimal SHA-256 characters; read the document first")
	}
	root, err := os.OpenRoot(c.Path)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	doc, err := readDocument(root, id)
	if err != nil {
		return nil, err
	}
	if doc.Hash != hash {
		return nil, conflict(id)
	}
	if err := root.Remove(doc.filename); err != nil {
		return nil, fmt.Errorf("delete document: %w", err)
	}
	return &DeleteResult{ID: id}, nil
}
