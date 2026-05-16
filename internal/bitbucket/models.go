package bitbucket

type PaginatedResponse[T any] struct {
	Values  []T    `json:"values"`
	Next    string `json:"next,omitempty"`
	Size    int    `json:"size,omitempty"`
	Page    int    `json:"page,omitempty"`
	PageLen int    `json:"pagelen,omitempty"`
}

type Workspace struct {
	Slug string `json:"slug"`
	Name string `json:"name"`
	UUID string `json:"uuid,omitempty"`
}

type Repository struct {
	Slug        string `json:"slug"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Language    string `json:"language,omitempty"`
	FullName    string `json:"full_name"`
	UpdatedOn   string `json:"updated_on"`
	IsPrivate   bool   `json:"is_private"`
}

type PullRequestState string

const (
	PullRequestStateOpen       PullRequestState = "OPEN"
	PullRequestStateMerged     PullRequestState = "MERGED"
	PullRequestStateDeclined   PullRequestState = "DECLINED"
	PullRequestStateSuperseded PullRequestState = "SUPERSEDED"
)

type Account struct {
	DisplayName string `json:"display_name,omitempty"`
	Nickname    string `json:"nickname,omitempty"`
	UUID        string `json:"uuid,omitempty"`
}

type PullRequestParticipant struct {
	User           *Account `json:"user,omitempty"`
	Role           string   `json:"role,omitempty"`
	Approved       bool     `json:"approved,omitempty"`
	State          string   `json:"state,omitempty"`
	ParticipatedOn string   `json:"participated_on,omitempty"`
}

type PullRequest struct {
	ID           int                      `json:"id"`
	Title        string                   `json:"title"`
	Description  string                   `json:"description,omitempty"`
	State        PullRequestState         `json:"state"`
	Source       PullRequestEndpoint      `json:"source"`
	Destination  PullRequestEndpoint      `json:"destination"`
	Author       Account                  `json:"author"`
	CreatedOn    string                   `json:"created_on"`
	UpdatedOn    string                   `json:"updated_on"`
	ClosedOn     string                   `json:"closed_on,omitempty"`
	MergeCommit  *CommitRef               `json:"merge_commit,omitempty"`
	CommentCount int                      `json:"comment_count,omitempty"`
	TaskCount    int                      `json:"task_count,omitempty"`
	Draft        bool                     `json:"draft,omitempty"`
	Queued       bool                     `json:"queued,omitempty"`
	Reviewers    []Account                `json:"reviewers,omitempty"`
	Participants []PullRequestParticipant `json:"participants,omitempty"`
	Links        PullRequestLinks         `json:"links"`
}

type PullRequestEndpoint struct {
	Branch     PullRequestBranch     `json:"branch"`
	Repository PullRequestRepository `json:"repository,omitempty"`
}

type PullRequestBranch struct {
	Name string `json:"name"`
}

type PullRequestRepository struct {
	FullName string `json:"full_name"`
}

type PullRequestLinks struct {
	HTML Link `json:"html"`
}

type Link struct {
	Href string `json:"href"`
}

type CommitRef struct {
	Hash string `json:"hash"`
}

type Branch struct {
	Name   string       `json:"name"`
	Target BranchTarget `json:"target"`
}

type BranchTarget struct {
	Hash   string             `json:"hash"`
	Date   string             `json:"date"`
	Author BranchTargetAuthor `json:"author"`
}

type BranchTargetAuthor struct {
	User BranchTargetUser `json:"user"`
}

type BranchTargetUser struct {
	DisplayName string `json:"display_name"`
}

type Pipeline struct {
	UUID              string         `json:"uuid"`
	BuildNumber       int            `json:"build_number"`
	State             PipelineState  `json:"state"`
	Target            PipelineTarget `json:"target"`
	CreatedOn         string         `json:"created_on"`
	DurationInSeconds *int           `json:"duration_in_seconds,omitempty"`
	Links             PipelineLinks  `json:"links"`
}

type PipelineState struct {
	Name   string               `json:"name"`
	Result *PipelineStateResult `json:"result,omitempty"`
	Stage  *PipelineStateStage  `json:"stage,omitempty"`
}

type PipelineStateResult struct {
	Name string `json:"name"`
}

type PipelineStateStage struct {
	Name string `json:"name"`
}

type PipelineTarget struct {
	RefName string `json:"ref_name,omitempty"`
	RefType string `json:"ref_type,omitempty"`
}

type PipelineLinks struct {
	Self Link `json:"self"`
}

type Commit struct {
	Hash    string        `json:"hash"`
	Message string        `json:"message,omitempty"`
	Date    string        `json:"date,omitempty"`
	Author  *CommitAuthor `json:"author,omitempty"`
	Links   *CommitLinks  `json:"links,omitempty"`
}

type CommitAuthor struct {
	Raw  string   `json:"raw,omitempty"`
	User *Account `json:"user,omitempty"`
}

type CommitLinks struct {
	HTML *Link `json:"html,omitempty"`
}

type PullRequestComment struct {
	ID        int                        `json:"id"`
	CreatedOn string                     `json:"created_on"`
	UpdatedOn string                     `json:"updated_on,omitempty"`
	User      *PullRequestCommentUser    `json:"user,omitempty"`
	Content   *PullRequestCommentContent `json:"content,omitempty"`
	Deleted   bool                       `json:"deleted,omitempty"`
}

type PullRequestCommentUser struct {
	DisplayName string `json:"display_name,omitempty"`
}

type PullRequestCommentContent struct {
	Raw    string `json:"raw,omitempty"`
	Markup string `json:"markup,omitempty"`
	HTML   string `json:"html,omitempty"`
}

type PullRequestTask struct {
	ID         int                     `json:"id"`
	State      string                  `json:"state,omitempty"`
	Content    *PullRequestTaskContent `json:"content,omitempty"`
	Creator    *Account                `json:"creator,omitempty"`
	Assignee   *Account                `json:"assignee,omitempty"`
	CreatedOn  string                  `json:"created_on,omitempty"`
	UpdatedOn  string                  `json:"updated_on,omitempty"`
	ResolvedOn string                  `json:"resolved_on,omitempty"`
}

type PullRequestTaskContent struct {
	Raw  string `json:"raw,omitempty"`
	HTML string `json:"html,omitempty"`
}

type CommitStatus struct {
	Key         string `json:"key"`
	Name        string `json:"name,omitempty"`
	State       string `json:"state,omitempty"`
	URL         string `json:"url,omitempty"`
	Description string `json:"description,omitempty"`
	UpdatedOn   string `json:"updated_on,omitempty"`
	CreatedOn   string `json:"created_on,omitempty"`
}

type CommitFile struct {
	Path        string `json:"path"`
	EscapedPath string `json:"escaped_path,omitempty"`
	Type        string `json:"type,omitempty"`
}

type PullRequestDiffstat struct {
	Type         string      `json:"type"`
	Status       string      `json:"status,omitempty"`
	LinesAdded   int         `json:"lines_added,omitempty"`
	LinesRemoved int         `json:"lines_removed,omitempty"`
	Old          *CommitFile `json:"old,omitempty"`
	New          *CommitFile `json:"new,omitempty"`
}

type PrMergePayload struct {
	MergeStrategy string `json:"merge_strategy,omitempty"`
	Message       string `json:"message,omitempty"`
}

type PrCreatePayload struct {
	Title       string                  `json:"title"`
	Source      PrCreatePayloadEndpoint `json:"source"`
	Destination PrCreatePayloadEndpoint `json:"destination"`
	Draft       bool                    `json:"draft,omitempty"`
	Description string                  `json:"description,omitempty"`
}

type PrCreatePayloadEndpoint struct {
	Branch PullRequestBranch `json:"branch"`
}

type PrUpdatePayload struct {
	Title       string                   `json:"title,omitempty"`
	Description string                   `json:"description,omitempty"`
	Destination *PrCreatePayloadEndpoint `json:"destination,omitempty"`
}
