package storage

import (
	"fmt"

	"github.com/restream/reindexer/v3"
	_ "github.com/restream/reindexer/v3/bindings/cproto"
)

const (
	NamespaceName    = "documents"
	InitStorage      = "storage.InitStorage"
	CreateDocument   = "storage.CreateDocument"
	CheckCollections = "storage.CheckCollections"
	GetDocument      = "storage.GetDocument"
	GetDocuments     = "storage.GetDocuments"
	DeleteDocument   = "storage.DeleteDocument"
	UpdateDocument   = "storage.UpdateDocument"
)

type Document struct {
	Id                int                `reindex:"id,,pk" json:"id"`
	Title             string             `reindex:"title" json:"title"`
	ChildrenDocuments []ChildrenDocument `reindex:"children_documents" json:"childrenDocuments"`
}

type ChildrenDocument struct {
	Title             string             `reindex:"title" json:"title"`
	Sort              int                `reindex:"sort" json:"sort"`
	ChildrenDocuments []ChildrenDocument `reindex:"children_documents" json:"childrenDocuments"`
}

type Storage struct {
	rx *reindexer.Reindexer
}

func New(dsn, name string) (*Storage, error) {
	rx := reindexer.NewReindex(fmt.Sprintf("%v/%v", dsn, name), reindexer.WithCreateDBIfMissing())

	if err := rx.Status().Err; err != nil {
		return nil, fmt.Errorf("%v: %v", InitStorage, err)
	}

	if err := rx.OpenNamespace(NamespaceName, reindexer.DefaultNamespaceOptions(), Document{}); err != nil {
		return nil, fmt.Errorf("%v: %v", InitStorage, err)
	}

	return &Storage{rx: rx}, nil
}

func (s *Storage) Ping() error {
	return s.rx.Ping()
}

func (s *Storage) CheckCollections() error {
	iterator := s.rx.
		Query(NamespaceName).
		Exec()

	defer iterator.Close()

	if iterator.Count() != 0 {
		return nil
	}

	for i := 1; i < 50; i++ {
		if _, err := s.CreateDocument(Document{
			Title: fmt.Sprintf("Document %v", i),
			// Sort:        rand.Intn(10),
			// ChildrenIds: []int{},
		}); err != nil {
			return fmt.Errorf("%v: %v", CheckCollections, err)
		}

	}

	return nil
}

func (s *Storage) Close() {
	s.rx.Close()
}

func (s *Storage) CreateDocument(document Document) (Document, error) {
	if err := s.rx.Upsert(NamespaceName, &document, "id=serial()"); err != nil {
		return Document{}, fmt.Errorf("%v: %v", CreateDocument, err)
	}

	parentDocument, found, err := s.getParentDocument(document.ParentId)

	if err != nil {
		return Document{}, fmt.Errorf("%v: %v", CreateDocument, err)
	}

	if found {
		if err := s.toggleChildrenIds(&parentDocument, document.Id, "add"); err != nil {
			return Document{}, fmt.Errorf("%v: %v", UpdateDocument, err)
		}
	}

	return document, nil
}

func (s *Storage) GetDocument(id int) (Document, error) {
	document, found := s.rx.
		Query(NamespaceName).
		Where("id", reindexer.EQ, id).
		Get()

	if !found {
		return Document{}, fmt.Errorf("%v: %v", GetDocument, "The document is missing")
	}

	return *document.(*Document), nil
}

func (s *Storage) GetDocuments(limit, offset int, ids []int) ([]Document, error) {
	query := s.rx.Query(NamespaceName).
		Sort("sort", true).
		Offset(offset).
		Limit(limit)

	if len(ids) != 0 {
		query = query.Where("id", reindexer.EQ, ids)
	}

	iterator := query.Exec()
	defer iterator.Close()

	docs := make([]Document, 0, limit)

	for iterator.Next() {
		docs = append(docs, *iterator.Object().(*Document))
	}

	if err := iterator.Error(); err != nil {
		return []Document{}, fmt.Errorf("%v: %v", GetDocuments, err)
	}

	return docs, nil
}

func (s *Storage) DeleteDocument(documentId int) error {
	_, err := s.rx.
		Query(NamespaceName).
		Where("id", reindexer.EQ, documentId).
		Delete()

	if err != nil {
		return fmt.Errorf("%v: %v", DeleteDocument, err)
	}

	return nil
}

func (s *Storage) getParentDocument(parentId int) (Document, bool, error) {
	if parentId == 0 {
		return Document{}, false, nil
	}

	parentDocument, err := s.GetDocument(parentId)

	if err != nil {
		return Document{}, false, fmt.Errorf("%v: %v", GetDocument, "The parent document is missing")
	}

	return parentDocument, true, nil
}

func (s *Storage) UpdateDocument(document Document) (Document, error) {
	oldDocument, err := s.GetDocument(document.Id)

	if err != nil {
		return Document{}, fmt.Errorf("%v: %v", UpdateDocument, err)
	}

	oldParentDocument, foundOldParent, err := s.getParentDocument(oldDocument.ParentId)

	if err != nil {
		return Document{}, fmt.Errorf("%v: %v", UpdateDocument, err)
	}

	parentDocument, foundParent, err := s.getParentDocument(document.ParentId)

	if err != nil {
		return Document{}, fmt.Errorf("%v: %v", UpdateDocument, err)
	}

	if foundOldParent && oldDocument.ParentId != document.ParentId {
		if err := s.toggleChildrenIds(&oldParentDocument, document.Id, "delete"); err != nil {
			return Document{}, fmt.Errorf("%v: %v", UpdateDocument, err)
		}
	}

	if foundParent && oldDocument.ParentId != document.ParentId {
		if err := s.toggleChildrenIds(&parentDocument, document.Id, "add"); err != nil {
			return Document{}, fmt.Errorf("%v: %v", UpdateDocument, err)
		}
	}

	iterator := s.rx.
		Query(NamespaceName).
		Where("id", reindexer.EQ, document.Id).
		Set("title", document.Title).
		Set("sort", document.Sort).
		Set("parent_id", document.ParentId).
		Update()

	if err := iterator.Error(); err != nil {
		return Document{}, nil
	}

	for iterator.Next() {
		return *iterator.Object().(*Document), nil
	}

	return Document{}, nil
}

func (s *Storage) toggleChildrenIds(
	changedDocument *Document,
	currentDocumentId int,
	method string,
) error {
	query := s.rx.
		Query(NamespaceName).
		Where("id", reindexer.EQ, changedDocument.Id)

	switch method {
	case "add":
		changedDocument.ChildrenIds = append(changedDocument.ChildrenIds, currentDocumentId)

		iterator := query.
			Set("children_ids", changedDocument.ChildrenIds).
			Update()

		if err := iterator.Error(); err != nil {
			return fmt.Errorf("%v: %v", UpdateDocument, err)
		}
	case "delete":
		{
			childrenIds := make([]int, 0, len(changedDocument.ChildrenIds)-1)
			for _, v := range changedDocument.ChildrenIds {
				if v == currentDocumentId {
					continue
				}

				childrenIds = append(childrenIds, v)
			}

			changedDocument.ChildrenIds = childrenIds

			iterator := query.
				Set("children_ids", changedDocument.ChildrenIds).
				Update()

			if err := iterator.Error(); err != nil {
				return fmt.Errorf("%v: %v", UpdateDocument, err)
			}
		}
	case "default":
		{
			return fmt.Errorf("%v: %v", UpdateDocument, "Method is not allowed")
		}
	}

	return nil
}
