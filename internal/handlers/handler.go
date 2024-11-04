package handler

import (
	"documents/internal/cache"
	st "documents/internal/storage"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"sync"

	"github.com/go-chi/chi"
	"github.com/go-chi/render"
)

type ResponseWrapper struct {
	Message string `json:"message"`
}

type Handler struct {
	storage *st.Storage
	cache   *cache.Cache
}

const (
	CreateDocument = "handler.CreateDocument"
	GetDocument    = "handler.GetDocument"
	GetDocuments   = "handler.GetDocuments"
	DeleteDocument = "handler.DeleteDocument"
	UpdateDocument = "handler.UpdateDocument"
)

func New(storage *st.Storage, cache *cache.Cache) *Handler {
	return &Handler{
		storage: storage,
		cache:   cache,
	}
}

func (h *Handler) GetDocuments(w http.ResponseWriter, r *http.Request) {
	page := 1
	limit := 10

	queryPage, err := strconv.Atoi(r.URL.Query().Get("page"))

	if err == nil {
		page = queryPage
	}

	queryLimit, err := strconv.Atoi(r.URL.Query().Get("limit"))

	if err == nil {
		limit = queryLimit
	}

	queryIds := r.URL.Query().Get("ids")
	var ids []int

	err = json.Unmarshal([]byte(queryIds), &ids)

	if queryIds != "" && err != nil {
		log.Printf("%v: %v", GetDocuments, err)
		responseWrapper(w, r, http.StatusBadRequest, ResponseWrapper{
			Message: "Invalid ids",
		})
		return
	}

	documents, err := h.storage.GetDocuments(limit, (page-1)*limit, ids)

	if err != nil {
		log.Printf("%v: %v", GetDocuments, err)
		responseWrapper(w, r, http.StatusBadRequest, ResponseWrapper{
			Message: "Failed to get a list of documents",
		})
		return
	}

	var wg sync.WaitGroup

	for i := range documents {
		wg.Add(1)

		go func(index int) {
			defer wg.Done()

			documents[index].Title = documents[index].Title + " обработан в горутине"
		}(i)
	}

	wg.Wait()

	responseWrapper(w, r, http.StatusOK, documents)
}

func (h *Handler) GetDocument(w http.ResponseWriter, r *http.Request) {
	documentId, err := strconv.Atoi(chi.URLParam(r, "id"))

	if err != nil {
		responseWrapper(w, r, http.StatusBadRequest, ResponseWrapper{
			Message: "Invalid Document ID value",
		})
		return
	}

	cachedDocument, found := h.cache.Get(documentId)

	if found {
		responseWrapper(w, r, http.StatusOK, cachedDocument)
		return
	}

	document, err := h.storage.GetDocument(documentId)

	if err != nil {
		log.Printf("%v: %v", GetDocument, err)
		responseWrapper(w, r, http.StatusBadRequest, ResponseWrapper{
			Message: "An error occurred while receiving the document",
		})
		return
	}

	h.cache.Set(document)

	responseWrapper(w, r, http.StatusOK, document)
}

func (h *Handler) CreateDocument(w http.ResponseWriter, r *http.Request) {
	var request st.Document

	err := render.DecodeJSON(r.Body, &request)

	if err != nil {
		log.Printf("%v: %v", CreateDocument, err)
		responseWrapper(w, r, http.StatusBadRequest, ResponseWrapper{
			Message: "Failed to decode request",
		})
		return
	}

	request.ChildrenIds = []int{}

	if err := validate(request); err != "" {
		responseWrapper(w, r, http.StatusBadRequest, ResponseWrapper{
			Message: err,
		})
		return
	}

	document, err := h.storage.CreateDocument(request)

	if err != nil {
		log.Printf("%v: %v", CreateDocument, err)
		responseWrapper(w, r, http.StatusBadRequest, ResponseWrapper{
			Message: "Error when creating a document",
		})
		return
	}

	responseWrapper(w, r, http.StatusOK, document)
}

func (h *Handler) DeleteDocument(w http.ResponseWriter, r *http.Request) {
	documentId, err := strconv.Atoi(chi.URLParam(r, "id"))

	if err != nil {
		responseWrapper(w, r, http.StatusBadRequest, ResponseWrapper{
			Message: "Invalid Document ID value",
		})
		return
	}

	if err := h.storage.DeleteDocument(documentId); err != nil {
		log.Printf("%v: %v", DeleteDocument, err)
		responseWrapper(w, r, http.StatusBadRequest, ResponseWrapper{
			Message: "An error occurred when deleting the document",
		})
		return
	}

	responseWrapper(w, r, http.StatusNoContent, ResponseWrapper{})
}

func (h *Handler) UpdateDocument(w http.ResponseWriter, r *http.Request) {
	documentId, err := strconv.Atoi(chi.URLParam(r, "id"))

	if err != nil {
		responseWrapper(w, r, http.StatusBadRequest, ResponseWrapper{
			Message: "Invalid Document ID value",
		})
		return
	}

	var request st.Document

	err = render.DecodeJSON(r.Body, &request)

	request.Id = documentId
	request.ChildrenIds = []int{}

	if err != nil {
		log.Printf("%v: %v", UpdateDocument, err)
		responseWrapper(w, r, http.StatusBadRequest, ResponseWrapper{
			Message: "Failed to decode request",
		})
		return
	}

	if err := validate(request); err != "" {
		responseWrapper(w, r, http.StatusBadRequest, ResponseWrapper{
			Message: err,
		})
		return
	}

	document, err := h.storage.UpdateDocument(request)

	if err != nil {
		log.Printf("%v: %v", UpdateDocument, err)
		responseWrapper(w, r, http.StatusBadRequest, ResponseWrapper{
			Message: "Error when editing a document",
		})
		return
	}

	responseWrapper(w, r, http.StatusOK, document)
}

func validate(doc st.Document) string {
	if doc.Title == "" {
		return "Invalid title"
	}

	if doc.Id != 0 && doc.ParentId != 0 && doc.ParentId == doc.Id {
		return "A document cannot be a parent of itself"
	}

	return ""
}

func responseWrapper[T any](
	w http.ResponseWriter,
	r *http.Request,
	code int,
	response T,
) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	render.JSON(w, r, response)
}
