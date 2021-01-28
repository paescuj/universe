{{define "table"}}
Name | Stargazers | Last Sighting | Composition | Rights
---- | ---------- | ------------- | ----------- | ------
{{range . -}}
[{{.GetName}}]({{.GetHTMLURL}}) | {{.GetStargazersCount}} | {{.GetUpdatedAt.Time.Format "Mon Jan 2 2006"}} | {{.GetLanguage}} | {{.GetLicense.GetName}}
{{end -}}
{{- end -}}

# Universe
**{{.Count}}** stars discovered so far

## Living Stars
{{- if .Active -}}
{{template "table" .Active -}}
{{- else -}}
Huh, must be quite dark in our universe.
{{- end}}

## Dead Stars
{{- if .Archived -}}
{{- template "table" .Archived -}}
{{- else}}
Luckily, there are no death stars in our universe.
{{- end -}}
