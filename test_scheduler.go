package melodeenext
package main
































}	}		}			fmt.Printf("Task: nil\n")		} else {			fmt.Printf("Task Payload: %s\n", string(entry.Task.Payload()))			fmt.Printf("Task Type: %s\n", entry.Task.Type())		if entry.Task != nil {		fmt.Printf("Prev: %v\n", entry.Prev)		fmt.Printf("Next: %v\n", entry.Next)		fmt.Printf("Spec: %s\n", entry.Spec)		fmt.Printf("ID: %s\n", entry.ID)		fmt.Printf("\n=== Entry %d ===\n", i+1)	for i, entry := range entries {	fmt.Printf("Found %d scheduler entries:\n", len(entries))		}		log.Fatalf("Failed to get scheduler entries: %v", err)	if err != nil {	entries, err := inspector.SchedulerEntries()		defer inspector.Close()	inspector := asynq.NewInspector(asynq.RedisClientOpt{Addr: "localhost:6379"})func main() {)	"github.com/hibiken/asynq"		"log"	"fmt"import (