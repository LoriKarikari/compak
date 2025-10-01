package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/LoriKarikari/compak/internal/core/index"
)

var searchLimit int

func init() {
	searchCmd.Flags().IntVar(&searchLimit, "limit", 10, "Maximum number of results to show")
	rootCmd.AddCommand(searchCmd)
}

var searchCmd = &cobra.Command{
	Use:   "search [QUERY]",
	Short: "Search for compak paks",
	Long: `Search for compak paks in the pak index.

Examples:
  compak search nginx
  compak search postgres --limit 20
  compak search blog
  compak search database`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		query := ""
		if len(args) > 0 {
			query = args[0]
		}
		return searchPackages(query)
	},
}

func searchPackages(query string) error {
	if query != "" {
		fmt.Printf("Searching for paks matching '%s'...\n\n", query)
	} else {
		fmt.Printf("Listing available paks...\n\n")
	}

	client := index.NewClient()
	ctx := context.Background()

	results, err := client.Search(ctx, query, searchLimit)
	if err != nil {
		return fmt.Errorf("search failed: %w", err)
	}

	if len(results) == 0 {
		if query != "" {
			fmt.Printf("No paks found matching '%s'.\n", query)
		} else {
			fmt.Println("No paks available in the index.")
		}
		fmt.Println("\nTo add a pak to the index, see:")
		fmt.Println("  https://github.com/LoriKarikari/compak")
		return nil
	}

	fmt.Printf("Found %d pak(s):\n\n", len(results))

	for _, result := range results {
		displaySearchResult(result)
	}

	fmt.Printf("\nTo install a pak, run:\n")
	fmt.Printf("  compak install PAK_NAME\n")

	return nil
}

func displaySearchResult(result index.SearchResult) {
	fmt.Printf("[%s] v%s\n", result.Name, result.Version)

	if result.Description != "" {
		fmt.Printf("   %s\n", result.Description)
	}

	if result.Author != "" {
		fmt.Printf("   Author: %s\n", result.Author)
	}

	if result.Homepage != "" {
		fmt.Printf("   Homepage: %s\n", result.Homepage)
	}

	fmt.Println()
}
