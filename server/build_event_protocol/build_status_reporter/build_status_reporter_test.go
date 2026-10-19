package build_status_reporter_test

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/buildbuddy-io/buildbuddy/proto/build_event_stream"
	"github.com/buildbuddy-io/buildbuddy/server/backends/github"
	"github.com/buildbuddy-io/buildbuddy/server/build_event_protocol/build_status_reporter"
	"github.com/buildbuddy-io/buildbuddy/server/interfaces"
	"github.com/buildbuddy-io/buildbuddy/server/nullauth"
	"github.com/buildbuddy-io/buildbuddy/server/testutil/testauth"
	"github.com/buildbuddy-io/buildbuddy/server/testutil/testenv"
	"github.com/buildbuddy-io/buildbuddy/server/util/status"
	"github.com/buildbuddy-io/buildbuddy/server/util/testing/flags"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	inpb "github.com/buildbuddy-io/buildbuddy/proto/invocation"
	zllog "github.com/rs/zerolog/log"
)

const (
	authWarning   = "Failed to get authenticated user"
	anonymousSkip = "Not reporting GitHub status for anonymous invocation"
)

type fakeAccumulator struct {
	invocation *inpb.Invocation
}

func (a *fakeAccumulator) Invocation() *inpb.Invocation       { return a.invocation }
func (a *fakeAccumulator) MetadataIsLoaded() bool             { return true }
func (a *fakeAccumulator) StartTime() time.Time               { return time.Time{} }
func (a *fakeAccumulator) DisableCommitStatusReporting() bool { return false }
func (a *fakeAccumulator) DisableTargetTracking() bool        { return false }
func (a *fakeAccumulator) WorkflowID() string                 { return "" }
func (a *fakeAccumulator) ActionName() string                 { return "" }
func (a *fakeAccumulator) CommitStatusLabel() string          { return "" }
func (a *fakeAccumulator) Pattern() string                    { return "//..." }
func (a *fakeAccumulator) BuildFinished() bool                { return false }

type createdStatus struct {
	groupID   string
	ownerRepo string
	commitSHA string
	payload   *github.GithubStatusPayload
}

type fakeStatusClient struct {
	enabledChecks int
	statuses      []createdStatus
}

func (c *fakeStatusClient) CreateStatus(ctx context.Context, groupID, ownerRepo, commitSHA string, payload *github.GithubStatusPayload) error {
	c.statuses = append(c.statuses, createdStatus{groupID, ownerRepo, commitSHA, payload})
	return nil
}

func (c *fakeStatusClient) IsStatusReportingEnabled(ctx context.Context, groupID, repoURL string) (bool, error) {
	c.enabledChecks++
	return true, nil
}

type fakeStatusService struct {
	client *fakeStatusClient
}

func (s *fakeStatusService) GetStatusClient() interfaces.GitHubStatusClient { return s.client }

// failingAuthenticator simulates a non-anonymous auth failure (e.g. bad credentials).
type failingAuthenticator struct {
	*nullauth.NullAuthenticator
}

func (a *failingAuthenticator) AuthenticatedUser(ctx context.Context) (interfaces.UserInfo, error) {
	return nil, status.UnauthenticatedError("invalid API key")
}

func captureLogs(t *testing.T) *bytes.Buffer {
	buf := &bytes.Buffer{}
	prev := zllog.Logger
	zllog.Logger = zerolog.New(buf).Level(zerolog.DebugLevel)
	t.Cleanup(func() { zllog.Logger = prev })
	return buf
}

func finishedEvent() *build_event_stream.BuildEvent {
	return &build_event_stream.BuildEvent{
		Id: &build_event_stream.BuildEventId{Id: &build_event_stream.BuildEventId_BuildFinished{}},
		Payload: &build_event_stream.BuildEvent_Finished{Finished: &build_event_stream.BuildFinished{
			ExitCode: &build_event_stream.BuildFinished_ExitCode{Name: "OK", Code: 0},
		}},
	}
}

func TestReportStatusForEvent_AuthHandling(t *testing.T) {
	for _, test := range []struct {
		name          string
		accessToken   string
		authenticator func(t *testing.T) (interfaces.Authenticator, context.Context)
		wantStatuses  int
		wantGroupID   string
		wantWarning   bool
		wantSkipDebug bool
	}{
		{
			name: "anonymous user with anonymous usage enabled skips silently",
			authenticator: func(t *testing.T) (interfaces.Authenticator, context.Context) {
				return nullauth.NewNullAuthenticator(true), context.Background()
			},
			wantSkipDebug: true,
		},
		{
			name: "anonymous user with anonymous usage disabled warns",
			authenticator: func(t *testing.T) (interfaces.Authenticator, context.Context) {
				return nullauth.NewNullAuthenticator(false), context.Background()
			},
			wantWarning: true,
		},
		{
			name: "non-anonymous auth failure warns",
			authenticator: func(t *testing.T) (interfaces.Authenticator, context.Context) {
				return &failingAuthenticator{nullauth.NewNullAuthenticator(true)}, context.Background()
			},
			wantWarning: true,
		},
		{
			name:        "anonymous user with access token reports without group",
			accessToken: "ghp_test",
			authenticator: func(t *testing.T) (interfaces.Authenticator, context.Context) {
				return nullauth.NewNullAuthenticator(true), context.Background()
			},
			wantStatuses: 1,
		},
		{
			name: "authenticated user reports with group",
			authenticator: func(t *testing.T) (interfaces.Authenticator, context.Context) {
				auth := testauth.NewTestAuthenticator(t, testauth.TestUsers("USER1", "GROUP1"))
				ctx, err := auth.WithAuthenticatedUser(context.Background(), "USER1")
				require.NoError(t, err)
				return auth, ctx
			},
			wantStatuses: 1,
			wantGroupID:  "GROUP1",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			flags.Set(t, "github.access_token", test.accessToken)

			te := testenv.GetTestEnv(t)
			auth, ctx := test.authenticator(t)
			te.SetAuthenticator(auth)
			client := &fakeStatusClient{}
			te.SetGitHubStatusService(&fakeStatusService{client: client})

			acc := &fakeAccumulator{invocation: &inpb.Invocation{
				InvocationId: "11111111-2222-3333-4444-555555555555",
				Role:         "CI",
				Command:      "build",
				RepoUrl:      "https://github.com/testowner/testrepo.git",
				CommitSha:    "0c894fe31c2e91d59cb1a59bb25aaa78089919c2",
			}}
			reporter := build_status_reporter.NewBuildStatusReporter(te, acc)
			// Capture only the reporter's output; test env setup logs unrelated warnings.
			logs := captureLogs(t)
			reporter.ReportStatusForEvent(ctx, finishedEvent())

			require.Len(t, client.statuses, test.wantStatuses)
			if test.wantStatuses > 0 {
				require.Equal(t, 1, client.enabledChecks)
				require.Equal(t, test.wantGroupID, client.statuses[0].groupID)
				require.Equal(t, "testowner/testrepo", client.statuses[0].ownerRepo)
				require.Equal(t, "success", client.statuses[0].payload.GetState())
			} else {
				require.Equal(t, 0, client.enabledChecks)
			}

			out := logs.String()
			require.Equal(t, test.wantWarning, bytes.Contains([]byte(out), []byte(authWarning)), out)
			require.Equal(t, test.wantSkipDebug, bytes.Contains([]byte(out), []byte(anonymousSkip)), out)
			if !test.wantWarning {
				require.NotContains(t, out, `"level":"warn"`)
			}
		})
	}
}
