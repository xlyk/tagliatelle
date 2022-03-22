# tagliatelle

Utility to update tags and version numbers in project files. Useful for config files, kustomize projects, and helm
charts.

.env file (required)
---

```shell
GIT_USER=your-username
GIT_TOKEN=your-personal-access-token
```

Options
---

| Option   | Required | Description                                                        |
|----------|----------|--------------------------------------------------------------------|
| -repo    | yes      | repo that contains target file (access and permissions required)   |
| -file    | yes      | target file to update                                              |
| -tag     | yes      | string to use as replacement                                       |
| -pattern | yes      | regex pattern to use                                               |
| -dry-run | no       | (optional) don't actually write changes to file or create a commit |

Example
---

```shell
tagliatelle \
  -repo "https://github.com/xlyk/tagliatelle.git" \
  -file "sample.yaml" \
  -tag "1.0.0.2" \
  -pattern '(app1"\n.*newTag: ")(.*?)(")' \
  -dry-run
```

Docker example
---

```shell
docker run --rm -it --env-file ./.env xlyk/tagliatelle:latest \
  -repo "https://github.com/xlyk/tagliatelle.git" \
  -file "sample.yaml" \
  -tag "1.0.0.2" \
  -pattern '(app1"\n.*newTag: ")(.*?)(")' \
  -dry-run
```
