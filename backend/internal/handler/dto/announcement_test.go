package dto

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestUserAnnouncementFromServicePreservesPublisherSource(t *testing.T) {
	adminID := int64(42)
	tests := []struct {
		name      string
		createdBy *int64
	}{
		{name: "system", createdBy: nil},
		{name: "administrator", createdBy: &adminID},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := UserAnnouncementFromService(&service.UserAnnouncement{
				Announcement: service.Announcement{ID: 1, CreatedBy: tt.createdBy},
			})
			require.NotNil(t, got)
			require.Equal(t, tt.createdBy, got.CreatedBy)
		})
	}
}
