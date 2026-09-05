package websites

import "testing"

func TestWebsiteRemoveConfirmationPrompt(t *testing.T) {
	tests := []struct {
		name         string
		websiteName  string
		contentCount int
		uploadCount  int
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
		{
			name:         "uploads keep the website",
			websiteName:  "test",
			contentCount: 2,
			uploadCount:  1,
			want:         "Delete 2 content items from 'test'? The website is kept, it still has 1 upload.",
		},
		{
			name:         "multiple uploads keep the website",
			websiteName:  "test",
			contentCount: 1,
			uploadCount:  3,
			want:         "Delete 1 content item from 'test'? The website is kept, it still has 3 uploads.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := websiteRemoveConfirmationPrompt(tt.websiteName, tt.contentCount, tt.uploadCount)
			if got != tt.want {
				t.Fatalf("unexpected prompt: got %q want %q", got, tt.want)
			}
		})
	}
}

func TestWebsiteRemoveSuccessMessage(t *testing.T) {
	tests := []struct {
		name         string
		contentCount int
		uploadCount  int
		want         string
	}{
		{
			name:         "website removed",
			contentCount: 2,
			want:         "Removed test and 2 content items",
		},
		{
			name:         "website kept for one upload",
			contentCount: 2,
			uploadCount:  1,
			want:         "Removed 2 content items, kept test because it still has 1 upload",
		},
		{
			name:         "website kept for several uploads",
			contentCount: 2,
			uploadCount:  3,
			want:         "Removed 2 content items, kept test because it still has 3 uploads",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := websiteRemoveSuccessMessage("test", tt.contentCount, tt.uploadCount)
			if got != tt.want {
				t.Fatalf("unexpected message: got %q want %q", got, tt.want)
			}
		})
	}
}
