load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "build_status_reporter",
    srcs = ["build_status_reporter.go"],
    importpath = "github.com/buildbuddy-io/buildbuddy/server/build_event_protocol/build_status_reporter",
    visibility = ["//visibility:public"],
    deps = [
        "//proto:build_event_stream_go_proto",
        "//server/backends/github",
        "//server/build_event_protocol/accumulator",
        "//server/endpoint_urls/build_buddy_url",
        "//server/environment",
        "//server/interfaces",
        "//server/util/authutil",
        "//server/util/git",
        "//server/util/log",
        "//server/util/timeutil",
    ],
)

go_test(
    name = "build_status_reporter_test",
    size = "small",
    srcs = ["build_status_reporter_test.go"],
    deps = [
        ":build_status_reporter",
        "//proto:build_event_stream_go_proto",
        "//proto:invocation_go_proto",
        "//server/backends/github",
        "//server/interfaces",
        "//server/nullauth",
        "//server/testutil/testauth",
        "//server/testutil/testenv",
        "//server/util/status",
        "//server/util/testing/flags",
        "@com_github_rs_zerolog//:zerolog",
        "@com_github_rs_zerolog//log",
        "@com_github_stretchr_testify//require",
    ],
)
