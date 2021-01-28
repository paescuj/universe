{{define "table"}}
Name | Stargazers | Last Sighting | Composition | Rights
---- | ---------- | ------------- | ----------- | ------
{{range . -}}
[{{.GetName}}]({{.GetHTMLURL}}) | {{.GetStargazersCount}} | {{.GetPushedAt.Time.Format "2006-01-02 15:04"}} | {{.GetLanguage}} | {{.GetLicense.GetName}}
{{end -}}
{{- end -}}

# Universe
**{{.Count}}** stars discovered so far

## Living Stars
{{- if .LivingStars -}}
{{template "table" .LivingStars -}}
{{- else -}}
Huh, must be quite dark in our universe.
{{- end}}

## Dead Stars
{{- if .DeadStars -}}
{{- template "table" .DeadStars -}}
{{- else}}
Luckily, there are no death stars in our universe.
{{- end -}}
