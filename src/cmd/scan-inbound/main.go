package main

import (
	"flag"
	"fmt"
	"os"

	"melodee/internal/scanner"
)

func main() {
	// Parse command line flags
	inboundPath := flag.String("path", "", "Path to inbound directory to scan")
	scanDBPath := flag.String("output", "/tmp", "Directory to store scan database")
	workers := flag.Int("workers", 4, "Number of worker goroutines")
	flag.Parse()

	if *inboundPath == "" {
		fmt.Println("Usage: scan-inbound -path <inbound-directory> [-output <scan-db-directory>] [-workers <num>]")
		flag.PrintDefaults()
		os.Exit(1)
	}

	// Check if inbound path exists
	if _, err := os.Stat(*inboundPath); os.IsNotExist(err) {
		fmt.Printf("Error: Inbound path does not exist: %s\n", *inboundPath)
		os.Exit(1)
	}

	// Create scan database
	fmt.Printf("Creating scan database in %s...\n", *scanDBPath)
	scanDB, err := scanner.NewScanDB(*scanDBPath)
	if err != nil {
		fmt.Printf("Error creating scan database: %v\n", err)
		os.Exit(1)
	}
	defer scanDB.Close()

	fmt.Printf("Scan ID: %s\n", scanDB.GetScanID())
	fmt.Printf("Database: %s\n", scanDB.GetPath())
	fmt.Println()

	// Create file scanner
	fileScanner := scanner.NewFileScanner(scanDB, *workers)

	// Scan the directory
	fmt.Printf("Scanning %s with %d workers...\n", *inboundPath, *workers)
	if err := fileScanner.ScanDirectory(*inboundPath); err != nil {
		fmt.Printf("Error scanning directory: %v\n", err)
		os.Exit(1)
	}

	// Compute album grouping
	fmt.Println("\nComputing album grouping...")
	if err := scanDB.ComputeAlbumGrouping(); err != nil {
		fmt.Printf("Error computing album grouping: %v\n", err)
		os.Exit(1)
	}

	// Get statistics
	stats, err := scanDB.GetStats()
	if err != nil {
		fmt.Printf("Error getting stats: %v\n", err)
		os.Exit(1)
	}

	// Print results
	fmt.Println("\n=== Scan Complete ===")
	fmt.Printf("Total files: %d\n", stats.TotalFiles)
	fmt.Printf("Valid files: %d\n", stats.ValidFiles)
	fmt.Printf("Invalid files: %d\n", stats.InvalidFiles)
	fmt.Printf("Albums found: %d\n", stats.AlbumsFound)
	fmt.Printf("Duration: %v\n", stats.Duration)
	fmt.Printf("Files/sec: %.2f\n", stats.FilesPerSecond)
	fmt.Println()

	// Get and display album groups
	fmt.Println("=== Album Groups ===")
	groups, err := scanDB.GetAlbumGroups()
	if err != nil {
		fmt.Printf("Error getting album groups: %v\n", err)
		os.Exit(1)
	}

	for i, group := range groups {
		if i >= 10 {
			fmt.Printf("\n... and %d more albums\n", len(groups)-10)
			break
		}
		fmt.Printf("%d. %s - %s (%d)\n", i+1, group.ArtistName, group.AlbumName, group.Year)
		fmt.Printf("   Tracks: %d, Size: %.2f MB\n", group.TrackCount, float64(group.TotalSize)/(1024*1024))
	}

	fmt.Printf("\nScan database saved to: %s\n", scanDB.GetPath())
}
