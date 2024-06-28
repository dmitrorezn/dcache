package domain

type Feedback struct {
	ID         ID
	Anonymous  bool
	UserID     ID
	Content    string
	MediaIDs   []ID
	ShowStatus ShowStatus
}

type ShowStatus int32

const (
	Show = iota
	Hidden
	RandoShow
)

func NewFeedback(userID ID, anonymous bool, content string) Feedback {
	return Feedback{
		ID:        NewID(),
		Anonymous: anonymous,
		UserID:    userID,
		Content:   content,
	}
}

const (
	photo = "photo"
	video = "video"
	file  = "file"
)

type Media struct {
	ID   ID
	Type string
	URI  string
}
