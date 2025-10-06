package ws

import (
	"encoding/json"
	"net/http"

	"github.com/jibitesh/request-response-manager/internal/store"
)

func SessionLookupHandler(service *store.SessionService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Path[len("/session/"):]
		if id == "" {
			http.Error(w, "Invalid session id", http.StatusBadRequest)
			return
		}
		si, err := service.GetSession(r.Context(), id)
		if err == store.ErrNotFound {
			http.Error(w, "Session not found", http.StatusNotFound)
			return
		} else if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(si)
	}
}
