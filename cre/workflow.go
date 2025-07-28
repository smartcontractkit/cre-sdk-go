package cre

type Workflow[C any] []ExecutionHandler[C, Runtime]
