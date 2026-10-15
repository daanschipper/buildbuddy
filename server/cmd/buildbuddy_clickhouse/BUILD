load("@io_bazel_rules_docker//container:container.bzl", "container_image")
load("@io_bazel_rules_docker//container:push.bzl", "container_push")
load("@io_bazel_rules_docker//go:image.bzl", "go_image")
load("@io_bazel_rules_go//go:def.bzl", "go_binary", "go_library")

# The OSS server (//server/cmd/buildbuddy) with the ClickHouse statistics backend
# registered and without the usage-telemetry client. See README.md.
go_binary(
    name = "buildbuddy",
    embed = [":buildbuddy_clickhouse_lib"],
    visibility = ["//visibility:public"],
)

go_image(
    name = "buildbuddy_go_image",
    base = select({
        "@platforms//cpu:arm64": "@buildbuddy_go_image_base_arm64//image",
        "//conditions:default": "@buildbuddy_go_image_base//image",
    }),
    binary = ":buildbuddy",
    tags = ["manual"],
)

# The deployment mounts its config and data volumes, so the image carries no config file.
container_image(
    name = "buildbuddy_image",
    base = ":buildbuddy_go_image",
    tags = ["manual"],
    user = "1000",
    visibility = ["//visibility:public"],
)

# Pushed by .github/workflows/image.yaml on every v*-clickhouse.* tag.
container_push(
    name = "push",
    format = "Docker",
    image = ":buildbuddy_image",
    registry = "ghcr.io",
    repository = "daanschipper/buildbuddy",
    tag_file = "//deployment:image_tag_file",
    tags = ["manual"],
)

go_library(
    name = "buildbuddy_clickhouse_lib",
    srcs = ["main.go"],
    importpath = "github.com/buildbuddy-io/buildbuddy/server/cmd/buildbuddy_clickhouse",
    visibility = [":__subpackages__"],
    deps = [
        "//app:bundle",
        "//server/config",
        "//server/janitor",
        "//server/libmain",
        "//server/remote_cache/capabilities_server",
        "//server/util/clickhouse",
        "//server/util/grpc_server",
        "//server/util/healthcheck",
        "//server/util/log",
        "//server/util/redact",
        "//server/version",
        "@org_golang_google_grpc//xds",
    ],
)
