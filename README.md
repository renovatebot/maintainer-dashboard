# Renovate maintainer dashboard

A set of tooling to produce a local-first view of the [Renovate](https://docs.renovatebot.com/) project's Discussions.

As Renovate [uses GitHub Discussions for triage process](https://www.jvt.me/posts/2026/01/07/renovate-why-discussions/), this provides [a better view of the GitHub Discussions interface](https://www.jvt.me/posts/2026/02/20/renovate-discussions-data/).

If present, this also includes the [release train stats](https://github.com/renovatebot/release-train-stats) for the Renovate project.

## Components

### SQLite database

As its heart, this project produces an SQLite database that provides access to all the data that exists in GitHub Discussions.

This can then be presented and/or searched in ways independent to the GitHub Web UI and APIs, and provides a fully offline interaction model after-the-fact.

### Evidence.dev dashboard

A visualisation layer for the SQLite database, providing a mix of tabular and more appropriate visualisations of the common data points in the project.

This uses the great data visualisation toolkit [Evidence.dev](https://evidence.dev).

### (Future) Web Application

In the future, the intent is that this will have a web application that is available to folks who have Triage+ rights on the Renovate project, where this will be centrally deployed and kept updated.

## Getting started

### GitHub App

You will need a GitHub App to sync the data.

This app does not need access to private repositories.

As we are accessing public repositories, the basic permissions are sufficient.

Ideally, the GitHub App would have multiple installations (i.e. installed onto multiple organizations) to provide a higher API rate limit, which will be automagically be used when syncing/backfilling.

### Initialise database

```console
# as a one-time thing, or when the schema changes
go run ./cmd/init -db new.db
```

### Sync data

```console
# as it's running for the first time, this will take some time
go run ./cmd/sync -db new.db -app-id=$GITHUB_APP_ID -app-key=$GITHUB_APP_KEY_PEM_PATH
```

### Backfill data

> [!NOTE]
> This is only needed when new fields are added to the database, and is **not** needed on first sync.

```console
go run ./cmd/backfill -db new.db -app-id=$GITHUB_APP_ID -app-key=$GITHUB_APP_KEY_PEM_PATH
```

## License

This code is licensed as Renovate is, under the `AGPL-3.0-only`.

This project includes contributions that include code derived from Large Language Models.
