package types

import "time"

type RepositoryContents struct {
	Files   []RepositoryFile `json:"files"`
	Commits []CommitInfo     `json:"commits"`
	Stats   RepositoryStats  `json:"stats"`
	Error   string           `json:"error,omitempty"`
}

type RepositoryFile struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Type string `json:"type"` // "file" or "dir"
	Size int64  `json:"size"`
	URL  string `json:"url"`
}

type CommitInfo struct {
	SHA     string    `json:"sha"`
	Message string    `json:"message"`
	Author  string    `json:"author"`
	Date    time.Time `json:"date"`
	URL     string    `json:"url"`
}

type RepositoryStats struct {
	TotalFiles  int            `json:"total_files"`
	TotalSize   int64          `json:"total_size"`
	Languages   map[string]int `json:"languages"`
	CommitCount int            `json:"commit_count"`
	LastUpdated time.Time      `json:"last_updated"`
}

type FileContent struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	Content     string `json:"content"`
	Size        int64  `json:"size"`
	Type        string `json:"type"`
	Language    string `json:"language"`
	IsText      bool   `json:"is_text"`
	IsBinary    bool   `json:"is_binary"`
	Encoding    string `json:"encoding"`
	SHA         string `json:"sha"`
	DownloadURL string `json:"download_url"`
	Error       string `json:"error,omitempty"`
}
