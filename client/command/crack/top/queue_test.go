package top

import (
	"testing"

	"github.com/bishopfox/sliver/protobuf/clientpb"
)

type crackTopQueueCounts struct {
	queued  int
	leased  int
	running int
}

func TestBuildCrackTopDashboardCountsActiveJobTaskQueueStates(t *testing.T) {
	crackTasks := func(states ...clientpb.CrackTaskState) []*clientpb.CrackTask {
		tasks := make([]*clientpb.CrackTask, 0, len(states))
		for _, state := range states {
			tasks = append(tasks, &clientpb.CrackTask{
				Kind:  clientpb.CrackTaskKind_CRACK_TASK_CRACK,
				State: state,
			})
		}
		return tasks
	}

	testCases := []struct {
		name string
		jobs []*clientpb.CrackJob
		want crackTopQueueCounts
	}{
		{
			name: "in-progress job",
			jobs: []*clientpb.CrackJob{{
				ID:     "in-progress",
				Status: clientpb.CrackJobStatus_IN_PROGRESS,
				Tasks: crackTasks(
					clientpb.CrackTaskState_CRACK_TASK_QUEUED,
					clientpb.CrackTaskState_CRACK_TASK_LEASED,
					clientpb.CrackTaskState_CRACK_TASK_RUNNING,
				),
			}},
			want: crackTopQueueCounts{queued: 1, leased: 1, running: 1},
		},
		{
			name: "paused job",
			jobs: []*clientpb.CrackJob{{
				ID:     "paused",
				Status: clientpb.CrackJobStatus_PAUSED,
				Tasks: crackTasks(
					clientpb.CrackTaskState_CRACK_TASK_QUEUED,
					clientpb.CrackTaskState_CRACK_TASK_LEASED,
					clientpb.CrackTaskState_CRACK_TASK_RUNNING,
				),
			}},
			want: crackTopQueueCounts{queued: 1, leased: 1, running: 1},
		},
		{
			name: "legacy unspecified crack task with shard telemetry",
			jobs: []*clientpb.CrackJob{{
				ID:     "legacy-crack",
				Status: clientpb.CrackJobStatus_IN_PROGRESS,
				Tasks: []*clientpb.CrackTask{{
					Kind:       clientpb.CrackTaskKind_CRACK_TASK_UNSPECIFIED,
					State:      clientpb.CrackTaskState_CRACK_TASK_RUNNING,
					ShardLimit: 100,
				}},
			}},
			want: crackTopQueueCounts{running: 1},
		},
		{
			name: "keyspace and auxiliary scheduler stages",
			jobs: []*clientpb.CrackJob{{
				ID:     "queries",
				Status: clientpb.CrackJobStatus_IN_PROGRESS,
				Tasks: []*clientpb.CrackTask{
					{Kind: clientpb.CrackTaskKind_CRACK_TASK_UNSPECIFIED, State: clientpb.CrackTaskState_CRACK_TASK_QUEUED},
					{Kind: clientpb.CrackTaskKind_CRACK_TASK_KEYSPACE, State: clientpb.CrackTaskState_CRACK_TASK_QUEUED},
					{Kind: clientpb.CrackTaskKind_CRACK_TASK_BENCHMARK, State: clientpb.CrackTaskState_CRACK_TASK_LEASED},
					{Kind: clientpb.CrackTaskKind_CRACK_TASK_QUERY, State: clientpb.CrackTaskState_CRACK_TASK_RUNNING},
				},
			}},
			want: crackTopQueueCounts{queued: 2, leased: 1, running: 1},
		},
		{
			name: "terminal jobs",
			jobs: []*clientpb.CrackJob{
				{ID: "completed", Status: clientpb.CrackJobStatus_COMPLETED, Tasks: crackTasks(clientpb.CrackTaskState_CRACK_TASK_QUEUED)},
				{ID: "failed", Status: clientpb.CrackJobStatus_FAILED, Tasks: crackTasks(clientpb.CrackTaskState_CRACK_TASK_LEASED)},
				{ID: "cancelled", Status: clientpb.CrackJobStatus_CANCELLED, Tasks: crackTasks(clientpb.CrackTaskState_CRACK_TASK_RUNNING)},
			},
		},
		{
			name: "terminal task states",
			jobs: []*clientpb.CrackJob{{
				ID:     "active-with-terminal-tasks",
				Status: clientpb.CrackJobStatus_IN_PROGRESS,
				Tasks: append(
					crackTasks(
						clientpb.CrackTaskState_CRACK_TASK_COMPLETED,
						clientpb.CrackTaskState_CRACK_TASK_FAILED,
						clientpb.CrackTaskState_CRACK_TASK_CANCELLED,
					),
					nil,
				),
			}},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			dashboard := buildCrackTopDashboard(&crackTopSnapshot{Jobs: testCase.jobs})
			got := crackTopQueueCounts{
				queued:  dashboard.QueuedTasks,
				leased:  dashboard.LeasedTasks,
				running: dashboard.RunningTasks,
			}
			if got != testCase.want {
				t.Fatalf("task queue counts = %+v, want %+v", got, testCase.want)
			}
		})
	}
}
