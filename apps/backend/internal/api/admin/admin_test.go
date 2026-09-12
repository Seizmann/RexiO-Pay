package admin

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Seizmann/RexiO-Pay/backend/internal/apierr"
	"github.com/Seizmann/RexiO-Pay/backend/internal/middleware"
)

func TestValidAnnouncement(t *testing.T) {
	for _, test := range []struct {
		name     string
		request  announcementRequest
		wantCode string
		wantOK   bool
	}{
		{name: "defaults severity", request: announcementRequest{Title: "Maintenance", Body: "Soon"}, wantOK: true},
		{name: "requires title", request: announcementRequest{Body: "Soon"}, wantCode: apierr.CodeInvalidRequest},
		{name: "rejects severity", request: announcementRequest{Title: "x", Body: "y", Severity: "critical"}, wantCode: apierr.CodeInvalidRequest},
	} {
		t.Run(test.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			got := validAnnouncement(rec, &test.request)
			if got != test.wantOK {
				t.Fatalf("validAnnouncement() = %v, want %v", got, test.wantOK)
			}
			if test.wantCode != "" && rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
			}
		})
	}
}

func TestAdminContextIsAbsentByDefault(t *testing.T) {
	if got := middleware.GetMerchant(httptest.NewRequest(http.MethodGet, "/", nil).Context()); got != nil {
		t.Fatalf("merchant context = %#v, want nil", got)
	}
}
