# Spec: Markdown Output

## ADDED Requirements

### Requirement: Per-post file

For each collected post, the tool MUST write a Markdown file at:
`<output-dir>/<username>/<YYYYMMDD>-<post-id>.md`

The file MUST contain:
- The publication timestamp (ISO 8601) as a top-level metadata line
- The post body text
- For reposts: the reposting user's comment (if any) followed by a Markdown
  blockquote containing the original author's handle and text

### Requirement: Index file

The tool MUST write (or overwrite) an `index.md` in `<output-dir>/<username>/`
listing all posts that were written in the current run, each with:
- The first line of the post body as the title
- The publication timestamp

### Requirement: Overwrite behavior

If a file with the same `<YYYYMMDD>-<post-id>.md` name already exists, the tool
MUST overwrite it without error or warning.

### Requirement: Text-only filtering

The tool MUST skip posts that contain no text content. The `--limit` flag counts
only posts that pass this filter.
