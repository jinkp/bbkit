export interface PaginatedResponse<T> {
  values: T[]
  next?: string
  size?: number
  page?: number
  pagelen?: number
}

export interface BitbucketRepository {
  slug: string
  name: string
  description?: string
  language?: string
  updated_on: string
  full_name: string
  is_private: boolean
}

export interface BitbucketAccount {
  display_name?: string
  nickname?: string
  uuid?: string
}

export type BitbucketPullRequestState = 'OPEN' | 'MERGED' | 'DECLINED' | 'SUPERSEDED'

export interface BitbucketPullRequestParticipant {
  user?: BitbucketAccount
  role?: string
  approved?: boolean
  state?: string
  participated_on?: string
}

export interface BitbucketPullRequest {
  id: number
  title: string
  description?: string
  state: BitbucketPullRequestState
  source: { branch: { name: string }; repository: { full_name: string } }
  destination: { branch: { name: string }; repository?: { full_name: string } }
  author: BitbucketAccount
  created_on: string
  updated_on: string
  closed_on?: string
  merge_commit?: { hash: string }
  comment_count?: number
  task_count?: number
  draft?: boolean
  queued?: boolean
  reviewers?: BitbucketAccount[]
  participants?: BitbucketPullRequestParticipant[]
  links: { html: { href: string } }
}

export interface BitbucketCommit {
  hash: string
  message?: string
  date?: string
  author?: { raw?: string; user?: BitbucketAccount }
  links?: { html?: { href: string } }
}

export interface BitbucketPullRequestComment {
  id: number
  created_on: string
  updated_on?: string
  user?: { display_name?: string }
  content?: {
    raw?: string
    markup?: string
    html?: string
  }
  deleted?: boolean
}

export interface BitbucketPullRequestTask {
  id: number
  state?: 'OPEN' | 'RESOLVED' | string
  content?: {
    raw?: string
    html?: string
  }
  creator?: BitbucketAccount
  assignee?: BitbucketAccount
  created_on?: string
  updated_on?: string
  resolved_on?: string
}

export interface BitbucketCommitStatus {
  key: string
  name?: string
  state?: string
  url?: string
  description?: string
  updated_on?: string
  created_on?: string
}

export interface BitbucketCommitFile {
  path: string
  escaped_path?: string
  type?: string
}

export interface BitbucketPullRequestDiffstat {
  type: 'diffstat'
  status?: string
  lines_added?: number
  lines_removed?: number
  old?: BitbucketCommitFile
  new?: BitbucketCommitFile
}

export interface BitbucketBranch {
  name: string
  target: {
    date: string
    hash: string
    author: { user?: { display_name: string } }
  }
}

export interface BitbucketWorkspace {
  slug: string
  name: string
  uuid?: string
  type?: string
}

export interface BitbucketPipeline {
  uuid: string
  build_number: number
  state: {
    name: string
    result?: { name: string }
    stage?: { name: string }
  }
  target: { ref_name?: string; ref_type?: string }
  created_on: string
  duration_in_seconds?: number
  links: { self: { href: string } }
}
