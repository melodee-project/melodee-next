package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"melodee/internal/processor"
	"melodee/internal/scanner"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	// Parse command line flags
	scanDB := flag.String("scan", "", "Path to scan database file")
	stagingRoot := flag.String("staging", "/melodee/staging", "Root directory for staging")
	workers := flag.Int("workers", 4, "Number of worker goroutines")
	rateLimit := flag.Int("rate-limit", 0, "File operations per second (0 = unlimited)")
	dryRun := flag.Bool("dry-run", false, "Preview changes without moving files")
	dbHost := flag.String("db-host", "", "PostgreSQL host (optional, for saving staging items)")
	dbPort := flag.Int("db-port", 5432, "PostgreSQL port")
	dbName := flag.String("db-name", "melodee", "PostgreSQL database name")
	dbUser := flag.String("db-user", "melodee_user", "PostgreSQL user")
	dbPass := flag.String("db-pass", "", "PostgreSQL password")
	flag.Parse()

	if *scanDB == "" {
		fmt.Println("Usage: process-scan -scan <scan-database> [-staging <path>] [-workers <num>]")
		flag.PrintDefaults()
		os.Exit(1)
	}

	// Check if scan database exists
	if _, err := os.Stat(*scanDB); os.IsNotExist(err) {
		fmt.Printf("Error: Scan database does not exist: %s\n", *scanDB)
		os.Exit(1)
	}

	// Open scan database
	fmt.Printf("Opening scan database: %s\n", *scanDB)
	db, err := scanner.OpenScanDB(*scanDB)
	if err != nil {
		fmt.Printf("Error opening scan database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	// Get scan statistics
	stats, err := db.GetStats()
	if err != nil {
		fmt.Printf("Error getting scan stats: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\n=== Scan Database Info ===\n")
	fmt.Printf("Scan ID: %s\n", db.GetScanID())
	fmt.Printf("Total files: %d\n", stats.TotalFiles)
	fmt.Printf("Valid files: %d\n", stats.ValidFiles)
	fmt.Printf("Albums found: %d\n", stats.AlbumsFound)
	fmt.Println()

	if *dryRun {
		fmt.Println("*** DRY RUN MODE - No files will be moved ***")
		fmt.Println()
	}

	// Connect to PostgreSQL if credentials provided
	var pgDB *gorm.DB
	var stagingRepo *processor.StagingRepository
	if *dbHost != "" && *dbPass != "" {
		dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
			*dbHost, *dbPort, *dbUser, *dbPass, *dbName)
		
		pgDB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
		if err != nil {
			fmt.Printf("Warning: Could not connect to PostgreSQL: %v\n", err)
			fmt.Println("Continuing without database integration...")
		} else {
			fmt.Println("Connected to PostgreSQL database")
			stagingRepo = processor.NewStagingRepository(pgDB)
		}
		fmt.Println()
	}

	// Create processor
	config := &processor.ProcessorConfig{
		StagingRoot: *stagingRoot,
		Workers:     *workers,
		RateLimit:   *rateLimit,
		DryRun:      *dryRun,
	}

	proc := processor.NewProcessor(config, db)

	// Process all albums
	fmt.Printf("Processing albums to staging (%s)...\n", *stagingRoot)
	fmt.Printf("Workers: %d\n", *workers)
	if *rateLimit > 0 {
		fmt.Printf("Rate limit: %d files/sec\n", *rateLimit)
	}
	fmt.Println()

	startTime := time.Now()
	results, err := proc.ProcessAllAlbums()
	if err != nil {
		fmt.Printf("Error processing albums: %v\n", err)
		os.Exit(1)
	}
	duration := time.Since(startTime)

	// Save to database if connected
	if stagingRepo != nil && !*dryRun {
		fmt.Println("Saving staging items to database...")
		saved := 0
		for _, result := range results {
			if result.Success {
				// Read metadata file
				metadata, err := processor.ReadAlbumMetadata(result.MetadataFile)
				if err != nil {
					fmt.Printf("Warning: Could not read metadata for %s: %v\n", result.StagingPath, err)
					continue
				}

				// Create staging item
				if err := stagingRepo.CreateStagingItemFromResult(result, metadata); err != nil {
					fmt.Printf("Warning: Could not save staging item for %s: %v\n", result.StagingPath, err)
					continue
				}
				saved++
			}
		}
		fmt.Printf("Saved %d staging items to database\n\n", saved)
	}

	// Calculate statistics
	procStats := processor.GetProcessStats(results, duration)

	// Print results
	fmt.Println("=== Processing Complete ===")
	fmt.Printf("Duration: %v\n", duration)
	fmt.Printf("Total albums: %d\n", procStats.TotalAlbums)
	fmt.Printf("Successful: %d\n", procStats.SuccessAlbums)
	fmt.Printf("Failed: %d\n", procStats.FailedAlbums)
	fmt.Printf("Total tracks: %d\n", procStats.TotalTracks)
	fmt.Printf("Total size: %.2f MB\n", float64(procStats.TotalSize)/(1024*1024))
	fmt.Println()

	// Show successful albums
	if procStats.SuccessAlbums > 0 {
		fmt.Println("=== Staged Albums ===")
		count := 0
		for _, result := range results {
			if result.Success {
				count++
				if count <= 10 {
					fmt.Printf("%d. %s\n", count, result.StagingPath)
					fmt.Printf("   Tracks: %d, Size: %.2f MB\n", 
						result.TrackCount, float64(result.TotalSize)/(1024*1024))
				}
			}
		}
		if count > 10 {
			fmt.Printf("\n... and %d more albums\n", count-10)
		}
		fmt.Println()
	}

	// Show failed albums
	if procStats.FailedAlbums > 0 {
		fmt.Println("=== Failed Albums ===")
		count := 0
		for _, result := range results {
			if !result.Success {
				count++
				fmt.Printf("%d. Album Group: %s\n", count, result.AlbumGroupID)
				fmt.Printf("   Error: %v\n", result.Error)
			}
		}
		fmt.Println()
	}

	if *dryRun {
		fmt.Println("*** This was a DRY RUN - no files were actually moved ***")
	} else {
		fmt.Printf("Staging directory: %s\n", *stagingRoot)
	}
}
