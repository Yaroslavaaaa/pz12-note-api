package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"example.com/notes-api/internal/core"
	"example.com/notes-api/internal/repo"
	"github.com/go-chi/chi/v5"
)

type Handler struct {
	Repo *repo.NoteRepoMem
}

type ErrorResponse struct {
	Error string `json:"error"`
}

type SuccessResponse struct {
	Message string `json:"message"`
}

type UpdateNoteRequest struct {
	Title   *string `json:"title"`
	Content *string `json:"content"`
}

// CreateNote godoc
// @Summary      Создать заметку
// @Tags         notes
// @Accept       json
// @Produce      json
// @Param        input  body     core.Note  true  "Данные новой заметки"
// @Success      201    {object} core.Note
// @Failure      400    {object} map[string]string
// @Failure      500    {object} map[string]string
// @Router       /notes [post]
func (h *Handler) CreateNote(w http.ResponseWriter, r *http.Request) {
	var n core.Note

	if err := json.NewDecoder(r.Body).Decode(&n); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	if strings.TrimSpace(n.Title) == "" {
		respondWithError(w, http.StatusBadRequest, "Title is required")
		return
	}

	id, err := h.Repo.Create(n)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to create note")
		return
	}

	createdNote, err := h.Repo.GetByID(id)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to retrieve created note")
		return
	}

	respondWithJSON(w, http.StatusCreated, createdNote)
}

// GetNote godoc
// @Summary      Получить заметку
// @Tags         notes
// @Param        id   path   int  true  "ID"
// @Success      200  {object}  core.Note
// @Failure      404  {object}  map[string]string
// @Router       /notes/{id} [get]
func (h *Handler) GetNote(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid note ID")
		return
	}

	note, err := h.Repo.GetByID(id)
	if err != nil {
		if err == repo.ErrNoteNotFound {
			respondWithError(w, http.StatusNotFound, "Note not found")
		} else {
			respondWithError(w, http.StatusInternalServerError, "Failed to get note")
		}
		return
	}

	respondWithJSON(w, http.StatusOK, note)
}

// ListNotes godoc
// @Summary      Список заметок
// @Description  Возвращает список заметок с пагинацией и фильтром по заголовку
// @Tags         notes
// @Param        page   query  int     false  "Номер страницы"
// @Param        limit  query  int     false  "Размер страницы"
// @Param        q      query  string  false  "Поиск по title"
// @Success      200    {array}  core.Note
// @Header       200    {integer}  X-Total-Count  "Общее количество"
// @Failure      500    {object}  map[string]string
// @Router       /notes [get]
func (h *Handler) ListNotes(w http.ResponseWriter, r *http.Request) {
	notes, err := h.Repo.GetAll()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to get notes")
		return
	}

	if notes == nil {
		notes = []core.Note{}
	}

	respondWithJSON(w, http.StatusOK, notes)
}

// PatchNote godoc
// @Summary      Обновить заметку (частично)
// @Tags         notes
// @Accept       json
// @Param        id     path   int        true  "ID"
// @Param        input  body   core.Note true  "Поля для обновления"
// @Success      200    {object}  core.Note
// @Failure      400    {object}  map[string]string
// @Failure      404    {object}  map[string]string
// @Router       /notes/{id} [patch]
func (h *Handler) PatchNote(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid note ID")
		return
	}

	var update UpdateNoteRequest
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	if update.Title == nil && update.Content == nil {
		respondWithError(w, http.StatusBadRequest, "No fields to update")
		return
	}

	if update.Title != nil && strings.TrimSpace(*update.Title) == "" {
		respondWithError(w, http.StatusBadRequest, "Title cannot be empty")
		return
	}

	updates := make(map[string]interface{})
	if update.Title != nil {
		updates["title"] = *update.Title
	}
	if update.Content != nil {
		updates["content"] = *update.Content
	}

	err = h.Repo.UpdatePartial(id, updates)
	if err != nil {
		if err == repo.ErrNoteNotFound {
			respondWithError(w, http.StatusNotFound, "Note not found")
		} else {
			respondWithError(w, http.StatusInternalServerError, "Failed to update note")
		}
		return
	}

	updatedNote, err := h.Repo.GetByID(id)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to retrieve updated note")
		return
	}

	respondWithJSON(w, http.StatusOK, updatedNote)
}

// DeleteNote godoc
// @Summary      Удалить заметку
// @Tags         notes
// @Param        id  path  int  true  "ID"
// @Success      204  "No Content"
// @Failure      404  {object}  map[string]string
// @Router       /notes/{id} [delete]
func (h *Handler) DeleteNote(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid note ID")
		return
	}

	err = h.Repo.Delete(id)
	if err != nil {
		if err == repo.ErrNoteNotFound {
			respondWithError(w, http.StatusNotFound, "Note not found")
		} else {
			respondWithError(w, http.StatusInternalServerError, "Failed to delete note")
		}
		return
	}

	respondWithJSON(w, http.StatusOK, SuccessResponse{
		Message: "Note deleted successfully",
	})
}

func respondWithError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(ErrorResponse{Error: message})
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	encoder.Encode(payload)
}
