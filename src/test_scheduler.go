package main

import (
"fmt"
"log"

"github.com/hibiken/asynq"
)

func main() {
inspector := asynq.NewInspector(asynq.RedisClientOpt{Addr: "localhost:6379"})
defer inspector.Close()

entries, err := inspector.SchedulerEntries()
if err != nil {
log.Fatalf("Failed to get scheduler entries: %v", err)
}

fmt.Printf("Found %d scheduler entries:\n", len(entries))
for i, entry := range entries {
fmt.Printf("\n=== Entry %d ===\n", i+1)
fmt.Printf("ID: %s\n", entry.ID)
fmt.Printf("Spec: %s\n", entry.Spec)
fmt.Printf("Next: %v\n", entry.Next)
fmt.Printf("Prev: %v\n", entry.Prev)
if entry.Task != nil {
fmt.Printf("Task Type: %s\n", entry.Task.Type())
fmt.Printf("Task Payload: %s\n", string(entry.Task.Payload()))
} else {
fmt.Printf("Task: nil\n")
}
}
}
