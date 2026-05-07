package websites

import "testing"

func TestWebsiteRemoveConfirmationPrompt(t *testing.T) {
	tests := []struct {
		name         string
		websiteName  string
		contentCount int
		want         string
	}{
		{
			name:         "zero content",
			websiteName:  "test",
			contentCount: 0,
			want:         "Delete website 'test' and 0 content items?",
		},
		{
			name:         "single content",
			websiteName:  "test",
			contentCount: 1,
			want:         "Delete website 'test' and 1 content item?",
		},
		{
			name:         "multiple content",
			websiteName:  "test",
			contentCount: 2,
			want:         "Delete website 'test' and 2 content items?",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := websiteRemoveConfirmationPrompt(tt.websiteName, tt.contentCount)
			if got != tt.want {
				t.Fatalf("unexpected prompt: got %q want %q", got, tt.want)
			}
		})
	}
}

func TestWebsiteRemoveSuccessMessage(t *testing.T) {
	got := websiteRemoveSuccessMessage("test", 2)
	want := "Removed test and 2 content items"
	if got != want {
		t.Fatalf("unexpected message: got %q want %q", got, want)
	}
}
