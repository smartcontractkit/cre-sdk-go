// TODO go install the protos tool from the right hash...?
//
//go:generate mkdir -p .tools
//go:generate sh -c "(cd ../../../generator/protos && go build .) && mv ../../../generator/protos/protos ./.tools/protos"
//go:generate ./.tools/protos --category=scheduler --pkg=cron --major-version=1 --files=capabilities/scheduler/cron/v1/trigger.proto
package cron
