package admin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSuperAdminOnlyRejectsCampaignAdmin(t *testing.T) {
	handler := superAdminOnly(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	request := httptest.NewRequest(http.MethodGet, "/admin/admins", nil)
	request = request.WithContext(context.WithValue(request.Context(), adminContextKey, LoginSession{Role: RoleCampaignAdmin}))
	response := httptest.NewRecorder()
	handler(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("status campaign admin = %d", response.Code)
	}

	request = httptest.NewRequest(http.MethodGet, "/admin/admins", nil)
	request = request.WithContext(context.WithValue(request.Context(), adminContextKey, LoginSession{Role: RoleSuperAdmin}))
	response = httptest.NewRecorder()
	handler(response, request)
	if response.Code != http.StatusNoContent {
		t.Fatalf("status super admin = %d", response.Code)
	}
}
