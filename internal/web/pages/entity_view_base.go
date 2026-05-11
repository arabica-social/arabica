package pages

import "tangled.org/arabica.social/arabica/internal/firehose"

// EntityViewBase holds the social and auth fields shared by all simple
// entity view pages across both apps (arabica's bean/roaster/grinder/brewer,
// oolong's tea/vendor/brewer/cafe/drink). Embed it in XxxViewProps so the
// fields are promoted (props.IsAuthenticated etc. in templ).
type EntityViewBase struct {
	IsOwnProfile      bool
	IsAuthenticated   bool
	SubjectURI        string
	SubjectCID        string
	IsLiked           bool
	LikeCount         int
	CommentCount      int
	Comments          []firehose.IndexedComment
	CurrentUserDID    string
	ShareURL          string
	IsModerator       bool
	CanHideRecord     bool
	CanBlockUser      bool
	IsRecordHidden    bool
	AuthorDID         string
	AuthorHandle      string
	AuthorDisplayName string
	AuthorAvatar      string
}
