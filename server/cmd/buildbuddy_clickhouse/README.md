# BuildBuddy OSS server with ClickHouse statistics

`main.go` follows `//server/cmd/buildbuddy/main.go` with two intentional differences:

- It registers the MIT-licensed ClickHouse backend (`//server/util/clickhouse`) before
  starting services, so invocation statistics are written to ClickHouse
  (`olap_database.data_source`). The OLAP DSN is redacted from startup-failure diagnostics.
- It does not instantiate or start the outbound usage-telemetry client.

No Go package under `enterprise/` is imported. The `clickhouse` branch carries only this
directory and the workflow below on top of the upstream release tag it is based on.

## Publishing an image

`.github/workflows/image.yaml` builds `:buildbuddy_image` with `--config=release` (`-c opt`,
which embeds the frontend bundle; `--stamp` for the reported version) and pushes it to
`ghcr.io/daanschipper/buildbuddy:<tag>` with the workflow's `GITHUB_TOKEN`. The image runs as
uid 1000 and carries no config file; pass `--config_file` at runtime.

Tags are `<upstream tag>-clickhouse.<n>`, `<n>` counting builds of the same upstream release:

```sh
git tag v2.297.0-clickhouse.1
git push origin v2.297.0-clickhouse.1
```

The package must be made public once (GitHub → Packages → `buildbuddy` → Package settings →
Change visibility) for anonymous pulls.

## Upgrading to a new upstream release

```sh
git fetch https://github.com/buildbuddy-io/buildbuddy.git tag v2.XXX.0
git rebase --onto v2.XXX.0 v2.297.0 clickhouse
```

Re-check `main.go` against upstream's `server/cmd/buildbuddy/main.go` for new startup steps,
build locally, then tag `v2.XXX.0-clickhouse.1`.

## Building locally

```sh
bazelisk build --config=release --bes_backend= --bes_results_url= \
  //server/cmd/buildbuddy_clickhouse:buildbuddy //server/cmd/buildbuddy_clickhouse:buildbuddy_image.tar
docker load -i bazel-bin/server/cmd/buildbuddy_clickhouse/buildbuddy_image.tar
```

`--bes_backend=` disables the anonymous build-event upload to buildbuddy.io that
`.bazelrc` enables by default.
